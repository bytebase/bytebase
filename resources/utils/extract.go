package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// ExtractTarGz extracts the given file as .tar.gz format to the given directory.
func ExtractTarGz(tarGzF io.Reader, targetDir string) error {
	gzipR, err := gzip.NewReader(tarGzF)
	if err != nil {
		return err
	}
	defer gzipR.Close()
	return extractTar(gzipR, targetDir)
}

// ExtractTarXz extracts the given file as .tar.xz or .txz format to the given directory.
func ExtractTarXz(tarXzF io.Reader, targetDir string) error {
	xzR, err := xz.NewReader(tarXzF)
	if err != nil {
		return err
	}
	return extractTar(xzR, targetDir)
}

func extractTar(r io.Reader, targetDir string) error {
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath, err := filepath.Abs(filepath.Join(targetDir, header.Name))
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %q, error: %w", header.Name, err)
		}

		// Ensure that output paths constructed from zip archive entries
		// are validated to prevent writing files to unexpected locations.
		if strings.Contains(targetPath, "..") {
			return fmt.Errorf("invalid path %q", targetPath)
		}

		if err := os.MkdirAll(path.Dir(targetPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %q, error: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type flag %d", header.Typeflag)
		}
	}

	return nil
}
