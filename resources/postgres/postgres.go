// Package postgres provides the resource for PostgreSQL server and utility packages.
package postgres

import (
	"bytes"
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

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/resources/utils"
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
	version := strings.TrimRight(tarName, ".txz")
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
func Start(port int, binDir string, dataDir string) (err error) {
	pgbin := filepath.Join(binDir, "pg_ctl")

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	p := exec.Command(pgbin, "start", "-w",
		"-D", dataDir,
		"-o", fmt.Sprintf(`-p %d -k %s -h "" -c stats_temp_directory=/tmp`, port, common.GetPostgresSocketDir()))

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
		if err := os.Chown(pgDataDir, int(uid), int(gid)); err != nil {
			return errors.Wrapf(err, "failed to change owner of data directory %q to bytebase", pgDataDir)
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
