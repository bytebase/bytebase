package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRefusedEndpointsAreNotAdvertised pins the two halves of how the index
// treats an RPC no MCP ceiling serves: invisible to every discovery path an
// agent has, and still resolvable by operation ID so a call that arrives anyway
// meets the gate's actionable denial rather than "unknown operation, use
// search_api".
//
// The predicate is "refused under every ceiling", not "FORBIDDEN". Both refused
// classes are covered because the gate refuses both, and an agent offered work
// that always comes back 403 burns turns on it and writes a policy-denial audit
// row each time — which dilutes the very signal those rows exist to carry.
func TestRefusedEndpointsAreNotAdvertised(t *testing.T) {
	idx, err := NewOpenAPIIndex()
	require.NoError(t, err)

	const forbidden = "bytebase.v1.AuthService.Login"
	const forbiddenPath = "/bytebase.v1.AuthService/Login"
	require.True(t, refusedToMCP(forbiddenPath), "precondition: Login is annotated FORBIDDEN")

	const excluded = "bytebase.v1.UserService.ListUsers"
	const excludedPath = "/bytebase.v1.UserService/ListUsers"
	require.True(t, refusedToMCP(excludedPath), "precondition: ListUsers is annotated EXCLUDED")

	t.Run("still resolvable by operation ID", func(t *testing.T) {
		for operation, path := range map[string]string{forbidden: forbiddenPath, excluded: excludedPath} {
			ep, ok := idx.GetEndpoint(operation)
			require.True(t, ok, "call_api must still resolve %s, so the denial can explain itself", operation)
			require.Equal(t, path, ep.Path)
		}
	})

	t.Run("only callable auth endpoints are listed", func(t *testing.T) {
		require.Contains(t, idx.Services(), "AuthService",
			"AuthService has a served authentication restriction endpoint")
		endpoints := idx.GetServiceEndpoints("AuthService")
		require.Len(t, endpoints, 1)
		require.Equal(t, "bytebase.v1.AuthService.GetAuthenticationRestriction", endpoints[0].OperationID)
	})

	t.Run("a service whose every method is refused is not listed at all", func(t *testing.T) {
		// Every identity-provider method is refused: the writes choose what
		// will later be trusted to mint a credential, and the reads are
		// workspace administration. Nothing about the service is offered.
		require.NotContains(t, idx.Services(), "IdentityProviderService")
		require.Empty(t, idx.GetServiceEndpoints("IdentityProviderService"))
	})

	t.Run("a service keeps only the methods a ceiling serves", func(t *testing.T) {
		// ProjectService is the mixed case. Its four reads are served now that
		// the webhook URL no longer rides out on them; its writes administer
		// the workspace and TestWebhook posts to a third party, so those stay
		// refused and stay unadvertised.
		var served []string
		for _, ep := range idx.GetServiceEndpoints("ProjectService") {
			served = append(served, ep.OperationID)
		}
		require.ElementsMatch(t, []string{
			"bytebase.v1.ProjectService.GetProject",
			"bytebase.v1.ProjectService.ListProjects",
			"bytebase.v1.ProjectService.BatchGetProjects",
			"bytebase.v1.ProjectService.SearchProjects",
		}, served)
		require.Contains(t, idx.Services(), "ProjectService")
	})

	t.Run("absent from search", func(t *testing.T) {
		for _, query := range []string{"login", "password", "workspace", "project", "user", "instance"} {
			for _, hit := range idx.Search(query) {
				require.False(t, refusedToMCP(hit.Path),
					"search %q offered %s, which an MCP session can never call", query, hit.Path)
			}
		}
	})

	t.Run("served endpoints are unaffected", func(t *testing.T) {
		_, ok := idx.GetEndpoint("bytebase.v1.DatabaseService.ListDatabases")
		require.True(t, ok)
		require.Contains(t, idx.Services(), "DatabaseService")
		require.NotEmpty(t, idx.Search("database"))
	})
}
