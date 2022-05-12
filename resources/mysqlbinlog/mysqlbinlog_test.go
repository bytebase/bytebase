package mysqlbinlog

import (
	"fmt"
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
	a := require.New(t)

	var tarName string
	var version string
	fmt.Println(runtime.GOOS, runtime.GOARCH)
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysqlbinlog-8.0.28-macos11-arm64.tar.gz"
		version = "mysqlbinlog-8.0.28-macos11-arm64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64.tar.gz"
		version = "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64"
	default:
		t.Logf("Unsupported combination of OS[%s] and ARCH[%s]", runtime.GOOS, runtime.GOARCH)
		t.Fail()
	}

	tarF, err := resources.Open(tarName)
	a.NoError(err)

	defer tarF.Close()

	tmpDir := t.TempDir()
	err = utils.ExtractTarGz(tarF, tmpDir)
	a.NoError(err)

	mysqlbinlogPath := filepath.Join(tmpDir, version, "bin", "mysqlbinlog")
	cmd := exec.Command(mysqlbinlogPath, "-V")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	a.NoError(err)
}
