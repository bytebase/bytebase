package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/bytebase/bytebase/backend/common"
)

// StartMetadataInstance starts the metadata instance.
func StartMetadataInstance(ctx context.Context, pgBinDir, pgDataDir string, port int, mode common.ReleaseMode) (func(), error) {
	slog.Info("-----Embedded Postgres BEGIN-----")
	slog.Info(fmt.Sprintf("Start embedded Postgres datastorePort=%d pgDataDir=%s", port, pgDataDir))
	if err := initDB(pgBinDir, pgDataDir, "bb"); err != nil {
		return nil, err
	}
	serverLog := mode == common.ReleaseModeDev
	if err := start(port, pgBinDir, pgDataDir, serverLog); err != nil {
		return nil, err
	}
	slog.Info("-----Embedded Postgres END-----")
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%d user=bb database=postgres", common.GetPostgresSocketDir(), port))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var ok bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = bb)").Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		if _, err := db.ExecContext(ctx, "CREATE DATABASE bb"); err != nil {
			slog.Debug("database should already exists")
		}
	}

	return func() {
		if err := stop(pgBinDir, pgDataDir); err != nil {
			panic(err)
		}
	}, nil
}
