//go:build mysql
// +build mysql

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

	"github.com/bytebase/bytebase/resources/utils"

	// install mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// Instance is MySQL instance installed by bytebase for testing.
type Instance struct {
	// basedir is the directory where the mysql binary is installed.
	basedir string
	// datadir is the directory where the mysql data is stored.
	datadir string
	// port is the port of the mysql instance.
	port int
	// proc is the process of the mysql instance.
	proc *os.Process
}

// BackendPort returns the port of the mysql instance.
func (i Instance) Port() int { return i.port }

// Start starts the mysql instance on the given port, outputs to stdout and stderr.
// Waits at most `waitSec` seconds for the mysql instance to ready for connection.
// If `waitSec` is 0, it returns immediately.
func (i *Instance) Start(port int, stdout, stderr io.Writer) (err error) {
	i.port = port
	cmd := exec.Command(filepath.Join(i.basedir, "bin", "mysqld"),
		fmt.Sprintf("--defaults-file=%s", filepath.Join(i.basedir, "my.cnf")),
		fmt.Sprintf("--port=%d", i.port),
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	i.proc = cmd.Process

	// wait for mysql to start
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", i.port))
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
				return fmt.Errorf("mysql instance has started as process %d, but failed to kill it, error: %w", i.proc.Pid, err)
			}
			return fmt.Errorf("failed to connect to mysql")
		}
	}

	return nil
}

// Stop stops the mysql instance, outputs to stdout and stderr.
func (i *Instance) Stop(stdout, stderr io.Writer) error {
	return i.proc.Kill()
}

// Install installs mysql on basedir, prepares the data directory and default user.
func Install(basedir, datadir, user string) (*Instance, error) {
	var tarName, version string
	// Mysql uses both tar.gz and tar.xz, so we use this ugly hack.
	var extractFn func(io.Reader, string) error
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysql-8.0.28-macos11-arm64.tar.gz"
		version = "mysql-8.0.28-macos11-arm64"
		extractFn = utils.ExtractTarGz
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz"
		version = "mysql-8.0.28-linux-glibc2.17-x86_64-minimal"
		extractFn = utils.ExtractTarXz
	default:
		return nil, fmt.Errorf("unsupported os %q and arch %q", runtime.GOOS, runtime.GOARCH)
	}

	tarF, err := resources.Open(tarName)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql dist %q, error: %w", tarName, err)
	}
	defer tarF.Close()

	if err := extractFn(tarF, basedir); err != nil {
		return nil, fmt.Errorf("failed to extract mysql distribution %q, error: %w", tarName, err)
	}

	basedir = filepath.Join(basedir, version)

	const configFmt = `[mysqld]
basedir=%s
datadir=%s
socket=mysql.sock
user=%s
`
	defaultCfgFile, err := os.Create(filepath.Join(basedir, "my.cnf"))
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(defaultCfgFile, configFmt, basedir, datadir, user)
	defaultCfgFile.Close()

	args := []string{
		fmt.Sprintf("--defaults-file=%s", defaultCfgFile.Name()),
		"--initialize-insecure",
	}

	cmd := exec.Command(filepath.Join(basedir, "bin", "mysqld"), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to initialize mysql, error: %w", err)
	}

	return &Instance{
		basedir: basedir,
		datadir: datadir,
	}, nil
}

// SetupTestInstance installs and starts a mysql instance for testing,
// returns the instance and the stop function.
func SetupTestInstance(t *testing.T, port int) (*Instance, func()) {
	basedir, datadir := t.TempDir(), t.TempDir()
	t.Log("Installing Mysql...")
	i, err := Install(basedir, datadir, "root")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Starting Mysql...")
	if err := i.Start(port, os.Stdout, os.Stderr); err != nil {
		t.Fatal(err)
	}

	stopFn := func() {
		t.Log("Stopping Mysql...")
		if err := i.Stop(os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
	}

	return i, stopFn
}

// Import executes sql script in the given path on the instance.
// If the path is a directory, it imports all sql scripts in the directory recursively.
func (i *Instance) Import(path string, stdout, stderr io.Writer) error {
	var buf bytes.Buffer
	if err := cat(path, &buf); err != nil {
		return err
	}

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/?multiStatements=true", i.port))
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
