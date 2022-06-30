package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/stretchr/testify/require"
)

// TODO(zp): remove this test when remove the related block in mysqlutil.go.
// TestReinstallOnLinuxAmd64 tests is it possible to reinstall mysqlutil on Linux amd64.
func TestReinstallOnLinuxAmd64(t *testing.T) {
	t.Parallel()

	if !(runtime.GOOS == "linux" && runtime.GOARCH == "amd64") {
		return
	}

	port := getTestPort(t.Name())

	a := require.New(t)

	_, stopFn := resourcemysql.SetupTestInstance(t, port)
	defer stopFn()

	connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "mysql")

	tmpDir := t.TempDir()
	mysqlutilInstance, err := mysqlutil.Install(tmpDir)
	a.NoError(err)

	tryRunMySQLClient := func(a *require.Assertions, expectedRunErr bool) {
		cmd := exec.Command(mysqlutilInstance.GetPath(mysqlutil.MySQL), []string{
			"-u",
			connCfg.Username,
			"-h",
			connCfg.Host,
			"-P",
			connCfg.Port,
		}...)

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

	mysqlutilInstance, err = mysqlutil.Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client after resinstall", func(t *testing.T) {
		a := require.New(t)
		tryRunMySQLClient(a, false)
	})
}
