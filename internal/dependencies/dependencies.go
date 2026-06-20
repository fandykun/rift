//go:build dependencies

// Package dependencies pins phase-planned Go modules so `go mod tidy` keeps
// them available as later Rift phases wire API, audit, and local metadata code.
package dependencies

import (
	_ "github.com/go-chi/chi/v5"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)
