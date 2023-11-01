package demo

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/pkg/errors"
)

//go:embed data
var demoFS embed.FS

// setupDemoData loads the demo data.
func SetupDemoData(demoName string, db *sql.DB) error {
	if demoName == "" {
		slog.Debug("Skip setting up demo data. Demo not specified.")
		return nil
	}

	slog.Info(fmt.Sprintf("Setting up demo %q...", demoName))

	// Reset existing demo data.
	if err := applyDataFile("data/reset.sql", db); err != nil {
		return errors.Wrapf(err, "Failed to reset demo data")
	}

	names, err := fs.Glob(demoFS, fmt.Sprintf("data/%s/*.sql", demoName))
	if err != nil {
		return err
	}

	// We separate demo data for each table into their own demo data file.
	// And there exists foreign key dependency among tables, so we
	// name the data file as 10001_xxx.sql, 10002_xxx.sql. Here we sort
	// the file name so they are loaded accordingly.
	sort.Strings(names)

	// Loop over all data files and execute them in order.
	for _, name := range names {
		if err := applyDataFile(name, db); err != nil {
			return errors.Wrapf(err, "Failed to load demo data: %q", name)
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
	if buf, err := fs.ReadFile(demoFS, name); err != nil {
		return err
	} else if _, err := tx.Exec(string(buf)); err != nil {
		return err
	}

	return tx.Commit()
}
