// Package mysqlutil provides the resource for MySQL utility packages.
package mysqlutil

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/resources/utils"
)

type binaryName string

const (
	// MySQL is the name of mysql binary.
	MySQL binaryName = "mysql"
	// MySQLBinlog is the name of mysqlbinlog binary.
	MySQLBinlog binaryName = "mysqlbinlog"
	// MySQLDump is the name of mysqldump binary.
	MySQLDump binaryName = "mysqldump"
)

// GetPath returns the binary path specified by `binName`, `binDir` is the path that installed the mysqlutil.
func GetPath(binName binaryName, binDir string) string {
	switch binName {
	case MySQL:
		return filepath.Join(binDir, "mysql")
	case MySQLBinlog:
		return filepath.Join(binDir, "mysqlbinlog")
	case MySQLDump:
		return filepath.Join(binDir, "mysqldump")
	}
	return "UNKNOWN_BINARY"
}

func getTarNameAndVersion() (tarname string, version string, err error) {
	var tarName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		tarName = "mysqlutil-8.0.33-darwin-amd64.tar.gz"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlutil-8.0.33-darwin-arm64.tar.gz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlutil-8.0.33-linux-amd64.tar.gz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		tarName = "mysqlutil-8.0.33-linux-arm64.tar.gz"
	default:
		return "", "", errors.Errorf("unsupported combination of OS %q and ARCH %q", runtime.GOOS, runtime.GOARCH)
	}
	return tarName, strings.TrimSuffix(tarName, "tar.gz"), nil
}

// Install will extract the mysqlutil tar in resourceDir.
// Returns the bin directory on success.
func Install(resourceDir string) (string, error) {
	tarName, version, err := getTarNameAndVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get tarball name and version")
	}

	mysqlutilDir := path.Join(resourceDir, version)
	if _, err := os.Stat(mysqlutilDir); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to check binary directory path %q", mysqlutilDir)
		}
		// Install if not exist yet
		log.Info("Installing MySQL utilities...")
		if err := installImpl(resourceDir, mysqlutilDir, tarName, version); err != nil {
			return "", errors.Wrap(err, "cannot install mysqlutil")
		}
	}

	return path.Join(mysqlutilDir, "bin"), nil
}

// installImpl installs mysqlutil in resourceDir.
func installImpl(resourceDir, mysqlutilDir, tarName, version string) error {
	tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpDir); err != nil {
		return errors.Wrapf(err, "failed to remove mysqlutil binaries temp directory %q", tmpDir)
	}

	f, err := resources.Open(tarName)
	if err != nil {
		return errors.Wrapf(err, "failed to find %q in embedded resources", tarName)
	}
	defer f.Close()

	if err := utils.ExtractTarGz(f, tmpDir); err != nil {
		return errors.Wrap(err, "failed to extract tar.gz file")
	}

	if err := os.Rename(tmpDir, mysqlutilDir); err != nil {
		return errors.Wrapf(err, "failed to rename mysqlutil binaries directory from %q to %q", tmpDir, mysqlutilDir)
	}

	return nil
}
