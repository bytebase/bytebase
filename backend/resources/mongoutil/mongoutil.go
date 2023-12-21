// Package mongoutil provides the resource for MongoDB utility packages.
package mongoutil

import (
	"log/slog"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/resources/utils"
)

// nolint
// GetMongoshPath returns the mongosh path.
func GetMongoshPath(binDir string) string {
	return path.Join(binDir, "mongosh")
}

// getTarnameAndVersion returns the mongoutil tarball name and version string.
func getTarNameAndVersion() (tarname string, version string, err error) {
	var tarName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		tarName = "mongoutil-1.6.1-darwin-amd64.txz"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mongoutil-1.6.1-darwin-arm64.txz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mongoutil-1.6.1-linux-amd64.txz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		tarName = "mongoutil-1.6.1-linux-arm64.txz"
	default:
		return "", "", errors.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return tarName, strings.TrimSuffix(tarName, ".txz"), nil
}

// Install installs mongoutil in resourceDir.
func Install(resourceDir string) (string, error) {
	tarName, version, err := getTarNameAndVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get tarball name and version")
	}

	mongoutilDir := path.Join(resourceDir, version)
	if _, err := os.Stat(mongoutilDir); err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "failed to check binary directory path %q", mongoutilDir)
		}
		// Install if not exist yet
		slog.Info("Installing MongoDB utilities, it may take about several minutes...")
		if err := utils.InstallImpl(resourceDir, mongoutilDir, tarName, version, resources); err != nil {
			return "", errors.Wrap(err, "cannot install mongoutil")
		}
	}

	return path.Join(mongoutilDir, "bin"), nil
}
