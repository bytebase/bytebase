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
)

type binaryName string

const (
	// MySQL is the name of mysql binary
	MySQL binaryName = "mysql"
	// MySQLBinlog is the name of mysqlbinlog binary
	MySQLBinlog binaryName = "mysqlbinlog"
)

// Instance involve the path of all binaries binary.
type Instance struct {
	mysqlBinPath       string
	mysqlbinlogBinPath string
}

// GetPath returns the binary path specified by `binName`.
func (ins *Instance) GetPath(binName binaryName) string {
	switch binName {
	case MySQL:
		return ins.mysqlBinPath
	case MySQLBinlog:
		return ins.mysqlbinlogBinPath
	}
	return "UNKNOWN"
}

// Version returns the raw output of ``binName` -V`.
func (ins *Instance) Version(binName binaryName) (string, error) {
	var cmd *exec.Cmd
	var version bytes.Buffer
	switch binName {
	case MySQL:
		cmd = exec.Command(ins.GetPath(MySQL), "-V")
	case MySQLBinlog:
		cmd = exec.Command(ins.GetPath(MySQLBinlog), "-V")
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

// Install will extract the mysqlutil tar in resourceDir.
func Install(resourceDir string) (*Instance, error) {
	var tarName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlutil-8.0.28-macos11-arm64.tar.gz"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		tarName = "mysqlutil-8.0.28-macos11-x86_64.tar.gz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlutil-8.0.28-linux-glibc2.17-x86_64.tar.gz"
	default:
		return nil, fmt.Errorf("unsupported combination of OS %q and ARCH %q", runtime.GOOS, runtime.GOARCH)
	}

	version := strings.TrimRight(tarName, "tar.gz")
	mysqlutilDir := path.Join(resourceDir, version)
	if _, err := os.Stat(mysqlutilDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check binary directory path %q, error: %w", mysqlutilDir, err)
		}
		// Install if not exist yet
		if err := install(resourceDir, mysqlutilDir, tarName, version); err != nil {
			return nil, fmt.Errorf("cannot install mysqlutil, error: %w", err)
		}
	}

	// TODO(zp): remove this block in the next few months
	// We change Dockerfile's base image from alpine to debian slim in v1.2.1,
	// and add libncurses.so.5 and libtinfo.so.5 to make mysqlbinlog run successfully.
	// So we need to reinstall mysqlutil if the users upgrade from an older version that missing these shared libraries.
	libncursesPath := path.Join(mysqlutilDir, "lib", "private", "libncurses.so.5")
	if _, err := os.Stat(libncursesPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check libncurses library path %q, error: %w", libncursesPath, err)
		}
		// Remove mysqlutil of old version and reinstall it
		if err := os.RemoveAll(mysqlutilDir); err != nil {
			return nil, fmt.Errorf("failed to remove the old version mysqlutil binary directory %q, error: %w", mysqlutilDir, err)
		}
		// Install the current version
		if err := install(resourceDir, mysqlutilDir, tarName, version); err != nil {
			return nil, fmt.Errorf("cannot install mysqlutil, error: %w", err)
		}
	}

	return &Instance{
		mysqlBinPath:       filepath.Join(mysqlutilDir, "bin", "mysql"),
		mysqlbinlogBinPath: filepath.Join(mysqlutilDir, "bin", "mysqlbinlog"),
	}, nil
}

// install installs mysqlutil in resourceDir
func install(resourceDir, mysqlutilDir, tarName, version string) error {
	tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("failed to remove mysqlutil binaries temp directory %q, error: %w", tmpDir, err)
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
		return fmt.Errorf("failed to rename mysqlutil binaries directory from %q to %q, error: %w", tmpDir, mysqlutilDir, err)
	}

	return nil
}
