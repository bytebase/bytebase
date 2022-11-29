package helm

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArtifactHub tests our artifact metadata exists.
// We provide the chart template via GitHub Pages.
func TestArtifactHub(t *testing.T) {
	a := require.New(t)
	indexURL := "https://bytebase.github.io/bytebase/index.yaml"
	resp, err := http.Get(indexURL)
	a.NoError(err)
	a.Equal(http.StatusOK, resp.StatusCode)
}
