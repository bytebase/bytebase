package mongoutil

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunBinary tests whether all binarys can be started on the target platform.
func TestRunBinary(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	tmpDir := t.TempDir()
	binDir, err := Install(tmpDir)
	a.NoError(err)
	expectedVersion := `1.6.1`
	cmd := exec.Command(GetMongoshPath(binDir), "--version")
	out, err := cmd.CombinedOutput()
	a.NoError(err)
	a.Equal(strings.TrimSuffix(string(out), "\n"), expectedVersion)
}
