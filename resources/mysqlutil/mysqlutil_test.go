package mysqlutil

import (
	"io/ioutil"
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
// TestReinstall tests is it possible to reinstall mysqlutil.
func TestReinstallOnLinuxAmd64(t *testing.T) {
	t.Parallel()

	if !(runtime.GOOS == "linux" && runtime.GOARCH == "amd64") {
		return
	}

	a := require.New(t)
	tmpDir := t.TempDir()
	ins, err := Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client", func(t *testing.T) {
		_, err := ins.Version(MySQL)
		a.NoError(err)
	})

	err = os.Remove(filepath.Join(ins.libraryDir, "libncurses.so.5"))
	a.NoError(err)

	t.Run("run mysql client after drop libncurses.so.5", func(t *testing.T) {
		// We don't need to start mysqld, just test whether the mysql client can start normally
		cmd := exec.Command(ins.GetPath(MySQL), []string{
			"-U",
			"root",
		}...)

		stderr, err := cmd.StderrPipe()
		a.NoError(err)
		defer stderr.Close()

		err = cmd.Run()
		a.NoError(err)

		errBytes, err := ioutil.ReadAll(stderr)
		a.NoError(err)

		a.NotZero(len(errBytes))
	})

	ins, err = Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client after resinstall", func(t *testing.T) {
		_, err := ins.Version(MySQL)
		a.NoError(err)
	})
}
