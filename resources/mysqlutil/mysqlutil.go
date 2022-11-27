// Package mysqlutil provides the resource for MySQL utility packages.
package mysqlutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/resources/utils"
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

// getExecutableVersion returns the raw output of "binName -V".
func getExecutableVersion(binName binaryName, binDir string) (string, error) {
	var cmd *exec.Cmd
	var v bytes.Buffer
	switch binName {
	case MySQL:
		cmd = exec.Command(GetPath(MySQL, binDir), "-V")
	case MySQLBinlog:
		cmd = exec.Command(GetPath(MySQLBinlog, binDir), "-V")
	case MySQLDump:
		cmd = exec.Command(GetPath(MySQLDump, binDir), "-V")
	default:
		return "", errors.Errorf("unknown binary name: %s", binName)
	}

	cmd.Stdout = &v
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return v.String(), nil
}

func getTarNameAndVersion() (tarname string, version string, err error) {
	var tarName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlutil-8.0.28-macos11-arm64.tar.gz"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		tarName = "mysqlutil-8.0.28-macos11-x86_64.tar.gz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlutil-8.0.28-linux-glibc2.17-x86_64.tar.gz"
	default:
		return "", "", errors.Errorf("unsupported combination of OS %q and ARCH %q", runtime.GOOS, runtime.GOARCH)
	}
	return tarName, strings.TrimRight(tarName, "tar.gz"), nil
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
		if err := installImpl(resourceDir, mysqlutilDir, tarName, version); err != nil {
			return "", errors.Wrap(err, "cannot install mysqlutil")
		}
	}

	checks := []string{
		// TODO(zp): remove this line in the next few months
		// We change Dockerfile's base image from alpine to debian slim in v1.2.1,
		// and add libncurses.so.5 and libtinfo.so.5 to make mysqlbinlog run successfully.
		// So we need to reinstall mysqlutil if the users upgrade from an older version that missing these shared libraries.
		path.Join(mysqlutilDir, "lib", "private", "libncurses.so.5"),
		path.Join(mysqlutilDir, "lib", "private", "libtinfo.so.5"),

		// We embed mysqldump in version v1.2.3.
		path.Join(mysqlutilDir, "bin", "mysqldump"),
	}

	for _, fp := range checks {
		if _, err := os.Stat(fp); err != nil {
			if !os.IsNotExist(err) {
				return "", errors.Wrapf(err, "failed to check libncurses library path %q", fp)
			}
			// Remove mysqlutil of old version and reinstall it
			if err := os.RemoveAll(mysqlutilDir); err != nil {
				return "", errors.Wrapf(err, "failed to remove the old version mysqlutil binary directory %q", mysqlutilDir)
			}
			// Install the current version
			if err := installImpl(resourceDir, mysqlutilDir, tarName, version); err != nil {
				return "", errors.Wrap(err, "cannot install mysqlutil")
			}
			break
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
