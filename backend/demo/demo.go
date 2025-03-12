package demo

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
)

//go:embed data
var demoFS embed.FS

// LoadDemoDataIfNeeded loads the demo data if specified.
func LoadDemoDataIfNeeded(ctx context.Context, db *sql.DB, demo string) error {
	if demo == "" {
		slog.Debug("Skip setting up demo data. Demo not specified.")
		return nil
	}
	slog.Info(fmt.Sprintf("Setting up demo %q...", demo))

	var ok bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'environment')`,
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

	slog.Info("Completed demo data setup.")
	return nil
}
