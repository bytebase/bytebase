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
// and MCP_READ_WRITE admit a connection in this phase. MCP_DISABLED and the
// not-yet-enforceable MCP_METADATA_ONLY / MCP_READ_ONLY ceilings all fail closed
// so a ceiling the server cannot apply per-tool never silently grants read-write.
func TestMCPConnectionAllowed(t *testing.T) {
	tests := []struct {
		capability storepb.WorkspaceProfileSetting_MCPCapability
		allowed    bool
	}{
		{storepb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED, true},
		{storepb.WorkspaceProfileSetting_MCP_READ_WRITE, true},
		{storepb.WorkspaceProfileSetting_MCP_DISABLED, false},
		{storepb.WorkspaceProfileSetting_MCP_METADATA_ONLY, false},
		{storepb.WorkspaceProfileSetting_MCP_READ_ONLY, false},
	}
	for _, tt := range tests {
		t.Run(tt.capability.String(), func(t *testing.T) {
			require.Equal(t, tt.allowed, mcpConnectionAllowed(tt.capability))
		})
	}
}

// TestMCPKillSwitchEndToEnd drives the /mcp auth middleware against a real store:
// a workspace with MCP_DISABLED is rejected server-side with 403, while a
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
		Value:     &storepb.WorkspaceProfileSetting{McpCapability: storepb.WorkspaceProfileSetting_MCP_DISABLED},
	})
	require.NoError(t, err)

	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	srv, err := NewServer(s, profile, secret)
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
		"MCP_DISABLED workspace must be rejected server-side despite a valid token")
	require.Equal(t, http.StatusOK, statusFor("ws-open"),
		"workspace with no MCP ceiling must default to allowed (backward compatible)")
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
