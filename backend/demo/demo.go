package demo

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"

	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

//go:embed data
var demoFS embed.FS

// LoadDemoDataIfNeeded loads the demo data if specified.
func LoadDemoDataIfNeeded(ctx context.Context, storeDB *store.DB, demoName string, mode common.ReleaseMode) error {
	if demoName == "" {
		slog.Debug("Skip setting up demo data. Demo not specified.")
		return nil
	}

	slog.Info(fmt.Sprintf("Setting up demo %q...", demoName))

	metadataDriver, err := dbdriver.Open(
		ctx,
		storepb.Engine_POSTGRES,
		dbdriver.DriverConfig{},
		storeDB.ConnCfg,
	)
	if err != nil {
		return err
	}
	defer metadataDriver.Close(ctx)

	var exists bool
	if err := metadataDriver.GetDB().QueryRowContext(ctx,
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

	// We separate demo data for each table into their own demo data file.
	// And there exists foreign key dependency among tables, so we
	// name the data file as 10001_xxx.sql, 10002_xxx.sql. Here we sort
	// the file name so they are loaded accordingly.
	sort.Strings(names)

	// Loop over all data files and execute them in order.
	for _, name := range names {
		if err := applyDataFile(name, metadataDriver.GetDB(), mode); err != nil {
			return errors.Wrapf(err, "Failed to load file: %q", name)
		}
	}
	slog.Info("Completed demo data setup.")
	return nil
}

// applyDataFile runs a single demo data file within a transaction.
func applyDataFile(name string, db *sql.DB, mode common.ReleaseMode) error {
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
	// The demo dump is generated from dev mode and using "bbdev" as the owner. For release mode,
	// we need to replace with "bb".
	if mode == common.ReleaseModeProd {
		stmt = strings.ReplaceAll(stmt, "bbdev", "bb")
	}
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}

	return tx.Commit()
}
