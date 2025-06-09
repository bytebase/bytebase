package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
)

// Sample data is from https://github.com/bytebase/employee-sample-database/tree/main/postgres/dataset_small
//
//go:embed sample
var sampleFS embed.FS

var (
	SampleUser         = "bbsample"
	SampleDatabaseTest = "hr_test"
	SampleDatabaseProd = "hr_prod"

	envs  = []string{"test", "prod"}
	envDB = map[string]string{
		"test": SampleDatabaseTest,
		"prod": SampleDatabaseProd,
	}
)

// StartAllSampleInstances starts all postgres sample instances.
func StartAllSampleInstances(ctx context.Context, dataDir string, basePort int) []func() {
	sampleData, err := loadSampleData()
	if err != nil {
		slog.Error("failed to load sample data", log.BBError(err))
		return nil
	}

	slog.Info("-----Sample Postgres Instance BEGIN-----")
	stoppers := []func(){}
	for i, env := range envs {
		port := basePort + i
		dataDir := path.Join(dataDir, "pgdata-sample", env)

		if err := initDB(dataDir, SampleUser); err != nil {
			slog.Error("failed to init sample instance", log.BBError(err))
			continue
		}

		slog.Info(fmt.Sprintf("Start sample instance %v at port %d", env, port))
		if err := start(port, dataDir, true /* serverLog */); err != nil {
			slog.Error("failed to start sample instance", log.BBError(err))
			continue
		}
		stoppers = append(stoppers, func() {
			if err := stop(dataDir); err != nil {
				panic(err)
			}
		})

		if err := setupSampleInstance(ctx, envDB[env], port, sampleData); err != nil {
			slog.Error("failed to init sample instance", log.BBError(err))
			continue
		}
	}

	slog.Info("-----Sample Postgres Instance END-----")
	return stoppers
}

func loadSampleData() (string, error) {
	// Load sample data
	names, err := fs.Glob(sampleFS, "sample/*.sql")
	if err != nil {
		return "", err
	}
	slices.Sort(names)

	var builder strings.Builder
	for _, name := range names {
		buf, err := fs.ReadFile(sampleFS, name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read sample database data: %s", name)
		}
		if _, err := builder.Write(buf); err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}

// setupSampleInstance starts a single postgres sample instance.
func setupSampleInstance(ctx context.Context, databaseName string, port int, sampleData string) error {
	defaultDB, err := sql.Open("pgx", fmt.Sprintf("user=%s host=%s port=%d database=postgres", SampleUser, common.GetPostgresSocketDir(), port))
	if err != nil {
		return err
	}
	defer defaultDB.Close()

	var ok bool
	if err := defaultDB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1);", databaseName).Scan(&ok); err != nil {
		return err
	}
	if ok {
		slog.Info(fmt.Sprintf("Sample database %s already exists, skip setup", databaseName))
		return nil
	}

	if _, err := defaultDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", databaseName)); err != nil {
		return errors.Wrapf(err, "failed to create sample database")
	}

	db, err := sql.Open("pgx", fmt.Sprintf("user=%s host=%s port=%d database=%s", SampleUser, common.GetPostgresSocketDir(), port, databaseName))
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, sampleData); err != nil {
		return err
	}
	return nil
}
