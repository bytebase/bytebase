// Package postgres provides the resource for PostgreSQL server and utility packages.
package postgres

import (
	"bufio"
	"embed"
	"io/fs"
	"regexp"
	"sort"

	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/utils"
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
	// SampleDatabase is the sample database name.
	SampleDatabase = "employee"
)

// isPgDump15 returns true if the pg_dump binary is version 15.
func isPgDump15(pgDumpPath string) (bool, error) {
	var cmd *exec.Cmd
	var version bytes.Buffer
	cmd = exec.Command(pgDumpPath, "-V")
	cmd.Stdout = &version
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, err
	}
	pgDump15 := "pg_dump (PostgreSQL) 15.1\n"
	return pgDump15 == version.String(), nil
}

// Install will extract the postgres and utility tar in resourceDir.
// Returns the bin directory on success.
func Install(resourceDir string) (string, error) {
	var tarName string
	switch runtime.GOOS {
	case "darwin":
		tarName = "postgres-darwin-x86_64.txz"
	case "linux":
		tarName = "postgres-linux-x86_64.txz"
	default:
		return "", errors.Errorf("OS %q is not supported", runtime.GOOS)
	}
	version := strings.TrimSuffix(tarName, ".txz")
	pgBaseDir := path.Join(resourceDir, version)
	pgBinDir := path.Join(pgBaseDir, "bin")
	pgDumpPath := path.Join(pgBinDir, "pg_dump")
	needInstall := false

	if _, err := os.Stat(pgBaseDir); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to check postgres binary base directory path %q", pgBaseDir)
		}
		// Install if not exist yet.
		needInstall = true
	} else {
		// TODO(zp): remove this when pg_dump 15 is populated to all users.
		// Bytebase bump the pg_dump version to 15 to support PostgreSQL 15.
		// We need to reinstall the PostgreSQL resources if md5sum of pg_dump is different.
		// Check if pg_dump is version 15.
		isPgDump15, err := isPgDump15(pgDumpPath)
		if err != nil {
			return "", err
		}
		if !isPgDump15 {
			needInstall = true
			// Reinstall if pg_dump is not version 15.
			log.Info("Remove old postgres binary before installing new pg_dump...")
			if err := os.RemoveAll(pgBaseDir); err != nil {
				return "", errors.Wrapf(err, "failed to remove postgres binary base directory %q", pgBaseDir)
			}
		}
	}
	if needInstall {
		log.Info("Installing PostgreSQL utilities...")
		// The ordering below made Postgres installation atomic.
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
		if err := installInDir(tarName, tmpDir); err != nil {
			return "", err
		}

		if err := os.Rename(tmpDir, pgBaseDir); err != nil {
			return "", errors.Wrapf(err, "failed to rename postgres binary base directory from %q to %q", tmpDir, pgBaseDir)
		}
	}

	return pgBinDir, nil
}

// Start starts a postgres database instance.
// If port is 0, then it will choose a random unused port.
func Start(port int, binDir, dataDir string, serverLog bool) (err error) {
	pgbin := filepath.Join(binDir, "pg_ctl")

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	// We also set max_connections to 500 for tests.
	p := exec.Command(pgbin, "start", "-w",
		"-D", dataDir,
		"-o", fmt.Sprintf(`-p %d -k %s -N 500 -h "" -c stats_temp_directory=/tmp`, port, common.GetPostgresSocketDir()))

	uid, _, sameUser, err := shouldSwitchUser()
	if err != nil {
		return err
	}
	if !sameUser {
		p.SysProcAttr = &syscall.SysProcAttr{
			Setpgid:    true,
			Credential: &syscall.Credential{Uid: uint32(uid)},
		}
	}

	// It's useful to log the SQL statement errors from Postgres in developer environment.
	if serverLog {
		p.Stdout = os.Stdout
	}
	p.Stderr = os.Stderr
	if err := p.Run(); err != nil {
		return errors.Wrapf(err, "failed to start postgres %q", p.String())
	}

	return nil
}

// Stop stops a postgres instance, outputs to stdout and stderr.
func Stop(pgBinDir, pgDataDir string) error {
	pgbin := filepath.Join(pgBinDir, "pg_ctl")
	p := exec.Command(pgbin, "stop", "-w",
		"-D", pgDataDir)
	uid, _, sameUser, err := shouldSwitchUser()
	if err != nil {
		return err
	}
	if !sameUser {
		p.SysProcAttr = &syscall.SysProcAttr{
			Setpgid:    true,
			Credential: &syscall.Credential{Uid: uint32(uid)},
		}
	}

	// Suppress log spam
	p.Stdout = nil
	p.Stderr = os.Stderr
	return p.Run()
}

// InitDB inits a postgres database if not yet.
func InitDB(pgBinDir, pgDataDir, pgUser string) error {
	versionPath := filepath.Join(pgDataDir, "PG_VERSION")
	_, err := os.Stat(versionPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check postgres version in data directory path %q", versionPath)
	}
	initDBDone := false
	if err == nil {
		initDBDone = true
	}

	// Skip initDB if setup already.
	if initDBDone {
		// If file permission was mutated before, postgres cannot start up. We should change file permissions to 0700 for all pgdata files.
		if err := os.Chmod(pgDataDir, 0700); err != nil {
			return errors.Wrapf(err, "failed to chmod postgres data directory %q to 0700", pgDataDir)
		}
		return nil
	}

	// For pgDataDir and every intermediate to be created by MkdirAll, we need to prepare to chown
	// it to the bytebase user. Otherwise, initdb will complain file permission error.
	dirListToChown := []string{pgDataDir}
	path := filepath.Dir(pgDataDir)
	for path != "/" {
		_, err := os.Stat(path)
		if err == nil {
			break
		}
		if err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to check data directory path existence %q", path)
		}
		dirListToChown = append(dirListToChown, path)
		path = filepath.Dir(path)
	}
	log.Debug("Data directory list to Chown", zap.Any("dirListToChown", dirListToChown))

	if err := os.MkdirAll(pgDataDir, 0700); err != nil {
		return errors.Wrapf(err, "failed to make postgres data directory %q", pgDataDir)
	}

	args := []string{
		"-U", pgUser,
		"-D", pgDataDir,
	}
	initDBBinary := filepath.Join(pgBinDir, "initdb")
	p := exec.Command(initDBBinary, args...)
	p.Env = append(os.Environ(),
		"LC_ALL=en_US.UTF-8",
		"LC_CTYPE=en_US.UTF-8",
	)
	uid, gid, sameUser, err := shouldSwitchUser()
	if err != nil {
		return err
	}
	if !sameUser {
		p.SysProcAttr = &syscall.SysProcAttr{
			Setpgid:    true,
			Credential: &syscall.Credential{Uid: uint32(uid)},
		}
		log.Info(fmt.Sprintf("Recursively change owner of data directory %q to bytebase...", pgDataDir))
		for _, dir := range dirListToChown {
			log.Info(fmt.Sprintf("Change owner of %q to bytebase", dir))
			if err := os.Chown(dir, int(uid), int(gid)); err != nil {
				return errors.Wrapf(err, "failed to change owner of %q to bytebase", dir)
			}
		}
	}

	// Suppress log spam
	p.Stdout = nil
	p.Stderr = os.Stderr
	log.Info("-----Postgres initdb BEGIN-----")
	if err := p.Run(); err != nil {
		return errors.Wrapf(err, "failed to initdb %q", p.String())
	}
	log.Info("-----Postgres initdb END-----")

	return nil
}

func shouldSwitchUser() (int, int, bool, error) {
	sameUser := true
	bytebaseUser, err := user.Current()
	if err != nil {
		return 0, 0, true, errors.Wrap(err, "failed to get current user")
	}
	// If user runs Bytebase as root user, we will attempt to run as user `bytebase`.
	// https://www.postgresql.org/docs/14/app-initdb.html
	if bytebaseUser.Username == "root" {
		bytebaseUser, err = user.Lookup("bytebase")
		if err != nil {
			return 0, 0, false, errors.Errorf("please run Bytebase as non-root user. You can use the following command to create a dedicated \"bytebase\" user to run the application: addgroup --gid 113 --system bytebase && adduser --uid 113 --system bytebase && adduser bytebase bytebase")
		}
		sameUser = false
	}

	uid, err := strconv.ParseUint(bytebaseUser.Uid, 10, 32)
	if err != nil {
		return 0, 0, false, err
	}
	gid, err := strconv.ParseUint(bytebaseUser.Gid, 10, 32)
	if err != nil {
		return 0, 0, false, err
	}
	return int(uid), int(gid), sameUser, nil
}

// StartForTest starts a postgres instance on localhost given port, outputs to stdout and stderr.
// If port is 0, then it will choose a random unused port.
func StartForTest(port int, pgBinDir, pgDataDir string) (err error) {
	pgbin := filepath.Join(pgBinDir, "pg_ctl")

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	p := exec.Command(pgbin, "start", "-w",
		"-D", pgDataDir,
		"-o", fmt.Sprintf(`-p %d -k %s`, port, common.GetPostgresSocketDir()))

	// Suppress log spam
	p.Stdout = nil
	p.Stderr = os.Stderr
	if err := p.Run(); err != nil {
		return errors.Wrapf(err, "failed to start postgres %q", p.String())
	}

	return nil
}

// SetupTestInstance installs and starts a postgresql instance for testing,
// returns the stop function.
func SetupTestInstance(t *testing.T, port int, resourceDir string) func() {
	dataDir := t.TempDir()
	t.Log("Installing PostgreSQL...")
	binDir, err := Install(resourceDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("InitDB...")
	if err := InitDB(binDir, dataDir, "root"); err != nil {
		t.Fatal(err)
	}
	t.Log("Starting PostgreSQL...")
	if err := StartForTest(port, binDir, dataDir); err != nil {
		t.Fatal(err)
	}

	stopFn := func() {
		t.Log("Stopping PostgreSQL...")
		if err := Stop(binDir, dataDir); err != nil {
			t.Fatal(err)
		}
	}

	return stopFn
}

// Verify by pinging the sample database. As long as we encounter error, we will regard it as need
// to create sample database. This might not be 100% accurate since it could be connection issue.
// But if it's the connection issue, the following code will catch that anyway.
func needSetupSampleDatabase(ctx context.Context, pgUser, port, database string) bool {
	driver, err := db.Open(
		ctx,
		db.Postgres,
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

	// Connect the default postgres database created by initdb.
	if err := prepareDemoDatabase(ctx, pgUser, host, port, database); err != nil {
		return err
	}

	// Connect the just created sample database to load data.
	driver, err := db.Open(
		ctx,
		db.Postgres,
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

func prepareDemoDatabase(ctx context.Context, pgUser, host, port, database string) error {
	// Connect the default postgres database created by initdb.
	driver, err := db.Open(
		ctx,
		db.Postgres,
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

// StartSampleInstance starts a postgres sample instance.
func StartSampleInstance(ctx context.Context, pgBinDir, pgDataDir string, port int, mode common.ReleaseMode) error {
	if err := InitDB(pgBinDir, pgDataDir, SampleUser); err != nil {
		return errors.Wrapf(err, "failed to init sample instance")
	}

	if err := turnOnPGStateStatements(pgDataDir); err != nil {
		log.Warn("Failed to turn on pg_stat_statements", zap.Error(err))
	}

	if err := Start(port, pgBinDir, pgDataDir, mode == common.ReleaseModeDev /* serverLog */); err != nil {
		return errors.Wrapf(err, "failed to start sample instance")
	}

	host := common.GetPostgresSocketDir()
	if err := prepareSampleDatabaseIfNeeded(ctx, SampleUser, host, strconv.Itoa(port), SampleDatabase); err != nil {
		return errors.Wrapf(err, "failed to prepare sample database")
	}

	if err := createPGStatStatementsExtension(ctx, SampleUser, host, strconv.Itoa(port), SampleDatabase); err != nil {
		log.Warn("Failed to create pg_stat_statements extension", zap.Error(err))
	}

	return nil
}

func createPGStatStatementsExtension(ctx context.Context, pgUser, host, port, database string) error {
	driver, err := db.Open(
		ctx,
		db.Postgres,
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
	log.Info("Successfully created pg_stat_statements extension")
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

	log.Info("Successfully added pg_stat_statements to postgresql.conf file")
	return nil
}

func installInDir(tarName string, dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return errors.Wrapf(err, "failed to remove postgres binary temp directory %q", dir)
	}

	f, err := resources.Open(tarName)
	if err != nil {
		return errors.Wrapf(err, "failed to find %q in embedded resources", tarName)
	}
	defer f.Close()

	if err := utils.ExtractTarXz(f, dir); err != nil {
		return errors.Wrap(err, "failed to extract txz file")
	}
	return nil
}
