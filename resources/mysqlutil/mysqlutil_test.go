package mysqlutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether all binarys can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	tmpDir := t.TempDir()
	ins, err := Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client", func(t *testing.T) {
		_, err := ins.Version(MySQL)
		a.NoError(err)
	})

	t.Run("run mysqlbinlog", func(t *testing.T) {
		_, err := ins.Version(MySQLBinlog)
		a.NoError(err)
	})
}

// TODO(zp): remove this test when remove the related block in mysqlutil.go.
// TestReinstallOnLinuxAmd64 tests is it possible to reinstall mysqlutil on Linux amd64.
func TestReinstallOnLinuxAmd64(t *testing.T) {
	t.Parallel()

	if !(runtime.GOOS == "linux" && runtime.GOARCH == "amd64") {
		return
	}

	a := require.New(t)

	tmpDir := t.TempDir()
	mysqlutilInstance, err := Install(tmpDir)
	a.NoError(err)

	tryRunMySQLClient := func(a *require.Assertions, expectedRunErr bool) {
		cmd := exec.Command(mysqlutilInstance.GetPath(MySQL), "--version")

		stderr, err := cmd.StderrPipe()
		a.NoError(err)
		defer stderr.Close()

		err = cmd.Run()
		if expectedRunErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}

	t.Run("run mysql client", func(t *testing.T) {
		a := require.New(t)
		tryRunMySQLClient(a, false)
	})

	err = os.Remove(filepath.Join(tmpDir, "mysqlutil-8.0.28-linux-glibc2.17-x86_64", /*Hard code, don't care about this*/
		"lib", "private", "libncurses.so.5"))
	a.NoError(err)

	t.Run("run mysql client after drop libncurses.so.5", func(t *testing.T) {
		a := require.New(t)
		tryRunMySQLClient(a, true)
	})

	mysqlutilInstance, err = Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client after resinstall", func(t *testing.T) {
		a := require.New(t)
		tryRunMySQLClient(a, false)
	})
}
