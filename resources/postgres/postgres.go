package postgres

import (
	"fmt"
	"io"
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

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/resources/utils"
)

// Instance is a postgres instance installed by bytebase
// for backend storage or testing.
type Instance struct {
	// BaseDir is the directory where the postgres binary is installed.
	BaseDir string
	// dataDir is the directory where the postgres data is stored.
	dataDir string
	// port is the port number of the postgres instance.
	port int
}

// Port returns the port number of the postgres instance.
func (i Instance) Port() int { return i.port }

// Start starts a postgres instance on given port, outputs to stdout and stderr.
//
// If port is 0, then it will choose a random unused port.
//
// If waitSec > 0, watis at most `waitSec` seconds for the postgres instance to start.
// Otherwise, returns immediately.
func (i *Instance) Start(port int, stdout, stderr io.Writer) (err error) {
	pgbin := filepath.Join(i.BaseDir, "bin", "pg_ctl")

	i.port = port

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	p := exec.Command(pgbin, "start", "-w",
		"-D", i.dataDir,
		"-o", fmt.Sprintf(`-p %d -k %s -h ""`, i.port, common.GetPostgresSocketDir()))

	p.Stdout = stdout
	p.Stderr = stderr
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

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to start postgres %q, error %v", p.String(), err)
	}

	return nil
}

// Stop stops a postgres instance, outputs to stdout and stderr.
func (i *Instance) Stop(stdout, stderr io.Writer) error {
	pgbin := filepath.Join(i.BaseDir, "bin", "pg_ctl")
	p := exec.Command(pgbin, "stop", "-w",
		"-D", i.dataDir)

	p.Stderr = stderr
	p.Stdout = stdout
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

	return p.Run()
}

// Install returns the postgres binary depending on the OS.
func Install(resourceDir, pgDataDir, pgUser string) (*Instance, error) {
	var tarName string
	switch runtime.GOOS {
	case "darwin":
		tarName = "postgres-darwin-x86_64.txz"
	case "linux":
		tarName = "postgres-linux-x86_64.txz"
	default:
		return nil, fmt.Errorf("OS %q is not supported", runtime.GOOS)
	}
	version := strings.TrimRight(tarName, ".txz")
	pgBinDir := path.Join(resourceDir, version)

	// TODO(d): remove this when pg_dump is populated to all users.
	_, err := os.Stat(path.Join(pgBinDir, "bin", "pg_dump"))
	pgDumpNotExist := false
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check pg_dump %q, error: %w", pgBinDir, err)
		}
		pgDumpNotExist = true
	}
	if pgDumpNotExist {
		if err := os.RemoveAll(pgBinDir); err != nil {
			return nil, fmt.Errorf("failed to remove binary directory path %q, error: %w", pgBinDir, err)
		}
	}

	if _, err := os.Stat(pgBinDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check binary directory path %q, error: %w", pgBinDir, err)
		}
		// Install if not exist yet.
		// The ordering below made Postgres installation atomic.
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
		if err := os.RemoveAll(tmpDir); err != nil {
			return nil, fmt.Errorf("failed to remove postgres binary temp directory %q, error: %w", tmpDir, err)
		}

		f, err := resources.Open(tarName)
		if err != nil {
			return nil, fmt.Errorf("failed to find %q in embedded resources, error: %v", tarName, err)
		}
		defer f.Close()

		if err := utils.ExtractTarXz(f, tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract txz file, error: %w", err)
		}

		if err := os.Rename(tmpDir, pgBinDir); err != nil {
			return nil, fmt.Errorf("failed to rename postgres binary directory from %q to %q, error: %w", tmpDir, pgBinDir, err)
		}
	}

	// We will initialize Postgres only when pgDataDir is set.
	if pgDataDir != "" {
		if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
			return nil, err
		}
	}

	return &Instance{
		BaseDir: pgBinDir,
		dataDir: pgDataDir,
	}, nil
}

// initDB inits a postgres database if not yet.
func initDB(pgBinDir, pgDataDir, pgUser string) error {
	versionPath := filepath.Join(pgDataDir, "PG_VERSION")
	_, err := os.Stat(versionPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check postgres version in data directory path %q, error: %w", versionPath, err)
	}
	initDBDone := false
	if err == nil {
		initDBDone = true
	}

	// Skip initDB if setup already.
	if initDBDone {
		// If file permission was mutated before, postgres cannot start up. We should change file permissions to 0700 for all pgdata files.
		if err := os.Chmod(pgDataDir, 0700); err != nil {
			return fmt.Errorf("failed to chmod postgres data directory %q to 0700, error: %w", pgDataDir, err)
		}
		return nil
	}

	if err := os.MkdirAll(pgDataDir, 0700); err != nil {
		return fmt.Errorf("failed to make postgres data directory %q, error: %w", pgDataDir, err)
	}

	args := []string{
		"-U", pgUser,
		"-D", pgDataDir,
	}
	initDBBinary := filepath.Join(pgBinDir, "bin", "initdb")
	p := exec.Command(initDBBinary, args...)
	p.Env = append(os.Environ(),
		"LC_ALL=en_US.UTF-8",
		"LC_CTYPE=en_US.UTF-8",
	)
	p.Stderr = os.Stderr
	p.Stdout = os.Stdout
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
			return fmt.Errorf("failed to change owner of data directory %q to bytebase, error: %w", pgDataDir, err)
		}
	}

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to initdb %q, error %v", p.String(), err)
	}

	return nil
}

func shouldSwitchUser() (int, int, bool, error) {
	sameUser := true
	bytebaseUser, err := user.Current()
	if err != nil {
		return 0, 0, true, fmt.Errorf("failed to get current user, error: %w", err)
	}
	// If user runs Bytebase as root user, we will attempt to run as user `bytebase`.
	// https://www.postgresql.org/docs/14/app-initdb.html
	if bytebaseUser.Username == "root" {
		bytebaseUser, err = user.Lookup("bytebase")
		if err != nil {
			return 0, 0, false, fmt.Errorf("please run Bytebase as non-root user. You can use the following command to create a dedicated \"bytebase\" user to run the application: addgroup --gid 113 --system bytebase && adduser --uid 113 --system bytebase && adduser bytebase bytebase")
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
//
// If port is 0, then it will choose a random unused port.
//
// If waitSec > 0, watis at most `waitSec` seconds for the postgres instance to start.
// Otherwise, returns immediately.
func (i *Instance) StartForTest(port int, stdout, stderr io.Writer) (err error) {
	pgbin := filepath.Join(i.BaseDir, "bin", "pg_ctl")

	i.port = port

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	p := exec.Command(pgbin, "start", "-w",
		"-D", i.dataDir,
		"-o", fmt.Sprintf(`-p %d -k %s -h 127.0.0.1`, i.port, common.GetPostgresSocketDir()))

	p.Stdout = stdout
	p.Stderr = stderr

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to start postgres %q, error %v", p.String(), err)
	}

	return nil
}

// SetupTestInstance installs and starts a postgresql instance for testing,
// returns the instance and the stop function.
func SetupTestInstance(t *testing.T, port int) (*Instance, func()) {
	basedir, datadir := t.TempDir(), t.TempDir()
	t.Log("Installing PostgreSQL...")
	i, err := Install(basedir, datadir, "root")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Starting PostgreSQL...")
	if err := i.StartForTest(port, os.Stdout, os.Stderr); err != nil {
		t.Fatal(err)
	}

	stopFn := func() {
		t.Log("Stopping PostgreSQL...")
		if err := i.Stop(os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
	}

	return i, stopFn
}
