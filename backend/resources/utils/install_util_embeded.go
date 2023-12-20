//go:build !docker

package utils

import (
	"embed"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

// InstallImpl installs mongoutil, mysqlutil, postgres in resourceDir.
func InstallImpl(resourceDir, utilDir, tarName, version string, resources embed.FS) error {
	tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpDir); err != nil {
		return errors.Wrapf(err, "failed to remove util binaries temp directory %q", tmpDir)
	}

	f, err := resources.Open(tarName)
	if err != nil {
		return errors.Wrapf(err, "failed to find %q in embedded resources", tarName)
	}
	defer f.Close()

	if strings.Contains(tarName, ".txz") {
		if err := ExtractTarXz(f, tmpDir); err != nil {
			return errors.Wrap(err, "failed to extract tar.gz file")
		}
	} else {
		if err := ExtractTarGz(f, tmpDir); err != nil {
			return errors.Wrap(err, "failed to extract tar.gz file")
		}
	}

	if err := os.Rename(tmpDir, utilDir); err != nil {
		return errors.Wrapf(err, "failed to rename util binaries directory from %q to %q", tmpDir, utilDir)
	}

	return nil
}
