// Package utils provides the library for installing resources..
package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/xi2/xz"
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
	xzR, err := xz.NewReader(tarXzF, 0)
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
			return errors.Wrapf(err, "failed to get absolute path for %q", header.Name)
		}

		// Ensure that output paths constructed from zip archive entries
		// are validated to prevent writing files to unexpected locations.
		if strings.Contains(targetPath, "..") {
			return errors.Errorf("invalid path %q", targetPath)
		}

		if err := os.MkdirAll(path.Dir(targetPath), os.ModePerm); err != nil {
			return errors.Wrapf(err, "failed to create directory %q", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			var totalWritten int64
			for totalWritten < header.Size {
				written, err := io.CopyN(outFile, tarReader, 1024)
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
				totalWritten += written
			}
			if err := outFile.Close(); err != nil {
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
			return errors.Errorf("unsupported type flag %d", header.Typeflag)
		}
	}

	return nil
}

func LinkImpl(storeDir, utilDir string) (bool, error) {
	if _, err := os.Stat(storeDir); err != nil {
		if os.IsNotExist(err) {
			// source file doesn't exists.'
			slog.Info("storeDir does not exists")
			return false, nil
		}
	}
	if utilDir == storeDir {
		// they are same, just use it.
		return true, nil
	}

	// they are not same, create a symbolic link.
	if err := os.Symlink(storeDir, utilDir); err != nil {
		// panic if failed to create symbolic link
		return false, errors.Wrapf(err, "failed to create a symbolic link for util")
	}

	// create symbolic link success
	return true, nil
}
