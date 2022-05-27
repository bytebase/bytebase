package mysqlclient

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

// Instance involve the path of mysql client binary.
type Instance struct {
	binPath string
}

// GetPath returns the mysqlbinlog binary path.
func (ins *Instance) GetPath() string {
	return ins.binPath
}

// Version returns the raw output of `mysql -V`
func (ins *Instance) Version() (string, error) {
	var version bytes.Buffer

	cmd := exec.Command(ins.GetPath(), "-V")
	cmd.Stdout = &version
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return version.String(), nil
}

// Install will extract the mysqlbinlog tar in resourceDir.
func Install(resourceDir string) (*Instance, error) {
	var tarName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlclient-8.0.28-macos11-arm64.tar.gz"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		tarName = "mysqlclient-8.0.28-macos11-x86_64.tar.gz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlclient-8.0.28-linux-glibc-2.17-x86_64.tar.gz"
	default:
		return nil, fmt.Errorf("Unsupported combination of OS[%s] and ARCH[%s]", runtime.GOOS, runtime.GOARCH)
	}

	version := strings.TrimRight(tarName, "tar.gz")
	mysqlbinlogDir := path.Join(resourceDir, version)
	if _, err := os.Stat(mysqlbinlogDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check binary directory path %q, error: %w", mysqlbinlogDir, err)
		}
		// Install if not exist yet
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
		if err := os.RemoveAll(tmpDir); err != nil {
			return nil, fmt.Errorf("failed to remove mysqlbinlog binary temp directory %q, error: %w", tmpDir, err)
		}

		f, err := resources.Open(tarName)
		if err != nil {
			return nil, fmt.Errorf("failed to find %q in embedded resources, error: %v", tarName, err)
		}
		defer f.Close()

		if err := utils.ExtractTarGz(f, tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract tar.gz file, error: %w", err)
		}

		if err := os.Rename(tmpDir, mysqlbinlogDir); err != nil {
			return nil, fmt.Errorf("failed to rename mysqlbinlog binary directory from %q to %q, error: %w", tmpDir, mysqlbinlogDir, err)
		}
	}
	return &Instance{
		binPath: filepath.Join(mysqlbinlogDir, "bin", "mysql"),
	}, nil
}
