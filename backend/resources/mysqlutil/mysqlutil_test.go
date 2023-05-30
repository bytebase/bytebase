package mysqlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether all binarys can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	tmpDir := t.TempDir()
	binDir, err := Install(tmpDir)
	a.NoError(err)

	t.Run("run mysql client", func(t *testing.T) {
		_, err := getExecutableVersion(MySQL, binDir)
		a.NoError(err)
	})

	t.Run("run mysqlbinlog", func(t *testing.T) {
		_, err := getExecutableVersion(MySQLBinlog, binDir)
		a.NoError(err)
	})

	t.Run("run mysqldump", func(t *testing.T) {
		_, err := getExecutableVersion(MySQLDump, binDir)
		a.NoError(err)
	})
}
