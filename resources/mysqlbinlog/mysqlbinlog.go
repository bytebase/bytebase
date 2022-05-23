package mysqlbinlog

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bytebase/bytebase/resources/utils"
)

var bin *mysqlBinlog

// mysqlBinlog involve the path of mysqlbinlog binary.
type mysqlBinlog struct {
	binPath string
}

// GetBinPath return the mysqlbinlog binary path.
func GetBinPath() string {
	return getMySQLBinlog().binPath
}

// getMySQLBinlog returns the MySQLBinlog singleton.
func getMySQLBinlog() *mysqlBinlog {
	if bin == nil {
		bin = &mysqlBinlog{
			binPath: "UNKNOWN",
		}
	}
	return bin
}

// Install will extract the mysqlbinlog tar in resourceDir.
func Install(resourceDir string) error {
	var tarName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlbinlog-8.0.28-macos11-arm64.tar.gz"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64.tar.gz"
	default:
		return fmt.Errorf("Unsupported combination of OS[%s] and ARCH[%s]", runtime.GOOS, runtime.GOARCH)
	}

	version := strings.TrimRight(tarName, "tar.gz")
	mysqlbinlogDir := path.Join(resourceDir, version)
	if _, err := os.Stat(mysqlbinlogDir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check binary directory path %q, error: %w", mysqlbinlogDir, err)
		}
		// Install if not exist yet
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
		if err := os.RemoveAll(tmpDir); err != nil {
			return fmt.Errorf("failed to remove mysqlbinlog binary temp directory %q, error: %w", tmpDir, err)
		}

		f, err := resources.Open(tarName)
		if err != nil {
			return fmt.Errorf("failed to find %q in embedded resources, error: %v", tarName, err)
		}
		defer f.Close()

		if err := utils.ExtractTarGz(f, tmpDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz file, error: %w", err)
		}

		if err := os.Rename(tmpDir, mysqlbinlogDir); err != nil {
			return fmt.Errorf("failed to rename mysqlbinlog binary directory from %q to %q, error: %w", tmpDir, mysqlbinlogDir, err)
		}
	}

	bin = &mysqlBinlog{
		binPath: filepath.Join(mysqlbinlogDir, version, "bin", "mysqlbinlog"),
	}
	return nil
}
