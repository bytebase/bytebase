//go:build mysql
// +build mysql

// Package mysql provides the resource for MySQL server and utility packages.
package mysql

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/resources/utils"

	// install mysql driver.
	_ "github.com/go-sql-driver/mysql"
)

// Instance is MySQL instance installed by Bytebase for testing.
type Instance struct {
	// baseDir is the directory where the mysql binary is installed.
	baseDir string
	// dataDir is the directory where the mysql data is stored.
	dataDir string
	// cfgPath is the path where the my.cnf is stored.
	cfgPath string
	// port is the port of the mysql instance.
	port int
	// proc is the process of the mysql instance.
	proc *os.Process
}

// DataDir returns the data dir.
func (i Instance) DataDir() string {
	return i.dataDir
}

// Port returns the port of the mysql instance.
func (i Instance) Port() int {
	return i.port
}

// Start starts the mysql instance on the given port, outputs to stdout and stderr.
// Waits at most `waitSec` seconds for the mysql instance to ready for connection.
// If `waitSec` is 0, it returns immediately.
func (i *Instance) Start(port int, stdout, stderr io.Writer) (err error) {
	i.port = port
	cmd := exec.Command(filepath.Join(i.baseDir, "bin", "mysqld"),
		fmt.Sprintf("--defaults-file=%s", i.cfgPath),
		fmt.Sprintf("--port=%d", i.port),
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	i.proc = cmd.Process

	// wait for mysql to start
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", i.port))
	if err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	// Timeout after 60 seconds.
	endTime := time.Now().Add(60 * time.Second)
	for range ticker.C {
		if err := db.Ping(); err == nil {
			break
		}
		if time.Now().After(endTime) {
			err := i.proc.Kill()
			if err != nil {
				return errors.Wrapf(err, "mysql instance has started as process %d, but failed to kill it", i.proc.Pid)
			}
			return errors.Errorf("failed to connect to mysql")
		}
	}

	return nil
}

// Stop stops the mysql instance, outputs to stdout and stderr.
func (i *Instance) Stop() error {
	return i.proc.Kill()
}

// Install installs mysql on basedir, prepares the data directory and default user.
func Install(resourceDir string) (string, error) {
	var tarName, version string
	// Mysql uses both tar.gz and tar.xz, so we use this ugly hack.
	var extractFn func(io.Reader, string) error
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		tarName = "mysql-8.0.33-macos13-x86_64.tar.gz"
		version = "mysql-8.0.33-macos13-x86_64"
		extractFn = utils.ExtractTarGz
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysql-8.0.33-macos13-arm64.tar.gz"
		version = "mysql-8.0.33-macos13-arm64"
		extractFn = utils.ExtractTarGz
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysql-8.0.33-linux-glibc2.17-x86_64-minimal.tar.xz"
		version = "mysql-8.0.33-linux-glibc2.17-x86_64-minimal"
		extractFn = utils.ExtractTarXz
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		tarName = "mysql-8.0.33-linux-glibc2.17-aarch64.tar.gz"
		version = "mysql-8.0.33-linux-glibc2.17-aarch64"
		extractFn = utils.ExtractTarXz
	default:
		return "", errors.Errorf("unsupported os %q and arch %q", runtime.GOOS, runtime.GOARCH)
	}
	mysqlBinDir := filepath.Join(resourceDir, version)
	needInstall := false
	if _, err := os.Stat(mysqlBinDir); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to check mysql binary directory path %q", mysqlBinDir)
		}
		// Install if not exist yet.
		needInstall = true
	}
	if needInstall {
		tarF, err := resources.Open(tarName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to open mysql dist %q", tarName)
		}
		defer tarF.Close()
		if err := extractFn(tarF, resourceDir); err != nil {
			return "", errors.Wrapf(err, "failed to extract mysql distribution %q", tarName)
		}
	}
	return mysqlBinDir, nil
}

// SetupTestInstance installs and starts a mysql instance for testing,
// returns the stop function.
func SetupTestInstance(t *testing.T, port int, mysqlBinDir string) func() {
	dataDir, cfgDir := t.TempDir(), t.TempDir()
	t.Log("Installing mysql...")
	configContent := fmt.Sprintf(`[mysqld]
basedir=%s
datadir=%s
socket=mysql.sock
mysqlx=0
user=%s
`, mysqlBinDir, dataDir, "root")
	defaultCfgFile, err := os.Create(filepath.Join(cfgDir, "my.cnf"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(defaultCfgFile, configContent); err != nil {
		t.Fatal(err)
	}
	defer defaultCfgFile.Close()

	args := []string{
		fmt.Sprintf("--defaults-file=%s", defaultCfgFile.Name()),
		"--initialize-insecure",
	}

	cmd := exec.Command(filepath.Join(mysqlBinDir, "bin", "mysqld"), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(errors.Wrap(err, "failed to initialize mysql"))
	}

	i := &Instance{
		baseDir: mysqlBinDir,
		dataDir: dataDir,
		cfgPath: defaultCfgFile.Name(),
	}

	t.Log("Starting mysql...")
	if err := i.Start(port, os.Stdout, os.Stderr); err != nil {
		t.Fatal(err)
	}

	stopFn := func() {
		t.Log("Stopping Mysql...")
		if err := i.Stop(); err != nil {
			t.Fatal(err)
		}
	}

	return stopFn
}

// Import executes sql script in the given path on the instance.
// If the path is a directory, it imports all sql scripts in the directory recursively.
func (i *Instance) Import(path string) error {
	var buf bytes.Buffer
	if err := cat(path, &buf); err != nil {
		return err
	}

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/?multiStatements=true", i.port))
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(buf.String())
	return err
}

func cat(path string, out io.Writer) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := cat(filepath.Join(path, entry.Name()), out); err != nil {
				return err
			}
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = io.Copy(out, f); err != nil {
			return err
		}
	}
	return nil
}
