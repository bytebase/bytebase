package resources

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	// Embeded postgres binaries.
	_ "embed"
)

// PostgresInstance is a postgres instance installed by bytebase
// for backend storage or testing.
type PostgresInstance struct {
	// basedir is the directory where the postgres binary is installed.
	basedir string
	// datadir is the directory where the postgres data is stored.
	datadir string
	// port is the port number of the postgres instance.
	port int
}

// Port returns the port number of the postgres instance.
func (i PostgresInstance) Port() int { return i.port }

// Start starts a postgres instance on given port, outputs to stdout and stderr.
// If port is 0, then it will choose a random unused port.
func (i *PostgresInstance) Start(port int, stdout, stderr io.Writer) (err error) {
	pgbin := filepath.Join(i.basedir, "bin", "pg_ctl")

	i.port = port
	if port == 0 {
		if i.port, err = randomUnusedPort(); err != nil {
			return fmt.Errorf("Cannot find a random port: %v", err)
		}
	}

	p := exec.Command(pgbin, "start", "-w",
		"-D", i.datadir,
		"-o", fmt.Sprintf(`"-p %d"`, i.port))
	p.Stdout = stdout
	p.Stderr = stderr

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to start postgres %q, error %v", p.String(), err)
	}

	return nil
}

// Stop stops a postgres instance, outputs to stdout and stderr.
func (i *PostgresInstance) Stop(stdout, stderr io.Writer) error {
	pgbin := filepath.Join(i.basedir, "bin", "pg_ctl")
	p := exec.Command(pgbin, "stop", "-w",
		"-D", i.datadir)
	p.Stderr = stderr
	p.Stdout = stdout

	if err := p.Run(); err != nil {
		return err
	}

	return nil
}

// InstallPostgres returns the postgres binary depending on the OS.
func InstallPostgres(resourceDir, pgDataDir, pgUser string) (*PostgresInstance, error) {
	var tarName string
	switch runtime.GOOS {
	case "darwin":
		tarName = "postgres-darwin-x86_64.txz"
	case "linux":
		if isAlpineLinux() {
			tarName = "postgres-linux-x86_64-alpine_linux.txz"
		} else {
			tarName = "postgres-linux-x86_64.txz"
		}
	default:
		return nil, fmt.Errorf("OS %q is not supported", runtime.GOOS)
	}
	log.Printf("Installing Postgres OS %q Arch %q txz %q\n", runtime.GOOS, runtime.GOARCH, tarName)
	f, err := postgresResources.Open(tarName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	version := strings.TrimRight(tarName, ".txz")

	pgBinDir := path.Join(resourceDir, version)
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
		if err := extractTarXz(f, tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract txz file, error: %w", err)
		}
		if err := os.Rename(tmpDir, pgBinDir); err != nil {
			return nil, fmt.Errorf("failed to rename postgres binary directory from %q to %q, error: %w", tmpDir, pgBinDir, err)
		}
	}

	if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return nil, err
	}

	return &PostgresInstance{
		basedir: pgBinDir,
		datadir: pgDataDir,
	}, nil
}

func isAlpineLinux() bool {
	_, err := os.Stat("/etc/alpine-release")
	return err == nil
}

// initDB inits a postgres database.
func initDB(pgBinDir, pgDataDir, pgUser string) error {
	_, err := os.Stat(pgDataDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check data directory path %q, error: %w", pgDataDir, err)
	}
	// Skip initDB if setup already.
	if err == nil {
		return nil
	}

	if err := os.MkdirAll(pgDataDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to make postgres data directory %q, error: %w", pgDataDir, err)
	}

	args := []string{
		"-U", pgUser,
		"-D", pgDataDir,
	}
	initDBBinary := filepath.Join(pgBinDir, "bin", "initdb")
	p := exec.Command(initDBBinary, args...)
	p.Stderr = os.Stderr
	p.Stdout = os.Stdout

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to initdb %q, error %v", p.String(), err)
	}

	return nil
}
