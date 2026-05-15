package oauth2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
)

// TestProtectedResourceMetadataMatchesRequestedResource verifies the
// path-suffix routing required by RFC 9728 §3.3: the `resource` field in
// the metadata document must match the URL the client is accessing.
//
//	GET /.well-known/oauth-protected-resource         → resource: <base>
//	GET /.well-known/oauth-protected-resource/mcp     → resource: <base>/mcp
//
// The /mcp form is what the MCP server's WWW-Authenticate header points at,
// so strict clients fetch and validate the document against the same URL
// they were originally trying to reach.
func TestProtectedResourceMetadataMatchesRequestedResource(t *testing.T) {
	s := &Service{
		// ExternalURL short-circuits the workspace-lookup path so we don't
		// need a real store. This is the canonical base for the test.
		profile: &config.Profile{ExternalURL: "https://bb.example.com"},
	}

	cases := []struct {
		name         string
		path         string
		wantResource string
	}{
		{
			name:         "base path returns origin",
			path:         "/.well-known/oauth-protected-resource",
			wantResource: "https://bb.example.com",
		},
		{
			name:         "mcp suffix returns resource with /mcp",
			path:         "/.well-known/oauth-protected-resource/mcp",
			wantResource: "https://bb.example.com/mcp",
		},
		{
			name:         "trailing slash is normalized away",
			path:         "/.well-known/oauth-protected-resource/mcp/",
			wantResource: "https://bb.example.com/mcp",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			require.NoError(t, s.handleProtectedResourceMetadata(c))
			require.Equal(t, http.StatusOK, rec.Code)

			var got protectedResourceMetadata
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Equal(t, tc.wantResource, got.Resource,
				"strict RFC 9728 clients reject metadata whose resource field "+
					"does not match the URL they requested")
			// Authorization servers stays at the origin regardless — the AS
			// is a property of the deployment, not the specific resource path.
			require.Equal(t, []string{"https://bb.example.com"}, got.AuthorizationServers)
		})
	}
}
