package postgres

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

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
	// TestSampleInstanceResourceID is the resource id for the test sample database.
	TestSampleInstanceResourceID = "test-sample-instance"
	// ProdSampleInstanceResourceID is the resource id for the prod sample database.
	ProdSampleInstanceResourceID = "prod-sample-instance"
	// SampleUser is the user name for the sample database.
	SampleUser = "bbsample"
	// SampleDatabaseTest is the test sample database name.
	SampleDatabaseTest = "hr_test"
	// SampleDatabaseProd is the prod sample database name.
	SampleDatabaseProd = "hr_prod"
)

// StartSampleInstance starts a postgres sample instance.
func StartSampleInstance(ctx context.Context, pgBinDir, dataDir, sampleDatabaseName string, port int, mode common.ReleaseMode) (func(), error) {
	pgDataDir := path.Join(dataDir, "pgdata-sample", sampleDatabaseName)

	v, err := getVersion(pgDataDir)
	if err != nil {
		return nil, err
	}
	if v != currentVersion {
		slog.Warn("delete sample postgres with different version", slog.String("old", v), slog.String("new", currentVersion))
		err := os.RemoveAll(pgDataDir)
		if err != nil {
			return nil, err
		}
	}
	if err := initDB(pgBinDir, pgDataDir, SampleUser); err != nil {
		return nil, errors.Wrapf(err, "failed to init sample instance")
	}

	if err := turnOnPGStateStatements(pgDataDir); err != nil {
		slog.Warn("Failed to turn on pg_stat_statements", log.BBError(err))
	}

	if err := start(port, pgBinDir, pgDataDir, mode == common.ReleaseModeDev /* serverLog */); err != nil {
		return nil, errors.Wrapf(err, "failed to start sample instance")
	}

	host := common.GetPostgresSocketDir()
	if err := prepareSampleDatabaseIfNeeded(ctx, SampleUser, host, strconv.Itoa(port), sampleDatabaseName); err != nil {
		return nil, errors.Wrapf(err, "failed to prepare sample database")
	}

	if err := createPGStatStatementsExtension(ctx, SampleUser, host, strconv.Itoa(port), sampleDatabaseName); err != nil {
		slog.Warn("Failed to create pg_stat_statements extension", log.BBError(err))
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
func needSetupSampleDatabase(ctx context.Context, pgUser, port, database string) bool {
	driver, err := db.Open(
		ctx,
		storepb.Engine_POSTGRES,
		db.DriverConfig{},
		db.ConnectionConfig{
			Username: pgUser,
			Password: "",
			Host:     common.GetPostgresSocketDir(),
			Port:     port,
			Database: database,
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return true
	}
	defer driver.Close(ctx)

	if err := driver.Ping(ctx); err != nil {
		return true
	}
	return false
}

// prepareSampleDatabaseIfNeeded creates sample database if needed.
func prepareSampleDatabaseIfNeeded(ctx context.Context, pgUser, host, port, database string) error {
	if !needSetupSampleDatabase(ctx, pgUser, port, database) {
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
			Username: pgUser,
			Password: "",
			Host:     host,
			Port:     port,
			Database: database,
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to connect sample database")
	}
	defer driver.Close(ctx)

	// Load sample data
	names, err := fs.Glob(sampleFS, "sample/*.sql")
	if err != nil {
		return err
	}

	sort.Strings(names)

	for _, name := range names {
		if buf, err := fs.ReadFile(sampleFS, name); err != nil {
			return errors.Wrapf(err, fmt.Sprintf("failed to read sample database data: %s", name))
		} else if _, err := driver.Execute(ctx, string(buf), false, db.ExecuteOptions{}); err != nil {
			return errors.Wrapf(err, fmt.Sprintf("failed to load sample database data: %s", name))
		}
	}

	// Drop the default postgres database, this is to present a cleaner database list to the user.
	if _, err := driver.Execute(ctx, "DROP DATABASE postgres", true, db.ExecuteOptions{}); err != nil {
		return errors.Wrapf(err, "failed to drop default postgres database")
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
			Username: pgUser,
			Password: "",
			Host:     host,
			Port:     port,
			Database: "postgres",
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to connect sample instance")
	}
	defer driver.Close(ctx)

	// Create the sample database.
	if _, err := driver.Execute(ctx, fmt.Sprintf("CREATE DATABASE %s", database), true, db.ExecuteOptions{}); err != nil {
		return errors.Wrapf(err, "failed to create sample database")
	}

	return nil
}

func createPGStatStatementsExtension(ctx context.Context, pgUser, host, port, database string) error {
	driver, err := db.Open(
		ctx,
		storepb.Engine_POSTGRES,
		db.DriverConfig{},
		db.ConnectionConfig{
			Username: pgUser,
			Password: "",
			Host:     host,
			Port:     port,
			Database: database,
		},
		db.ConnectionContext{},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to connect sample database")
	}
	defer driver.Close(ctx)

	if _, err := driver.Execute(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;", false, db.ExecuteOptions{}); err != nil {
		return errors.Wrapf(err, "failed to create pg_stat_statements extension")
	}
	slog.Info("Successfully created pg_stat_statements extension")
	return nil
}

// turnOnPGStateStatements turns on pg_stat_statements extension.
// Only works for sample PostgreSQL.
func turnOnPGStateStatements(pgDataDir string) error {
	// Enable pg_stat_statements extension
	// Add shared_preload_libraries = 'pg_stat_statements' to postgresql.conf
	pgConfig := filepath.Join(pgDataDir, "postgresql.conf")

	// Check config in postgresql.conf
	configFile, err := os.OpenFile(pgConfig, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to open postgresql.conf file")
	}
	defer configFile.Close()

	scanner := bufio.NewScanner(configFile)
	shardPreloadLibrariesReg := regexp.MustCompile(`^\s*shared_preload_libraries\s*=\s*'pg_stat_statements'`)
	pgStatStatementsTrackReg := regexp.MustCompile(`^\s*pg_stat_statements.track\s*=`)
	shardPreloadLibraries := false
	pgStatStatementsTrack := false
	for scanner.Scan() {
		line := scanner.Text()
		if !shardPreloadLibraries && shardPreloadLibrariesReg.MatchString(line) {
			shardPreloadLibraries = true
		}

		if !pgStatStatementsTrack && pgStatStatementsTrackReg.MatchString(line) {
			pgStatStatementsTrack = true
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrapf(err, "failed to scan postgresql.conf file")
	}

	if !shardPreloadLibraries {
		if _, err := configFile.WriteString("\nshared_preload_libraries = 'pg_stat_statements'\n"); err != nil {
			return errors.Wrapf(err, "failed to write shared_preload_libraries = 'pg_stat_statements' to postgresql.conf file")
		}
	}

	if !pgStatStatementsTrack {
		if _, err := configFile.WriteString("\npg_stat_statements.track = all\n"); err != nil {
			return errors.Wrapf(err, "failed to write pg_stat_statements.track = all to postgresql.conf file")
		}
	}

	slog.Info("Successfully added pg_stat_statements to postgresql.conf file")
	return nil
}
