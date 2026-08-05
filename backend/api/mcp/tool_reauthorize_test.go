package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// newReauthorizeTestServer boots a store-backed MCP server with one OAuth2
// client, one principal, and one live refresh grant — the state the reauthorize
// tool operates on.
func newReauthorizeTestServer(ctx context.Context, t *testing.T) (*Server, *store.Store, string) {
	t.Helper()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-test');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'test@example.com', 'unused');
		INSERT INTO oauth2_client (client_id, workspace, client_secret_hash, config)
		VALUES ('client-A', NULL, 'unused-hash', '{"clientName":"test","redirectUris":["http://localhost/cb"],"grantTypes":["authorization_code","refresh_token"],"tokenEndpointAuthMethod":"none"}'::jsonb);
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	refreshToken := "refresh-token"
	_, err = st.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
		TokenHash: auth.HashToken(refreshToken),
		ClientID:  "client-A",
		UserEmail: "test@example.com",
		Workspace: "ws-test",
		Config:    &storepb.OAuth2RefreshTokenConfig{},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	s, err := NewServer(st, &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}, revalidationSecret, nil)
	require.NoError(t, err)
	return s, st, refreshToken
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
	s, st, refreshToken := newReauthorizeTestServer(ctx, t)
	const secret = revalidationSecret

	accessToken := generateOAuth2MCPToken(t, secret, "client-A", "ws-test")
	reauthorizeCtx := withAccessToken(ctx, accessToken)
	reauthorizeCtx = withUserEmail(reauthorizeCtx, "test@example.com")
	reauthorizeCtx = withOAuth2ClientID(reauthorizeCtx, "client-A")
	reauthorizeCtx = withWorkspaceID(reauthorizeCtx, "ws-test")
	_, _, err := s.handleReauthorize(reauthorizeCtx, nil, ReauthorizeInput{})
	require.NoError(t, err)

	stored, err := st.GetOAuth2RefreshToken(ctx, "client-A", auth.HashToken(refreshToken))
	require.NoError(t, err)
	require.Nil(t, stored)

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
	s, _, _ := newReauthorizeTestServer(ctx, t)

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
