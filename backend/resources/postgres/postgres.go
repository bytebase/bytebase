// Package postgres provides the resource for PostgreSQL server and utility packages.
package postgres

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/resources/utils"
)

const (
	currentVersion = "16"
)

// Install will extract the postgres and utility tar in resourceDir.
// Returns the bin directory on success.
func Install(resourceDir string) (string, error) {
	t1 := time.Now()
	defer func() {
		slog.Info("postgresutil", "cost", time.Now().Sub(t1))
	}()
	pkgNamePrefix, err := getTarName()
	if err != nil {
		return "", err
	}
	tarName := pkgNamePrefix + ".txz"

	var pgBaseDir string
	if currentVersion == "14" {
		pgBaseDir = path.Join(resourceDir, pkgNamePrefix)
	} else {
		pgBaseDir = path.Join(resourceDir, fmt.Sprintf("%s-%s", pkgNamePrefix, currentVersion))
	}
	needInstall := false
	if _, err := os.Stat(pgBaseDir); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to check postgres binary base directory path %q", pgBaseDir)
		}
		// Install if not exist yet.
		needInstall = true
	}
	if needInstall {
		slog.Info("Installing PostgreSQL utilities...")
		// The ordering below made Postgres installation atomic.
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s%s", pkgNamePrefix, currentVersion))
		if err := installInDir(tarName, tmpDir); err != nil {
			return "", err
		}

		if err := os.Rename(tmpDir, pgBaseDir); err != nil {
			return "", errors.Wrapf(err, "failed to rename postgres binary base directory from %q to %q", tmpDir, pgBaseDir)
		}
	}

	pgBinDir := path.Join(pgBaseDir, "bin")
	return pgBinDir, nil
}

func getTarName() (string, error) {
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		return "postgres-darwin-amd64", nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "postgres-darwin-arm64", nil
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "postgres-linux-amd64", nil
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		return "postgres-linux-arm64", nil
	default:
		return "", errors.Errorf("unsupported combination of OS %q and ARCH %q", runtime.GOOS, runtime.GOARCH)
	}
}

// start starts a postgres database instance.
// If port is 0, then it will choose a random unused port.
func start(port int, binDir, dataDir string, serverLog bool) (err error) {
	pgbin := filepath.Join(binDir, "pg_ctl")

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	// We also set max_connections to 500 for tests.
	p := exec.Command(pgbin, "start", "-w",
		"-D", dataDir,
		"-o", fmt.Sprintf(`-p %d -k %s -N 500 -h ""`, port, common.GetPostgresSocketDir()))

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

// stop stops a postgres instance, outputs to stdout and stderr.
func stop(pgBinDir, pgDataDir string) error {
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

// initDB inits a postgres database if not yet.
func initDB(pgBinDir, pgDataDir, pgUser string) error {
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
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to check data directory path existence %q", path)
		}
		dirListToChown = append(dirListToChown, path)
		path = filepath.Dir(path)
	}
	slog.Debug("Data directory list to Chown", slog.Any("dirListToChown", dirListToChown))

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
		slog.Info(fmt.Sprintf("Recursively change owner of data directory %q to bytebase...", pgDataDir))
		for _, dir := range dirListToChown {
			slog.Info(fmt.Sprintf("Change owner of %q to bytebase", dir))
			if err := os.Chown(dir, int(uid), int(gid)); err != nil {
				return errors.Wrapf(err, "failed to change owner of %q to bytebase", dir)
			}
		}
	}

	// Suppress log spam
	p.Stdout = nil
	p.Stderr = os.Stderr
	slog.Info("-----Postgres initdb BEGIN-----")
	if err := p.Run(); err != nil {
		return errors.Wrapf(err, "failed to initdb %q", p.String())
	}
	slog.Info("-----Postgres initdb END-----")

	return nil
}

func getVersion(pgDataPath string) (string, error) {
	versionPath := filepath.Join(pgDataPath, "PG_VERSION")
	data, err := os.ReadFile(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", errors.Wrapf(err, "failed to check postgres version in data directory path %q", versionPath)
	}
	return strings.TrimRight(string(data), "\n"), nil
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
