package demo

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

//go:embed data
var demoFS embed.FS

// LoadDemoDataIfNeeded loads the demo data if specified.
func LoadDemoDataIfNeeded(ctx context.Context, stores *store.Store, demoName string) error {
	if demoName == "" {
		slog.Debug("Skip setting up demo data. Demo not specified.")
		return nil
	}
	slog.Info(fmt.Sprintf("Setting up demo %q...", demoName))

	db := stores.GetDB()
	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'environment')`,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		slog.Info("Skip setting up demo data. Data already exists.")
		return nil
	}

	names, err := fs.Glob(demoFS, fmt.Sprintf("data/%s/*.sql", demoName))
	if err != nil {
		return err
	}

	// Loop over all data files and execute them in order.
	for _, name := range names {
		if err := applyDataFile(name, db); err != nil {
			return errors.Wrapf(err, "Failed to load file: %q", name)
		}
	}
	slog.Info("Completed demo data setup.")
	return nil
}

// applyDataFile runs a single demo data file within a transaction.
func applyDataFile(name string, db *sql.DB) error {
	slog.Info(fmt.Sprintf("Applying data file %s...", name))
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Read and execute migration file.
	var buf []byte
	if buf, err = fs.ReadFile(demoFS, name); err != nil {
		return err
	}
	stmt := string(buf)
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}

	return tx.Commit()
}
