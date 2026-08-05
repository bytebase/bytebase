package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
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

// TestMCPConnectionAllowed pins the connection-level gate: only an unset ceiling
// and READ_WRITE admit a connection in this phase. DISABLED, the
// not-yet-enforceable READ_ONLY ceiling, and unknown stored values (such as
// the reserved number 2) all fail closed so a ceiling the server cannot apply
// per-tool never silently grants read-write.
func TestMCPConnectionAllowed(t *testing.T) {
	tests := []struct {
		capability storepb.WorkspaceProfileSetting_MCPCapability
		allowed    bool
	}{
		{storepb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED, true},
		{storepb.WorkspaceProfileSetting_READ_WRITE, true},
		{storepb.WorkspaceProfileSetting_DISABLED, false},
		{storepb.WorkspaceProfileSetting_MCPCapability(2), false}, // reserved (was METADATA_ONLY)
		{storepb.WorkspaceProfileSetting_READ_ONLY, false},
	}
	for _, tt := range tests {
		t.Run(tt.capability.String(), func(t *testing.T) {
			require.Equal(t, tt.allowed, mcpConnectionAllowed(tt.capability))
		})
	}
}

// TestMCPKillSwitchEndToEnd drives the /mcp auth middleware against a real store:
// a workspace with DISABLED is rejected server-side with 403, while a
// workspace that never configured a ceiling defaults to allowed (backward
// compatible). This is the server-side enforcement the kill-switch promises —
// the token authenticates fine; policy, not auth, blocks the disabled workspace.
func TestMCPKillSwitchEndToEnd(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-disabled'), ('ws-open');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	// ws-disabled: MCP explicitly disabled. ws-open: no MCP setting → unset → allowed.
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: "ws-disabled",
		Value:     &storepb.WorkspaceProfileSetting{McpCapability: storepb.WorkspaceProfileSetting_DISABLED},
	})
	require.NoError(t, err)

	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	srv, err := NewServer(s, profile, secret, nil)
	require.NoError(t, err)

	statusFor := func(workspaceID string) int {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenForWorkspace(t, secret, workspaceID))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		handler := srv.authMiddleware(func(c *echo.Context) error {
			return c.String(http.StatusOK, "success")
		})
		if err := handler(c); err != nil {
			echo.DefaultHTTPErrorHandler(true)(c, err)
		}
		return rec.Code
	}

	require.Equal(t, http.StatusForbidden, statusFor("ws-disabled"),
		"DISABLED workspace must be rejected server-side despite a valid token")
	require.Equal(t, http.StatusOK, statusFor("ws-open"),
		"workspace with no MCP ceiling must default to allowed (backward compatible)")
}

// TestMCPKillSwitchBypassesSettingCache pins that the /mcp gate reads the
// stored policy fresh instead of through the store's setting cache. The cache
// has no TTL and only in-process writes refresh it, so with a cache-enabled
// store (the non-HA production shape) a profile cached as unset would keep
// admitting MCP forever after the ceiling is flipped by an out-of-band admin
// path (direct SQL, another process) — the kill switch must observe the stored
// truth on the next request.
func TestMCPKillSwitchBypassesSettingCache(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-cached');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, true /* enableCache */)
	require.NoError(t, err)

	// A profile row with no MCP ceiling, as any configured workspace has.
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: "ws-cached",
		Value:     &storepb.WorkspaceProfileSetting{},
	})
	require.NoError(t, err)

	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	srv, err := NewServer(s, profile, secret, nil)
	require.NoError(t, err)

	statusFor := func() int {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenForWorkspace(t, secret, "ws-cached"))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		handler := srv.authMiddleware(func(c *echo.Context) error {
			return c.String(http.StatusOK, "success")
		})
		if err := handler(c); err != nil {
			echo.DefaultHTTPErrorHandler(true)(c, err)
		}
		return rec.Code
	}

	// First request: unset ceiling → allowed. This also primes the setting cache.
	require.Equal(t, http.StatusOK, statusFor())

	// Flip the ceiling behind the store's back, as an emergency SQL toggle would.
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{mcpCapability}', '"DISABLED"')
		WHERE workspace = 'ws-cached' AND name = 'WORKSPACE_PROFILE';
	`)
	require.NoError(t, err)

	require.Equal(t, http.StatusForbidden, statusFor(),
		"a ceiling written out-of-band must be enforced on the next request, not served stale from the setting cache")
}

func tokenForWorkspace(t *testing.T, secret, workspaceID string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":          "bytebase",
		"sub":          "test@example.com",
		"aud":          auth.OAuth2AccessTokenAudience,
		"workspace_id": workspaceID,
		"exp":          time.Now().Add(time.Hour).Unix(),
		"iat":          time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}
