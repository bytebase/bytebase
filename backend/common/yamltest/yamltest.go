// Package yamltest provides yaml encoding helpers tuned for bytebase test
// fixtures. The encoder is pinned to 2-space indentation to match the
// existing on-disk format and avoid noisy diffs when test data is
// re-recorded.
//
// yaml.v3's default Marshal uses 4-space sequence indentation, which differs
// from the 2-space style used in bytebase's checked-in test fixtures. Every
// re-record under the default produces hundreds of lines of pure whitespace
// churn. The helpers here wrap yaml.NewEncoder with SetIndent(2) so that
// re-recording is a no-op when the test data has not changed.
//
// Other yaml.v3 behaviors (block-scalar collapse, comment loss, blank-line
// removal) are inherent to text-based marshalling and are not handled here.
package yamltest

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// NewEncoder returns a yaml.Encoder configured with 2-space indentation.
// Use directly when callers need custom buffering; most record-mode tests
// should use Record or WriteFile instead.
func NewEncoder(w io.Writer) *yaml.Encoder {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc
}

// WriteFile marshals v with 2-space indentation and writes it to path,
// returning any error. Use from helpers that propagate errors instead of
// failing tests directly (e.g., backend/tests integration helpers).
func WriteFile(path string, v any) error {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// Record marshals v with 2-space indentation and writes it to path,
// failing the test on any error. Use in test record-mode branches.
func Record(t *testing.T, path string, v any) {
	t.Helper()
	require.NoError(t, WriteFile(path, v))
}
