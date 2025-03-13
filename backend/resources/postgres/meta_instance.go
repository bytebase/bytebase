package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path"

	"github.com/bytebase/bytebase/backend/common"
)

// StartMetadataInstance starts the metadata instance.
func StartMetadataInstance(ctx context.Context, dataDir, pgBinDir, pgUser, demoName string, port int, mode common.ReleaseMode) (func(), error) {
	pgDataDir := getPostgresDataDir(dataDir, demoName)

	slog.Info("-----Embedded Postgres BEGIN-----")
	slog.Info(fmt.Sprintf("Start embedded Postgres datastorePort=%d pgDataDir=%s", port, pgDataDir))
	if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return nil, err
	}
	serverLog := mode == common.ReleaseModeDev
	if err := start(port, pgBinDir, pgDataDir, serverLog); err != nil {
		return nil, err
	}
	slog.Info("-----Embedded Postgres END-----")
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%d user=%s database=postgres", common.GetPostgresSocketDir(), port, pgUser))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var ok bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", pgUser).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", pgUser)); err != nil {
			slog.Debug("database should already exists")
		}
	}

	return func() {
		if err := stop(pgBinDir, pgDataDir); err != nil {
			panic(err)
		}
	}, nil
}

// getPostgresDataDir returns the postgres data directory of Bytebase.
func getPostgresDataDir(dataDir string, demoName string) string {
	// If demo is specified, we will use demo specific directory to store the demo data. Because
	// we reset the demo data when starting Bytebase and this can prevent accidentally removing the
	// production data.
	if demoName != "" {
		return path.Join(dataDir, "pgdata-demo", demoName)
	}
	return path.Join(dataDir, "pgdata")
}
