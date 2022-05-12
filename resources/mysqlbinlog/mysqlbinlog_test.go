package mysqlbinlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bytebase/bytebase/resources/utils"
	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether mysqlbinlog can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	var tarName string
	var version string

	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlbinlog-8.0.28-macos11-arm64.tar.gz"
		version = "mysqlbinlog-8.0.28-macos11-arm64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64.tar.gz"
		version = "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64"
	}

	tarF, err := resources.Open(tarName)
	require.NoError(t, err)

	defer tarF.Close()

	tmpDir := t.TempDir()
	err = utils.ExtractTarGz(tarF, tmpDir)
	require.NoError(t, err)

	mysqlbinlogPath := filepath.Join(tmpDir, version, "bin", "mysqlbinlog")
	cmd := exec.Command(mysqlbinlogPath, "-V")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	require.NoError(t, err)
}
