package mysqlbinlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether mysqlbinlog can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	a := require.New(t)
	tmpDir := t.TempDir()
	mysqlbinlogDir, err := Install(tmpDir)
	a.NoError(err)

	mysqlbinlogPath := filepath.Join(mysqlbinlogDir, "mysqlbinlog")
	cmd := exec.Command(mysqlbinlogPath, "-V")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	a.NoError(err)
}
