package helm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArtifactHub tests our artifact metadata exists.
// We provide the chart template via GitHub Pages.
func TestArtifactHub(t *testing.T) {
	metadataFilePaths := []string{
		"../docs/index.yaml",
		"../docs/bytebase-1.1.1.tgz",
	}
	a := require.New(t)
	for _, path := range metadataFilePaths {
		a.FileExists(path)
	}
}
