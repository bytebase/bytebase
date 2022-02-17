package resources

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
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
	binPath := path.Join(resourceDir, version)
	_, err := os.Stat(binPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check binary path %q, error: %w", binPath, err)
	}
	if err == nil {
		return binPath, nil
	}

	// The ordering below made Postgres installation atomic.
	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to make postgres data directory %q, error: %w", dataDir, err)
	}
	tmpPath := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpPath); err != nil {
		return "", fmt.Errorf("failed to remove postgres binary temp directory %q, error: %w", tmpPath, err)
	}
	if err := extractTXZ(tmpPath, data); err != nil {
		return "", fmt.Errorf("failed to extract txz file")
	}
	if err := os.Rename(tmpPath, binPath); err != nil {
		return "", fmt.Errorf("failed to rename postgres binary directory from %q to %q, error: %w", tmpPath, binPath, err)
	}
	return binPath, nil
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

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
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
