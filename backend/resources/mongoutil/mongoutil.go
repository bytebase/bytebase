// Package mongoutil provides the resource for MongoDB utility packages.
package mongoutil

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/resources/utils"
)

// nolint
// GetMongoshPath returns the mongosh path.
func GetMongoshPath(binDir string) string {
	return path.Join(binDir, "mongosh")
}

// Install installs mongoutil in resourceDir.
func Install(resourceDir string) (string, error) {
	tarName, version, err := utils.GetTarNameAndVersion("mongoutil", ".txz")
	if err != nil {
		return "", err
	}
	mongoutilDir := path.Join(resourceDir, version)
	if _, err := os.Stat(mongoutilDir); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to check binary directory path %q", mongoutilDir)
		}
		// Install if not exist yet
		log.Info("Installing MongoDB utilities, it may take about several minutes...")
		if err := installImpl(resourceDir, mongoutilDir, tarName, version); err != nil {
			return "", errors.Wrap(err, "cannot install mongoutil")
		}
	}
	return path.Join(mongoutilDir, "bin"), nil
}

// installImpl installs mongoutil in resourceDir.
func installImpl(resourceDir, mongoutilDir, tarName, version string) error {
	tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpDir); err != nil {
		return errors.Wrapf(err, "failed to remove mysqlutil binaries temp directory %q", tmpDir)
	}

	f, err := resources.Open(tarName)
	if err != nil {
		return errors.Wrapf(err, "failed to find %q in embedded resources", tarName)
	}
	defer f.Close()

	if err := utils.ExtractTarXz(f, tmpDir); err != nil {
		return errors.Wrap(err, "failed to extract tar.gz file")
	}

	if err := os.Rename(tmpDir, mongoutilDir); err != nil {
		return errors.Wrapf(err, "failed to rename mongoutil binaries directory from %q to %q", tmpDir, mongoutilDir)
	}

	return nil
}
