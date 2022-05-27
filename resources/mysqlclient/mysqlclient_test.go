package mysqlclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether mysql can be started on the target platform
// to check whether the lib extraction is correct.
func TestRunBinary(t *testing.T) {
	a := require.New(t)
	tmpDir := t.TempDir()
	ins, err := Install(tmpDir)
	a.NoError(err)

	_, err = ins.Version()
	a.NoError(err)
}
