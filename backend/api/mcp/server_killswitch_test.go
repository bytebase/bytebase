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

// TestMCPConnectionAllowed pins the connection-level gate: READ_WRITE and
// READ_ONLY admit a connection, and everything else fails closed — DISABLED,
// unknown stored values such as the reserved number 2, and the zero value.
//
// READ_ONLY is admitted from the cutover, once the SQL clamp made a read-only
// session one that cannot write. What it may then do is decided per method by
// the ceiling gate and per statement by the clamp; this function only decides
// whether the session opens at all.
//
// UNSPECIFIED is refused here even though an unset stored ceiling means
// READ_WRITE: the store resolves that before this point, so a zero value
// arriving here was resolved by nobody.
func TestMCPConnectionAllowed(t *testing.T) {
	tests := []struct {
		capability storepb.WorkspaceProfileSetting_MCPCapability
		allowed    bool
	}{
		{storepb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED, false},
		{storepb.WorkspaceProfileSetting_READ_WRITE, true},
		{storepb.WorkspaceProfileSetting_DISABLED, false},
		{storepb.WorkspaceProfileSetting_MCPCapability(2), false},  // reserved (was METADATA_ONLY)
		{storepb.WorkspaceProfileSetting_MCPCapability(99), false}, // no such value in any build
		{storepb.WorkspaceProfileSetting_READ_ONLY, true},
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

// TestMCPCeilingStoredValueFailsClosed drives the /mcp connection gate against
// the ceiling values a workspace row can actually hold, including the ones no
// Bytebase write produces.
//
// The typo row is the one that mattered: protojson discards a field whose enum
// NAME it does not recognize, so `{"mcpCapability":"READ_WRTIE"}` used to parse
// as unset and resolve to READ_WRITE — a mistyped kill switch that silently did
// nothing. An unknown enum NUMBER never had that problem, because protojson
// keeps it and no mode serves it.
//
// The read-only row is the cutover pin, now on its other side: READ_ONLY opens
// a session, because a read-only session can no longer write — the ceiling gate
// refuses the methods the mode does not cover and the SQL clamp refuses a
// statement that writes. The rows around it are what must NOT have moved with
// it: a ceiling this build cannot read still refuses, and so does a value no
// Bytebase write produces.
func TestMCPCeilingStoredValueFailsClosed(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	rows := []struct {
		workspace string
		value     string
		want      int
		why       string
	}{
		{"ws-typo", `{"mcpCapability":"READ_WRTIE"}`, http.StatusForbidden,
			"a ceiling this build cannot read is not a ceiling that permits"},
		{"ws-reserved", `{"mcpCapability":2}`, http.StatusForbidden,
			"the reserved number was METADATA_ONLY; no mode serves it"},
		{"ws-explicit-unset", `{"mcpCapability":"MCP_CAPABILITY_UNSPECIFIED"}`, http.StatusForbidden,
			"no Bytebase write produces this key with this value, so a row that has it was hand-edited"},
		{"ws-readonly", `{"mcpCapability":"READ_ONLY"}`, http.StatusOK,
			"READ_ONLY admits a session now that the SQL clamp holds it to reads"},
		{"ws-other-unknown-field", `{"someFieldFromANewerRelease":"x"}`, http.StatusOK,
			"one unknown field must not disable MCP: only the ceiling key is read strictly"},
		{"ws-open", `{}`, http.StatusOK,
			"a workspace that never set a ceiling keeps working"},
	}

	for _, row := range rows {
		_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, row.workspace)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx,
			`INSERT INTO setting (name, workspace, value) VALUES ('WORKSPACE_PROFILE', $1, $2)`,
			row.workspace, row.value)
		require.NoError(t, err)
	}

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	secret := "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	srv, err := NewServer(s, profile, secret, nil)
	require.NoError(t, err)

	// The connection verdict alone cannot separate "the ceiling parsed as
	// READ_ONLY" from "the ceiling could not be read", because mcpCapability
	// maps any error to DISABLED and both refuse. So the resolver is asserted
	// directly, on every workspace at once.
	//
	// That doubles as the composite-primary-key check the setting table needs.
	// It is keyed (workspace, name), and GetMCPCapabilityUncached is a new
	// reader of it: a future edit that dropped the workspace predicate would
	// return whichever row the scan reached first and apply one tenant's MCP
	// ceiling to every other tenant — a kill switch answering for the wrong
	// workspace. Six workspaces with six different stored values sit in this
	// one database, so each must still resolve to its own; the ws-readonly row
	// must resolve as itself rather than as a fail-closed error, which is what
	// separates "the ceiling says read-only" from "the ceiling could not be
	// read" now that both no longer answer the same way at the connection.
	for workspace, want := range map[string]storepb.WorkspaceProfileSetting_MCPCapability{
		"ws-readonly":            storepb.WorkspaceProfileSetting_READ_ONLY,
		"ws-open":                storepb.WorkspaceProfileSetting_READ_WRITE,
		"ws-other-unknown-field": storepb.WorkspaceProfileSetting_READ_WRITE,
	} {
		got, err := s.GetMCPCapabilityUncached(ctx, workspace)
		require.NoError(t, err, "%s stores a ceiling this build understands", workspace)
		require.Equal(t, want, got,
			"%s must resolve its OWN stored ceiling, not a neighbour's", workspace)
	}
	for _, workspace := range []string{"ws-typo", "ws-reserved", "ws-explicit-unset"} {
		got, err := s.GetMCPCapabilityUncached(ctx, workspace)
		if err == nil {
			// The reserved number parses; it is refused later because no
			// ceiling serves it. What must never happen is resolving to a
			// value some other workspace stored.
			require.NotEqual(t, storepb.WorkspaceProfileSetting_READ_WRITE, got,
				"%s must not resolve to a neighbour's permissive ceiling", workspace)
		}
	}

	for _, row := range rows {
		t.Run(row.workspace, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokenForWorkspace(t, secret, row.workspace))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			handler := srv.authMiddleware(func(c *echo.Context) error {
				return c.String(http.StatusOK, "success")
			})
			if err := handler(c); err != nil {
				echo.DefaultHTTPErrorHandler(true)(c, err)
			}
			require.Equal(t, row.want, rec.Code, row.why)
		})
	}
}
