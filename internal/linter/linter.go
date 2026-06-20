package linter

import (
	"regexp"
	"strings"
)

// LintWarning describes one dangerous or risky DDL pattern found in migration SQL.
type LintWarning struct {
	Pattern    string
	Line       int
	Severity   string
	Message    string
	Suggestion string
}

type lintRule struct {
	pattern    string
	severity   string
	message    string
	suggestion string
	expression *regexp.Regexp
}

var lintRules = []lintRule{
	{
		pattern:    "DROP_COLUMN",
		severity:   "error",
		message:    "Dropping a column can permanently destroy data and break older application versions.",
		suggestion: "Prefer a multi-step migration: stop writes, backfill/read from the replacement, then drop in a later release.",
		expression: regexp.MustCompile(`(?is)\bALTER\s+TABLE\b[^;]*\bDROP\s+COLUMN\b`),
	},
	{
		pattern:    "RENAME_COLUMN",
		severity:   "error",
		message:    "Renaming a column is not backward-compatible for rolling deploys.",
		suggestion: "Add the new column, dual-write/backfill, switch reads, then remove the old column in a later migration.",
		expression: regexp.MustCompile(`(?is)\bALTER\s+TABLE\b[^;]*\bRENAME\s+COLUMN\b`),
	},
	{
		pattern:    "SET_NOT_NULL_WITHOUT_DEFAULT",
		severity:   "error",
		message:    "Adding NOT NULL without a DEFAULT can fail on existing rows or take disruptive locks.",
		suggestion: "Add a DEFAULT or use a staged migration: add nullable column, backfill, validate, then set NOT NULL.",
		expression: regexp.MustCompile(`(?is)\bALTER\s+TABLE\b[^;]*\bALTER\s+COLUMN\b[^;]*\bSET\s+NOT\s+NULL\b`),
	},
	{
		pattern:    "DROP_TABLE",
		severity:   "error",
		message:    "Dropping a table can permanently destroy data and break running application versions.",
		suggestion: "Archive or rename first, remove application references, then drop in a later reviewed migration.",
		expression: regexp.MustCompile(`(?is)\bDROP\s+TABLE\b`),
	},
	{
		pattern:    "CREATE_INDEX_WITHOUT_CONCURRENTLY",
		severity:   "warning",
		message:    "Creating an index without CONCURRENTLY can block writes on large tables.",
		suggestion: "Use CREATE INDEX CONCURRENTLY outside a transaction when PostgreSQL supports it.",
		expression: regexp.MustCompile(`(?is)\bCREATE\s+(?:UNIQUE\s+)?INDEX\b`),
	},
	{
		pattern:    "FOREIGN_KEY_WITHOUT_NOT_VALID",
		severity:   "warning",
		message:    "Adding a foreign key without NOT VALID can scan existing rows and take heavier locks.",
		suggestion: "Use ADD CONSTRAINT ... FOREIGN KEY ... NOT VALID, then VALIDATE CONSTRAINT separately.",
		expression: regexp.MustCompile(`(?is)\bADD\s+CONSTRAINT\b[^;]*\bFOREIGN\s+KEY\b`),
	},
}

// LintSQL returns zero-downtime migration warnings for dangerous DDL patterns.
func LintSQL(sql string) []LintWarning {
	warnings := make([]LintWarning, 0)
	statements := splitStatements(sql)
	for _, statement := range statements {
		text := strings.TrimSpace(statement.text)
		if text == "" {
			continue
		}
		for _, rule := range lintRules {
			if !rule.expression.MatchString(text) {
				continue
			}
			if rule.pattern == "SET_NOT_NULL_WITHOUT_DEFAULT" && strings.Contains(strings.ToUpper(text), "DEFAULT") {
				continue
			}
			if rule.pattern == "CREATE_INDEX_WITHOUT_CONCURRENTLY" && strings.Contains(strings.ToUpper(text), "CONCURRENTLY") {
				continue
			}
			if rule.pattern == "FOREIGN_KEY_WITHOUT_NOT_VALID" && strings.Contains(strings.ToUpper(text), "NOT VALID") {
				continue
			}
			warnings = append(warnings, LintWarning{
				Pattern:    rule.pattern,
				Line:       statement.line,
				Severity:   rule.severity,
				Message:    rule.message,
				Suggestion: rule.suggestion,
			})
		}
	}
	return warnings
}

type sqlStatement struct {
	text string
	line int
}

func splitStatements(sql string) []sqlStatement {
	statements := make([]sqlStatement, 0)
	start := 0
	startLine := 1
	line := 1
	inSingleQuote := false
	inDoubleQuote := false

	for index, char := range sql {
		switch char {
		case '\n':
			line++
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote {
				statements = append(statements, sqlStatement{text: sql[start:index], line: firstContentLine(sql[start:index], startLine)})
				start = index + 1
				startLine = line
			}
		}
	}
	if strings.TrimSpace(sql[start:]) != "" {
		statements = append(statements, sqlStatement{text: sql[start:], line: firstContentLine(sql[start:], startLine)})
	}
	return statements
}

func firstContentLine(statement string, startLine int) int {
	line := startLine
	for _, char := range statement {
		switch char {
		case '\n':
			line++
		case ' ', '	', '\r':
			continue
		default:
			return line
		}
	}
	return startLine
}
