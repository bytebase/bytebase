package resources

import (
	"archive/tar"
	"bytes"
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

	"github.com/xi2/xz"
)

// InstallPostgres returns the postgres binary depending on the OS.
func InstallPostgres(resourceDir, pgDataDir, pgUser string) (string, error) {
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
		return "", fmt.Errorf("OS %q is not supported", runtime.GOOS)
	}
	log.Printf("Installing Postgres OS %q Arch %q txz %q\n", runtime.GOOS, runtime.GOARCH, tarName)
	data, err := postgresResources.ReadFile(tarName)
	if err != nil {
		return "", err
	}
	version := strings.TrimRight(tarName, ".txz")

	pgBinDir := path.Join(resourceDir, version)
	if _, err := os.Stat(pgBinDir); err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to check binary directory path %q, error: %w", pgBinDir, err)
		}
		// Install if not exist yet.
		// The ordering below made Postgres installation atomic.
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
		if err := os.RemoveAll(tmpDir); err != nil {
			return "", fmt.Errorf("failed to remove postgres binary temp directory %q, error: %w", tmpDir, err)
		}
		if err := extractTXZ(tmpDir, data); err != nil {
			return "", fmt.Errorf("failed to extract txz file, error: %w", err)
		}
		if err := os.Rename(tmpDir, pgBinDir); err != nil {
			return "", fmt.Errorf("failed to rename postgres binary directory from %q to %q, error: %w", tmpDir, pgBinDir, err)
		}
	}

	if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return "", err
	}

	return pgBinDir, nil
}

func isAlpineLinux() bool {
	_, err := os.Stat("/etc/alpine-release")
	return err == nil
}

func extractTXZ(directory string, data []byte) error {
	xzReader, err := xz.NewReader(bytes.NewReader(data), 0)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(xzReader)
	reader := func() io.Reader {
		return tarReader
	}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(directory, header.Name)
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, reader()); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// We need to remove existing file and redo the symlink.
			if err := os.RemoveAll(targetPath); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		}
	}
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

// StartPostgres starts postgres process.
func StartPostgres(pgBinDir, pgdataDir string, port int, stdout, stderr *os.File) error {
	pgbin := filepath.Join(pgBinDir, "bin", "pg_ctl")
	p := exec.Command(pgbin, "start", "-w",
		"-D", pgdataDir,
		"-o", fmt.Sprintf(`"-p %d"`, port))
	p.Stdout = stdout
	p.Stderr = stderr

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to start postgres %q, error %v", p.String(), err)
	}

	return nil
}

// StopPostgres stops postgres process.
func StopPostgres(pgBinDir, pgDataDir string, stdout, stderr *os.File) error {
	pgbin := filepath.Join(pgBinDir, "bin", "pg_ctl")
	p := exec.Command(pgbin, "stop", "-w",
		"-D", pgDataDir)
	p.Stderr = stderr
	p.Stdout = stdout

	if err := p.Run(); err != nil {
		return err
	}

	return nil
}
