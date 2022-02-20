package resources

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	// Embeded postgres binaries.
	_ "embed"

	"github.com/xi2/xz"
)

//go:embed postgres-darwin-x86_64.txz
var postgresDarwin []byte

//go:embed postgres-linux-x86_64.txz
var postgresLinux []byte

// InstallPostgres returns the postgres binary depending on the OS.
func InstallPostgres(resourceDir, dataDir string) (string, error) {
	var version string
	var data []byte
	switch runtime.GOOS {
	case "darwin":
		version, data = "postgres-darwin-amd64-14.2.0", postgresDarwin
	case "linux":
		version, data = "postgres-linux-amd64-14.2.0", postgresLinux
	default:
		return "", fmt.Errorf("OS %q is not supported", runtime.GOOS)
	}

	// Skip installation if installed already.
	_, err := os.Stat(dataDir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check data directory path %q, error: %w", dataDir, err)
	}

	// The ordering below made Postgres installation atomic.
	pgBinDir := path.Join(resourceDir, version)
	if err == nil {
		return pgBinDir, nil
	}

	tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpDir); err != nil {
		return "", fmt.Errorf("failed to remove postgres binary temp directory %q, error: %w", tmpDir, err)
	}
	if err := extractTXZ(tmpDir, data); err != nil {
		return "", fmt.Errorf("failed to extract txz file")
	}
	if err := os.Rename(tmpDir, pgBinDir); err != nil {
		return "", fmt.Errorf("failed to rename postgres binary directory from %q to %q, error: %w", tmpDir, pgBinDir, err)
	}
	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to make postgres data directory %q, error: %w", dataDir, err)
	}
	if err := initDB(pgBinDir, dataDir, "postgres"); err != nil {
		return "", err
	}
	return pgBinDir, nil
}

func extractTXZ(directory string, data []byte) error {
	xzReader, err := xz.NewReader(bytes.NewReader(data), 0)
	if err != nil {
		return nil
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
func initDB(pgBinDir, pgdataDir, username string) error {
	args := []string{
		"-U", username,
		"-D", pgdataDir,
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
