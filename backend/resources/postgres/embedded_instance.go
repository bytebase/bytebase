package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
)

// EmbeddedInstanceConfig configures one embedded PostgreSQL process and its
// initial database.
type EmbeddedInstanceConfig struct {
	DataDir      string
	Port         int
	User         string
	DatabaseName string
	SeedData     string
}

// StartEmbeddedInstance initializes and starts one embedded PostgreSQL process.
// The returned stopper preserves its data directory.
func StartEmbeddedInstance(ctx context.Context, config EmbeddedInstanceConfig) (func(), error) {
	if config.DataDir == "" || config.Port <= 0 || config.User == "" || config.DatabaseName == "" {
		return nil, errors.New("embedded PostgreSQL instance requires data directory, port, user, and database")
	}
	if err := initDB(config.DataDir, config.User); err != nil {
		return nil, errors.Wrap(err, "failed to initialize embedded PostgreSQL instance")
	}
	if err := start(config.Port, config.DataDir, true); err != nil {
		return nil, errors.Wrap(err, "failed to start embedded PostgreSQL instance")
	}
	stopper := func() {
		if err := stop(config.DataDir); err != nil {
			slog.Error("failed to stop embedded PostgreSQL instance", log.BBError(err))
		}
	}
	if err := setupEmbeddedInstance(ctx, config); err != nil {
		stopper()
		return nil, errors.Wrap(err, "failed to set up embedded PostgreSQL instance")
	}
	return stopper, nil
}

// RemoveEmbeddedInstance stops an embedded PostgreSQL process and removes its
// exact data directory. A missing directory is successful.
func RemoveEmbeddedInstance(dataDir string) error {
	if dataDir == "" {
		return errors.New("embedded PostgreSQL instance requires data directory")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "PG_VERSION")); err == nil {
		if err := stop(dataDir); err != nil {
			slog.Debug("embedded PostgreSQL instance was not running during removal", log.BBError(err))
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to inspect embedded PostgreSQL instance data")
	}
	return errors.Wrap(os.RemoveAll(dataDir), "failed to remove embedded PostgreSQL instance data")
}

func setupEmbeddedInstance(ctx context.Context, config EmbeddedInstanceConfig) error {
	defaultDB, err := sql.Open("pgx", fmt.Sprintf("user=%s host=%s port=%d database=postgres", config.User, common.GetPostgresSocketDir(), config.Port))
	if err != nil {
		return err
	}
	defer defaultDB.Close()

	var exists bool
	if err := defaultDB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1);", config.DatabaseName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := defaultDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{config.DatabaseName}.Sanitize())); err != nil {
		return errors.Wrap(err, "failed to create embedded PostgreSQL database")
	}

	database, err := sql.Open("pgx", fmt.Sprintf("user=%s host=%s port=%d database=%s", config.User, common.GetPostgresSocketDir(), config.Port, config.DatabaseName))
	if err != nil {
		return err
	}
	defer database.Close()
	if config.SeedData == "" {
		return nil
	}
	_, err = database.ExecContext(ctx, config.SeedData)
	return err
}
