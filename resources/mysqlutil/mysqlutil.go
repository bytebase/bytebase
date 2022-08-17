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

	"github.com/bytebase/bytebase/resources/utils"
	"github.com/pkg/errors"
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

// GetPath returns the binary path specified by `binName`, `resourceDir` is the path that installed the mysqlutil.
func GetPath(binName binaryName, resourceDir string) string {
	_, version, err := getTarNameAndVersion()
	if err != nil {
		return "UNEXPECTED_ERROR"
	}
	baseDir := filepath.Join(resourceDir, version)
	switch binName {
	case MySQL:
		return filepath.Join(baseDir, "bin", "mysql")
	case MySQLBinlog:
		return filepath.Join(baseDir, "bin", "mysqlbinlog")
	case MySQLDump:
		return filepath.Join(baseDir, "bin", "mysqldump")
	}
	return "UNKNOWN_BINARY"
}

// getExecutableVersion returns the raw output of ``binName` -V`.
func getExecutableVersion(binName binaryName, resourceDir string) (string, error) {
	var cmd *exec.Cmd
	var version bytes.Buffer
	switch binName {
	case MySQL:
		cmd = exec.Command(GetPath(MySQL, resourceDir), "-V")
	case MySQLBinlog:
		cmd = exec.Command(GetPath(MySQLBinlog, resourceDir), "-V")
	case MySQLDump:
		cmd = exec.Command(GetPath(MySQLDump, resourceDir), "-V")
	default:
		return "", fmt.Errorf("unknown binary name: %s", binName)
	}

	cmd.Stdout = &version
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return version.String(), nil
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
		return "", "", fmt.Errorf("unsupported combination of OS %q and ARCH %q", runtime.GOOS, runtime.GOARCH)
	}
	return tarName, strings.TrimRight(tarName, "tar.gz"), nil
}

// Install will extract the mysqlutil tar in resourceDir.
func Install(resourceDir string) error {
	tarName, version, err := getTarNameAndVersion()
	if err != nil {
		return fmt.Errorf("failed to get tarball name and version, error: %w", err)
	}

	mysqlutilDir := path.Join(resourceDir, version)

	if _, err := os.Stat(mysqlutilDir); err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to check binary directory path %q", mysqlutilDir)
		}
		// Install if not exist yet
		if err := installImpl(resourceDir, mysqlutilDir, tarName, version); err != nil {
			return fmt.Errorf("cannot install mysqlutil, error: %w", err)
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
				return errors.Wrapf(err, "failed to check libncurses library path %q", fp)
			}
			// Remove mysqlutil of old version and reinstall it
			if err := os.RemoveAll(mysqlutilDir); err != nil {
				return errors.Wrapf(err, "failed to remove the old version mysqlutil binary directory %q", mysqlutilDir)
			}
			// Install the current version
			if err := installImpl(resourceDir, mysqlutilDir, tarName, version); err != nil {
				return fmt.Errorf("cannot install mysqlutil, error: %w", err)
			}
			break
		}
	}

	return nil
}

// installImpl installs mysqlutil in resourceDir.
func installImpl(resourceDir, mysqlutilDir, tarName, version string) error {
	tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpDir); err != nil {
		return errors.Wrapf(err, "failed to remove mysqlutil binaries temp directory %q", tmpDir)
	}

	f, err := resources.Open(tarName)
	if err != nil {
		return fmt.Errorf("failed to find %q in embedded resources, error: %v", tarName, err)
	}
	defer f.Close()

	if err := utils.ExtractTarGz(f, tmpDir); err != nil {
		return fmt.Errorf("failed to extract tar.gz file, error: %w", err)
	}

	if err := os.Rename(tmpDir, mysqlutilDir); err != nil {
		return errors.Wrapf(err, "failed to rename mysqlutil binaries directory from %q to %q", tmpDir, mysqlutilDir)
	}

	return nil
}
