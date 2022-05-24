package mysqlbinlog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether mysqlbinlog can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	a := require.New(t)
	tmpDir := t.TempDir()
	ins, err := Install(tmpDir)
	a.NoError(err)

	version, err := ins.Version()
	a.NoError(err)
	fmt.Println(version)
}
