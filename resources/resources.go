package resources

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
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

// ExtractPostgresBinary returns the postgres binary depending on the OS.
func ExtractPostgresBinary(directory string) error {
	var data []byte
	switch runtime.GOOS {
	case "darwin":
		data = postgresDarwin
	case "linux":
		data = postgresLinux
	default:
		return fmt.Errorf("OS %q is not supported", runtime.GOOS)
	}

	if err := extractTXZ(directory, data); err != nil {
		return fmt.Errorf("failed to extract txz file")
	}
	return nil
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
