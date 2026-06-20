package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

// HasChanges reports whether a schema diff contains any structural change.
func HasChanges(schemaDiff *SchemaDiff) bool {
	if schemaDiff == nil {
		return false
	}
	return len(schemaDiff.TablesAdded) > 0 || len(schemaDiff.TablesDropped) > 0 || len(schemaDiff.TablesModified) > 0 || len(schemaDiff.IndexChanges) > 0
}

// Summary returns a compact human-readable change count summary.
func Summary(schemaDiff *SchemaDiff) string {
	if schemaDiff == nil || !HasChanges(schemaDiff) {
		return "no schema changes"
	}

	columnsAdded := 0
	columnsDropped := 0
	columnsModified := 0
	for _, modification := range schemaDiff.TablesModified {
		columnsAdded += len(modification.ColumnsAdded)
		columnsDropped += len(modification.ColumnsDropped)
		columnsModified += len(modification.ColumnsModified)
	}

	parts := make([]string, 0, 7)
	appendCount := func(count int, singular string, plural string) {
		if count == 0 {
			return
		}
		label := plural
		if count == 1 {
			label = singular
		}
		parts = append(parts, fmt.Sprintf("%d %s", count, label))
	}

	appendCount(len(schemaDiff.TablesModified), "table modified", "tables modified")
	appendCount(len(schemaDiff.TablesAdded), "table added", "tables added")
	appendCount(len(schemaDiff.TablesDropped), "table dropped", "tables dropped")
	appendCount(columnsAdded, "column added", "columns added")
	appendCount(columnsDropped, "column dropped", "columns dropped")
	appendCount(columnsModified, "column modified", "columns modified")
	appendCount(len(schemaDiff.IndexChanges), "index changed", "indexes changed")

	return strings.Join(parts, ", ")
}

// RenderTerminal writes a colored terminal representation of schema changes.
func RenderTerminal(writer io.Writer, schemaDiff *SchemaDiff) error {
	if schemaDiff == nil {
		schemaDiff = &SchemaDiff{}
	}

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	muted := color.New(color.FgHiBlack).SprintFunc()

	if !HasChanges(schemaDiff) {
		_, err := fmt.Fprintf(writer, "%s no schema changes\n", green("OK"))
		return err
	}

	if _, err := fmt.Fprintf(writer, "%s %s\n\n", yellow("DIFF"), Summary(schemaDiff)); err != nil {
		return err
	}

	if len(schemaDiff.TablesAdded) > 0 || len(schemaDiff.TablesDropped) > 0 {
		if _, err := fmt.Fprintln(writer, cyan("Tables")); err != nil {
			return err
		}
		for _, table := range schemaDiff.TablesAdded {
			if _, err := fmt.Fprintf(writer, "  %s table %s\n", green("+"), table.Name); err != nil {
				return err
			}
		}
		for _, table := range schemaDiff.TablesDropped {
			if _, err := fmt.Fprintf(writer, "  %s table %s\n", red("-"), table.Name); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}

	for _, modification := range schemaDiff.TablesModified {
		if _, err := fmt.Fprintf(writer, "%s %s\n", cyan("Table"), modification.TableName); err != nil {
			return err
		}

		sectionWriter := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
		for _, column := range modification.ColumnsAdded {
			if _, err := fmt.Fprintf(sectionWriter, "  %s column\t%s\t%s\n", green("+"), column.Name, describeColumn(column)); err != nil {
				return err
			}
		}
		for _, column := range modification.ColumnsDropped {
			if _, err := fmt.Fprintf(sectionWriter, "  %s column\t%s\t%s\n", red("-"), column.Name, describeColumn(column)); err != nil {
				return err
			}
		}
		for _, column := range modification.ColumnsModified {
			if _, err := fmt.Fprintf(sectionWriter, "  %s column\t%s\t%s %s %s\n", yellow("~"), column.After.Name, describeColumn(column.Before), muted("→"), describeColumn(column.After)); err != nil {
				return err
			}
		}
		if err := sectionWriter.Flush(); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}

	if len(schemaDiff.IndexChanges) > 0 {
		if _, err := fmt.Fprintln(writer, cyan("Indexes")); err != nil {
			return err
		}
		for _, change := range schemaDiff.IndexChanges {
			marker := yellow("~")
			name := change.Name
			table := ""
			switch change.Kind {
			case "added":
				marker = green("+")
				if change.After != nil {
					table = change.After.TableName
				}
			case "dropped":
				marker = red("-")
				if change.Before != nil {
					table = change.Before.TableName
				}
			case "modified":
				if change.After != nil {
					table = change.After.TableName
				}
			}
			if table == "" {
				_, err := fmt.Fprintf(writer, "  %s %s index %s\n", marker, change.Kind, name)
				if err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(writer, "  %s %s index %s on %s\n", marker, change.Kind, name, table); err != nil {
				return err
			}
		}
	}

	return nil
}

// RenderJSON returns a stable pretty-printed JSON representation of schema changes.
func RenderJSON(schemaDiff *SchemaDiff) ([]byte, error) {
	if schemaDiff == nil {
		schemaDiff = &SchemaDiff{}
	}
	content, err := json.MarshalIndent(schemaDiff, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("rendering schema diff JSON: %w", err)
	}
	return append(content, '\n'), nil
}

func describeColumn(column ColumnDef) string {
	parts := []string{column.DataType}
	if column.MaxLength != nil {
		parts[0] = fmt.Sprintf("%s(%d)", column.DataType, *column.MaxLength)
	}
	if column.Nullable {
		parts = append(parts, "NULL")
	} else {
		parts = append(parts, "NOT NULL")
	}
	if column.Default != "" {
		parts = append(parts, "DEFAULT", column.Default)
	}
	return strings.Join(parts, " ")
}
