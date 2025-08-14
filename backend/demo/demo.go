package demo

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log/slog"
)

//go:embed data
var demoFS embed.FS

// LoadDemoData loads the demo data.
func LoadDemoData(ctx context.Context, db *sql.DB) error {
	slog.Info("Setting up demo...")

	// This query in the dump.sql will poison the connection.
	// SELECT pg_catalog.set_config('search_path', '', false);
	var ok bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'principal')`,
	).Scan(&ok); err != nil {
		return err
	}
	if ok {
		slog.Info("Skip setting up demo data. Data already exists.")
		return nil
	}

	buf, err := fs.ReadFile(demoFS, "data/dump.sql")
	if err != nil {
		return err
	}
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	if _, err := txn.Exec(string(buf)); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}

	// Reset the search_path to public after loading demo data
	// The dump.sql contains pg_catalog.set_config('search_path', '', false) which
	// clears the search_path for the session, causing subsequent queries to fail
	if _, err := db.Exec("SET search_path TO public"); err != nil {
		return err
	}

	slog.Info("Completed demo data setup.")
	return nil
}
