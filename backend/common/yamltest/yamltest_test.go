package yamltest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// goldenYAML is the expected output for the test fixture below. It pins
// 2-space sequence indentation as the format guarantee.
const goldenYAML = `- name: alpha
  values:
    - one
    - two
- name: beta
  values:
    - three
`

func fixture() any {
	return []map[string]any{
		{"name": "alpha", "values": []string{"one", "two"}},
		{"name": "beta", "values": []string{"three"}},
	}
}

func TestNewEncoderPins2SpaceIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	require.NoError(t, enc.Encode(fixture()))
	require.NoError(t, enc.Close())
	require.Equal(t, goldenYAML, buf.String())
}

func TestWriteFileWritesGoldenBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	require.NoError(t, WriteFile(path, fixture()))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, goldenYAML, string(got))
}

func TestRecordWritesGoldenBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	Record(t, path, fixture())
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, goldenYAML, string(got))
}
