package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// bearerTransport injects the controller's session token into MCP client
// requests, the way a user pastes a token into an MCP client.
type bearerTransport struct {
	token string
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

// TestMCPToolCallParity is the P1a PR 4 behavior-parity pin. A tool call
// through /mcp must behave exactly like the same principal's direct v1 call:
// the delegated credential minted at the boundary establishes the same
// identity on the internal in-memory chain, and RBAC resolves live
// downstream. The session here is a plain bb.user.access web token — the
// legacy admission — so this also pins that legacy sessions keep working
// through tools with empty grant state.
func TestMCPToolCallParity(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Parity baseline: the principal's direct v1 API view.
	direct, err := ctl.projectServiceClient.ListProjects(ctx, connect.NewRequest(&v1pb.ListProjectsRequest{}))
	a.NoError(err)
	var directNames []string
	for _, p := range direct.Msg.Projects {
		directNames = append(directNames, p.Name)
	}
	a.NotEmpty(directNames)

	// The same principal through MCP: initialize a real session against /mcp
	// and run the call_api tool. Every internal hop rides the in-memory
	// transport with the minted credential — if the internal chain rejected it
	// or established a different principal, this call would fail or diverge.
	client := mcp.NewClient(&mcp.Implementation{Name: "bb-e2e", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   ctl.rootURL + "/mcp",
		HTTPClient: &http.Client{Transport: &bearerTransport{token: ctl.authInterceptor.token}},
	}, nil)
	a.NoError(err)
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "call_api",
		Arguments: map[string]any{"operationId": "ProjectService/ListProjects"},
	})
	a.NoError(err)
	a.False(result.IsError, "call_api must succeed through the internal transport: %v", result.Content)

	raw, err := json.Marshal(result.StructuredContent)
	a.NoError(err)
	var out struct {
		Status   int `json:"status"`
		Response struct {
			Projects []struct {
				Name string `json:"name"`
			} `json:"projects"`
		} `json:"response"`
	}
	a.NoError(json.Unmarshal(raw, &out))
	a.Equal(http.StatusOK, out.Status)

	var mcpNames []string
	for _, p := range out.Response.Projects {
		mcpNames = append(mcpNames, p.Name)
	}
	a.ElementsMatch(directNames, mcpNames,
		"the MCP tool call must see exactly the projects the same principal sees directly")
}
