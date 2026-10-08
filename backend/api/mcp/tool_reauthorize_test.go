package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// newReauthorizeTestServer builds an MCP server over the in-memory store, which
// records the refresh grants the reauthorize tool asks it to revoke. That the
// real store deletes them and the token endpoint stops refreshing is pinned
// in backend/tests TestOAuth2GrantLifecycle.
func newReauthorizeTestServer(t *testing.T) (*Server, *testServerStore) {
	t.Helper()
	st := newTestServerStore()
	s, err := newServerWithStore(st, &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}, revalidationSecret, nil)
	require.NoError(t, err)
	return s, st
}

// probeAuthMiddleware reports the status the /mcp boundary gives a bearer.
func probeAuthMiddleware(t *testing.T, s *Server, bearer string) int {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := s.authMiddleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "success")
	})
	if err := handler(c); err != nil {
		echo.DefaultHTTPErrorHandler(true)(c, err)
	}
	return rec.Code
}

func TestReauthorizeRejectsCurrentAccessToken(t *testing.T) {
	ctx := context.Background()
	s, st := newReauthorizeTestServer(t)

	accessToken := generateOAuth2MCPToken(t, revalidationSecret, "client-A", "ws-test")
	reauthorizeCtx := withAccessToken(ctx, accessToken)
	reauthorizeCtx = withUserEmail(reauthorizeCtx, "test@example.com")
	reauthorizeCtx = withOAuth2ClientID(reauthorizeCtx, "client-A")
	reauthorizeCtx = withWorkspaceID(reauthorizeCtx, "ws-test")
	_, _, err := s.handleReauthorize(reauthorizeCtx, nil, ReauthorizeInput{})
	require.NoError(t, err)

	require.Equal(t, [][2]string{{"test@example.com", "client-A"}}, st.deletedRefreshGrants,
		"the caller's refresh grants for this client are revoked, and only those")
	require.Equal(t, http.StatusUnauthorized, probeAuthMiddleware(t, s, accessToken))
}

// TestReauthorizeRejectsRefreshedAccessToken pins that reauthorize revokes the
// bearer the caller is actually holding.
//
// Tool handlers run on the initialize request's context, so a token read from
// that context is the one the session was opened with. A client that refreshed
// mid-session (same identity, so the session survives by design) would have
// reauthorize revoke the token it already discarded, while the token it is
// using keeps passing the boundary until it expires — deleting the refresh
// grants but never producing the OAuth challenge the tool promises.
func TestReauthorizeRejectsRefreshedAccessToken(t *testing.T) {
	ctx := context.Background()
	s, _ := newReauthorizeTestServer(t)

	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	defer ts.Close()

	// Same identity, so the refreshed token keeps the session alive.
	opened := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)
	refreshed := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", 2*time.Hour)
	require.NotEqual(t, opened, refreshed)

	transport := &swappingTransport{}
	transport.token.Store(&opened)
	client := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   ts.URL + "/mcp",
		HTTPClient: &http.Client{Transport: transport},
		MaxRetries: -1,
	}, nil)
	require.NoError(t, err)
	defer session.Close()

	transport.token.Store(&refreshed)
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := session.CallTool(callCtx, &mcp.CallToolParams{Name: "reauthorize"})
	require.NoError(t, err)
	require.False(t, result.IsError, "reauthorize must succeed: %v", result.Content)

	require.Equal(t, http.StatusUnauthorized, probeAuthMiddleware(t, s, refreshed),
		"the bearer the caller reauthorized with must stop working")
}
