package mysqlutil

import (
	"os"
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
	err := Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client", func(t *testing.T) {
		_, err := getExecutableVersion(MySQL, tmpDir)
		a.NoError(err)
	})

	t.Run("run mysqlbinlog", func(t *testing.T) {
		_, err := getExecutableVersion(MySQLBinlog, tmpDir)
		a.NoError(err)
	})

	t.Run("run mysqldump", func(t *testing.T) {
		_, err := getExecutableVersion(MySQLDump, tmpDir)
		a.NoError(err)
	})
}

// TODO(zp): remove this test when remove the related block in mysqlutil.go.
// TestReinstallOnLinuxAmd64 tests is it possible to reinstall mysqlutil on linux amd64.
func TestReinstallOnLinuxAmd64(t *testing.T) {
	t.Parallel()

	if !(runtime.GOOS == "linux" && runtime.GOARCH == "amd64") {
		return
	}

	a := require.New(t)
	tmpDir := t.TempDir()
	err := Install(tmpDir)
	a.NoError(err)

	baseDir := filepath.Join(tmpDir, "mysqlutil-8.0.28-linux-glibc2.17-x86_64" /*Hard code, don't care about this*/)
	binDir := filepath.Join(baseDir, "bin")
	libDir := filepath.Join(baseDir, "lib", "private")

	checks := []string{
		filepath.Join(libDir, "libncurses.so.5"),
		filepath.Join(libDir, "libtinfo.so.5"),
		filepath.Join(binDir, "mysqldump"),
	}

	mysqlPath := GetPath(MySQL, tmpDir)

	for _, fp := range checks {
		a.FileExists(fp)

		err = os.RemoveAll(fp)
		a.NoError(err)
		a.NoFileExists(fp)

		err = Install(tmpDir)
		a.NoError(err)
		a.FileExists(fp)
		a.FileExists(mysqlPath)
	}
}
