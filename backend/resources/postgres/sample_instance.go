package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Sample data is from https://github.com/bytebase/employee-sample-database/tree/main/postgres/dataset_small
//
//go:embed sample
var sampleFS embed.FS

const (
	// SampleUser is the user name for the sample database.
	SampleUser = "bbsample"
	// SampleDatabaseTest is the test sample database name.
	SampleDatabaseTest = "hr_test"
	// SampleDatabaseProd is the prod sample database name.
	SampleDatabaseProd = "hr_prod"
)

// StartAllSampleInstances starts all postgres sample instances.
func StartAllSampleInstances(ctx context.Context, pgBinDir, dataDir string, port int, includeBatch bool) []func() {
	// Load sample data
	sampleData, err := loadSampleData()
	if err != nil {
		slog.Error("failed to load sample data", log.BBError(err))
		return nil
	}

	envs := []string{"test", "prod"}
	dbsPerEnv := map[string][]string{
		"test": {SampleDatabaseTest},
		"prod": {SampleDatabaseProd},
	}

	slog.Info("-----Sample Postgres Instance BEGIN-----")
	i := 0
	for _, env := range envs {
		dbs := dbsPerEnv[env]
		slog.Info(fmt.Sprintf("Setup sample instance %v", env))
		if err := setupOneSampleInstance(ctx, pgBinDir, path.Join(dataDir, "pgdata-sample", env), dbs, port+i, includeBatch, sampleData); err != nil {
			slog.Error("failed to init sample instance", log.BBError(err))
			continue
		}
		i++
	}

	i = 0
	stoppers := []func(){}
	for _, env := range envs {
		slog.Info(fmt.Sprintf("Start sample instance %v at port %d", env, port+i))
		stopper, err := startOneSampleInstance(pgBinDir, path.Join(dataDir, "pgdata-sample", env), port+i)
		i++
		if err != nil {
			slog.Error("failed to init sample instance", log.BBError(err))
			continue
		}
		stoppers = append(stoppers, stopper)
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
	sort.Strings(names)

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

// setupOneSampleInstance starts a single postgres sample instance.
func setupOneSampleInstance(ctx context.Context, pgBinDir, pgDataDir string, dbs []string, port int, includeBatch bool, sampleData string) error {
	v, err := getVersion(pgDataDir)
	if err != nil {
		return err
	}
	if v != "" && v != currentVersion {
		slog.Warn("delete sample postgres with different version", slog.String("old", v), slog.String("new", currentVersion))
		err := os.RemoveAll(pgDataDir)
		if err != nil {
			return err
		}
	}
	if err := initDB(pgBinDir, pgDataDir, SampleUser); err != nil {
		return errors.Wrapf(err, "failed to init sample instance")
	}

	// TODO(tianzhou): Remove this after debugging completes.
	// turn on serverlog to debug sample instance startup in SaaS.
	if err := start(port, pgBinDir, pgDataDir, true /* serverLog */); err != nil {
		return errors.Wrapf(err, "failed to start sample instance")
	}

	host := common.GetPostgresSocketDir()
	for _, v := range dbs {
		if err := prepareSampleDatabaseIfNeeded(ctx, SampleUser, host, strconv.Itoa(port), v, sampleData); err != nil {
			return errors.Wrapf(err, "failed to prepare sample database %q", v)
		}
		if !includeBatch {
			break
		}
	}

	// Drop the default postgres database, this is to present a cleaner database list to the user.
	if err := dropDefaultPostgresDatabase(ctx, SampleUser, host, strconv.Itoa(port), dbs[0]); err != nil {
		slog.Warn("Failed to drop default postgres database", log.BBError(err))
	}

	return stop(pgBinDir, pgDataDir)
}

// startOneSampleInstance starts a single postgres sample instance.
func startOneSampleInstance(pgBinDir, pgDataDir string, port int) (func(), error) {
	if err := start(port, pgBinDir, pgDataDir, true /* serverLog */); err != nil {
		return nil, errors.Wrapf(err, "failed to start sample instance")
	}
	return func() {
		if err := stop(pgBinDir, pgDataDir); err != nil {
			panic(err)
		}
	}, nil
}

// Verify by pinging the sample database. As long as we encounter error, we will regard it as need
// to create sample database. This might not be 100% accurate since it could be connection issue.
// But if it's the connection issue, the following code will catch that anyway.
func needSetupSampleDatabase(ctx context.Context, pgUser, port, database string) (bool, error) {
	driver, err := db.Open(
		ctx,
		storepb.Engine_POSTGRES,
		db.DriverConfig{},
		db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username: pgUser,
				Password: "",
				Host:     common.GetPostgresSocketDir(),
				Port:     port,
				Database: database,
			},
			ConnectionContext: db.ConnectionContext{
				DatabaseName: database,
			},
			MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
		},
	)
	if err != nil {
		return false, err
	}
	defer driver.Close(ctx)

	if err := driver.Ping(ctx); err != nil {
		slog.Debug("sample database ping error", slog.String("database", database))
		// nolint
		return true, nil
	}
	row := driver.GetDB().QueryRowContext(ctx, "select count(1) from pg_tables where schemaname='public';")
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

// prepareSampleDatabaseIfNeeded creates sample database if needed.
func prepareSampleDatabaseIfNeeded(ctx context.Context, pgUser, host, port, database, sampleData string) error {
	needSetup, err := needSetupSampleDatabase(ctx, pgUser, port, database)
	if err != nil {
		return err
	}
	if !needSetup {
		slog.Info(fmt.Sprintf("Sample database %v already exists, skip setup", database))
		return nil
	}

	// Connect the default database created by initdb.
	if err := prepareSampleDatabase(ctx, pgUser, host, port, database); err != nil {
		return err
	}

	// Connect the just created sample database to load data.
	driver, err := db.Open(
		ctx,
		storepb.Engine_POSTGRES,
		db.DriverConfig{},
		db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username: pgUser,
				Password: "",
				Host:     host,
				Port:     port,
				Database: database,
			},
			ConnectionContext: db.ConnectionContext{
				DatabaseName: database,
			},
			MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to connect sample database")
	}
	defer driver.Close(ctx)

	if _, err := driver.GetDB().ExecContext(ctx, sampleData); err != nil {
		return errors.Wrapf(err, "failed to load sample database data")
	}
	return nil
}

func prepareSampleDatabase(ctx context.Context, pgUser, host, port, database string) error {
	// Connect the default postgres database created by initdb.
	driver, err := db.Open(
		ctx,
		storepb.Engine_POSTGRES,
		db.DriverConfig{},
		db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username: pgUser,
				Password: "",
				Host:     host,
				Port:     port,
				Database: "postgres",
			},
			ConnectionContext: db.ConnectionContext{
				DatabaseName: database,
			},
			MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to connect sample instance")
	}
	defer driver.Close(ctx)

	// Create the sample database.
	if _, err := driver.Execute(ctx, fmt.Sprintf("CREATE DATABASE %s", database), db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return errors.Wrapf(err, "failed to create sample database")
	}
	slog.Info(fmt.Sprintf("Successfully created database %s", database))

	return nil
}

func dropDefaultPostgresDatabase(ctx context.Context, pgUser, host, port, connectingDb string) error {
	driver, err := db.Open(
		ctx,
		storepb.Engine_POSTGRES,
		db.DriverConfig{},
		db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username: pgUser,
				Password: "",
				Host:     host,
				Port:     port,
				Database: connectingDb,
			},
			ConnectionContext: db.ConnectionContext{
				DatabaseName: connectingDb,
			},
			MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to connect sample instance")
	}
	defer driver.Close(ctx)

	// Drop the default postgres database.
	if _, err := driver.Execute(ctx, "DROP DATABASE IF EXISTS postgres", db.ExecuteOptions{}); err != nil {
		return errors.Wrapf(err, "failed to drop default postgres database")
	}

	return nil
}
