package mysqlbinlog

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether mysqlbinlog can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	a := require.New(t)
	tmpDir := t.TempDir()
	err := Install(tmpDir)
	a.NoError(err)

	bin := GetMySQLBinlog()
	cmd := exec.Command(bin.GetPath(), "-V")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	a.NoError(err)
}
