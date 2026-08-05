package tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// mintMCPOAuthToken runs the full OAuth2 flow a real MCP client performs
// against the live server — RFC 7591 dynamic client registration, PKCE
// consent, code exchange — and returns the resource-bound access token.
func mintMCPOAuthToken(t *testing.T, ctl *controller) string {
	t.Helper()
	httpClient := &http.Client{}
	redirectURI := "http://localhost/cb"

	resp, err := httpClient.Post(ctl.rootURL+"/api/oauth2/register", "application/json",
		strings.NewReader(`{"client_name":"bb-e2e","redirect_uris":["http://localhost/cb"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	var reg struct {
		ClientID string `json:"client_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reg))
	require.NotEmpty(t, reg.ClientID)

	// 43-128 chars per RFC 7636.
	verifier := "e2eVerifier_e2eVerifier_e2eVerifier_e2eVerifier"
	challenge := sha256.Sum256([]byte(verifier))
	form := url.Values{
		"client_id":             {reg.ClientID},
		"redirect_uri":          {redirectURI},
		"state":                 {"e2e-state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"action":                {"allow"},
		"resource":              {ctl.rootURL + "/mcp"},
		"scope":                 {"mcp:read-only"},
	}
	req, err := http.NewRequest(http.MethodPost, ctl.rootURL+"/api/oauth2/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+ctl.authInterceptor.token)
	consentResp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer consentResp.Body.Close()
	consentBody, err := io.ReadAll(consentResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, consentResp.StatusCode, string(consentBody))

	// The authorize handler renders a meta-refresh page instead of a 302 to
	// sidestep CSP form-action restrictions; the callback URL carries the code.
	matches := regexp.MustCompile(`content="0;url=([^"]+)"`).FindStringSubmatch(string(consentBody))
	require.Len(t, matches, 2, "expected a meta-refresh redirect, got: %s", string(consentBody))
	callback, err := url.Parse(html.UnescapeString(matches[1]))
	require.NoError(t, err)
	require.Empty(t, callback.Query().Get("error"), callback.Query().Get("error_description"))
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	tokenResp, err := httpClient.PostForm(ctl.rootURL+"/api/oauth2/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
		"client_id":     {reg.ClientID},
	})
	require.NoError(t, err)
	defer tokenResp.Body.Close()
	tokenBody, err := io.ReadAll(tokenResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, tokenResp.StatusCode, string(tokenBody))
	var token struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(tokenBody, &token))
	require.NotEmpty(t, token.AccessToken)
	return token.AccessToken
}

// TestMCPTokenIsRejectedOnGeneralAPI is the P1a PR 5 boundary e2e: a real
// resource-bound MCP OAuth2 token — minted by the live server through the full
// registration/consent/exchange flow — keeps driving its MCP session's tool
// calls, while the very same bearer is refused by the public v1 API. Tool
// traffic is unaffected because since PR 4 it authenticates with the delegated
// credential minted at the /mcp boundary, never with this token; anything
// presenting an MCP token to the general API is therefore a leaked or misused
// token, and refusing it is what ends the token's life as a universal API
// bearer.
func TestMCPTokenIsRejectedOnGeneralAPI(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mcpToken := mintMCPOAuthToken(t, ctl)

	client := mcp.NewClient(&mcp.Implementation{Name: "bb-e2e", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   ctl.rootURL + "/mcp",
		HTTPClient: &http.Client{Transport: &bearerTransport{token: mcpToken}},
	}, nil)
	a.NoError(err)
	defer session.Close()

	callTool := func() {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "call_api",
			Arguments: map[string]any{"operationId": "ProjectService/ListProjects"},
		})
		a.NoError(err)
		a.False(result.IsError, "tool calls must keep working for an MCP-audience token: %v", result.Content)
	}
	callTool()

	// The very same bearer on the public v1 API is refused before any identity
	// resolution — the principal behind it is a workspace admin, so anything
	// but Unauthenticated here would mean the token still works as an API key.
	asMCPToken := v1connect.NewProjectServiceClient(ctl.client, ctl.rootURL,
		connect.WithInterceptors(&authInterceptor{token: mcpToken}))
	_, err = asMCPToken.ListProjects(ctx, connect.NewRequest(&v1pb.ListProjectsRequest{}))
	a.Error(err, "an MCP token must not be a general API bearer")
	a.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))
	a.Contains(err.Error(), "only accepted at /mcp")

	// The refusal is a property of the public surface, not damage to the MCP
	// session: the same token's tool calls still work afterwards.
	callTool()
}
