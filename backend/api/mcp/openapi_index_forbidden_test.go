package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestForbiddenEndpointsAreNotAdvertised pins the two halves of how the index
// treats a FORBIDDEN RPC: invisible to every discovery path an agent has, and
// still resolvable by operation ID so a call that arrives anyway meets the
// gate's actionable denial rather than "unknown operation, use search_api".
func TestForbiddenEndpointsAreNotAdvertised(t *testing.T) {
	idx, err := NewOpenAPIIndex()
	require.NoError(t, err)

	const forbidden = "bytebase.v1.AuthService.Login"
	const forbiddenPath = "/bytebase.v1.AuthService/Login"
	require.True(t, forbiddenToMCP(forbiddenPath), "precondition: Login is annotated FORBIDDEN")

	t.Run("still resolvable by operation ID", func(t *testing.T) {
		ep, ok := idx.GetEndpoint(forbidden)
		require.True(t, ok, "call_api must still resolve it, so the denial can explain itself")
		require.Equal(t, forbiddenPath, ep.Path)
	})

	t.Run("absent from service listings", func(t *testing.T) {
		require.NotContains(t, idx.Services(), "AuthService",
			"every AuthService RPC is FORBIDDEN, so the service itself has nothing to offer an agent")
		require.Empty(t, idx.GetServiceEndpoints("AuthService"))
	})

	t.Run("absent from search", func(t *testing.T) {
		for _, query := range []string{"login", "password", "workspace"} {
			for _, hit := range idx.Search(query) {
				require.False(t, forbiddenToMCP(hit.Path),
					"search %q offered %s, which an MCP session can never call", query, hit.Path)
			}
		}
	})

	t.Run("reachable endpoints are unaffected", func(t *testing.T) {
		_, ok := idx.GetEndpoint("bytebase.v1.ProjectService.ListProjects")
		require.True(t, ok)
		require.Contains(t, idx.Services(), "ProjectService")
		require.NotEmpty(t, idx.Search("project"))
	})
}
