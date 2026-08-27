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

// TestMCPConnectionAllowed pins the rule the connection gate decides from:
// READ_WRITE and READ_ONLY admit a connection, and everything else fails
// closed — DISABLED, unknown stored values such as the reserved number 2, and
// the zero value. The gate reaches it through auth.ClassifyMCPCeiling, which
// TestMCPCeilingVerdictAdmissionMatchesTheServesPredicate holds against this
// same predicate over the whole enum.
//
// READ_ONLY is admitted from the cutover, once the SQL clamp made a read-only
// session one that cannot write. What it may then do is decided per method by
// the ceiling gate and per statement by the clamp; this function only decides
// whether the session opens at all.
//
// UNSPECIFIED is refused here because migration and workspace creation always
// persist a concrete capability.
func TestMCPConnectionAllowed(t *testing.T) {
	tests := []struct {
		capability storepb.MCPSetting_Capability
		allowed    bool
	}{
		{storepb.MCPSetting_CAPABILITY_UNSPECIFIED, false},
		{storepb.MCPSetting_READ_WRITE, true},
		{storepb.MCPSetting_DISABLED, false},
		{storepb.MCPSetting_Capability(2), false},  // reserved (was METADATA_ONLY)
		{storepb.MCPSetting_Capability(99), false}, // no such value in any build
		{storepb.MCPSetting_READ_ONLY, true},
	}
	for _, tt := range tests {
		t.Run(tt.capability.String(), func(t *testing.T) {
			require.Equal(t, tt.allowed, auth.MCPCeilingServesAnything(tt.capability))
		})
	}
}

// TestMCPKillSwitchEndToEnd drives the /mcp auth middleware against a real store:
// a workspace with DISABLED is rejected server-side with 403, while an
// explicit READ_WRITE ceiling is allowed. This is the server-side enforcement
// the kill-switch promises — the token authenticates fine; policy, not auth,
// blocks the disabled workspace.
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

	// Both workspaces have the metadata row required after migration.
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_MCP,
		Workspace: "ws-disabled",
		Value:     &storepb.MCPSetting{Capability: storepb.MCPSetting_DISABLED},
	})
	require.NoError(t, err)
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_MCP,
		Workspace: "ws-open",
		Value:     &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_WRITE},
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
		"the explicit READ_WRITE ceiling must be allowed")
}

// TestMCPKillSwitchBypassesSettingCache pins that the /mcp gate reads the
// stored policy fresh instead of through the store's setting cache. The cache
// has no TTL and only in-process writes refresh it, so with a cache-enabled
// store (the non-HA production shape) a cached READ_WRITE value would keep
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

	// The explicit default row has to exist for the out-of-band flip below to
	// have something to write into.
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_MCP,
		Workspace: "ws-cached",
		Value:     &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_WRITE},
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

	// First request: READ_WRITE ceiling -> allowed. This also primes the setting cache.
	require.Equal(t, http.StatusOK, statusFor())

	// Flip the ceiling behind the store's back, as an emergency SQL toggle would.
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{capability}', '"DISABLED"')
		WHERE workspace = 'ws-cached' AND name = 'MCP';
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
// NAME it does not recognize, so `{"capability":"READ_WRTIE"}` would parse as
// unset and resolve to READ_WRITE — a mistyped kill switch that silently did
// nothing. An unknown enum NUMBER never had that problem, because protojson
// keeps it and no mode serves it.
//
// The migration owns the legacy workspace-profile translation. Runtime reads
// only the mandatory MCP row, so a missing row is invalid metadata and fails
// closed rather than silently becoming READ_WRITE.
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
		{"ws-typo", `{"capability":"READ_WRTIE"}`, http.StatusForbidden,
			"a ceiling this build cannot read is not a ceiling that permits"},
		{"ws-reserved", `{"capability":2}`, http.StatusForbidden,
			"the reserved number was METADATA_ONLY; no mode serves it"},
		{"ws-explicit-unset", `{"capability":"CAPABILITY_UNSPECIFIED"}`, http.StatusForbidden,
			"no Bytebase write produces this key with this value, so a row that has it was hand-edited"},
		{"ws-readonly", `{"capability":"READ_ONLY"}`, http.StatusOK,
			"READ_ONLY admits a session now that the SQL clamp holds it to reads"},
		{"ws-other-unknown-field", `{"capability":"READ_WRITE","someFieldFromANewerRelease":"x"}`, http.StatusOK,
			"one unknown field must not disable MCP when the required ceiling is valid"},
		{"ws-open", `{"capability":"READ_WRITE"}`, http.StatusOK,
			"the explicit default admits MCP"},
		{"ws-missing-capability", `{}`, http.StatusForbidden,
			"migration makes the capability explicit, so an unset row is invalid metadata"},
		{"ws-wrong-type", `{"capability":true}`, http.StatusForbidden,
			"a permanently invalid stored capability is a policy failure, not a retryable outage"},
		{"ws-ignoring", `{"capability":"READ_ONLY","ignoreMaskingExemptions":true}`, http.StatusOK,
			"ignoring masking exemptions is not a reason to refuse the connection; it changes what the session reads"},
	}

	for _, row := range rows {
		_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, row.workspace)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx,
			`INSERT INTO setting (name, workspace, value) VALUES ('MCP', $1, $2)`,
			row.workspace, row.value)
		require.NoError(t, err)
	}

	// These states can only be produced after migration by metadata corruption or
	// an incomplete manual import. Runtime must not recreate migration behavior.
	for _, ws := range []struct{ workspace, profile string }{
		{workspace: "ws-absent"},
		{workspace: "ws-legacy", profile: `{"mcpCapability":"DISABLED"}`},
	} {
		_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, ws.workspace)
		require.NoError(t, err)
		if ws.profile != "" {
			_, err = db.ExecContext(ctx,
				`INSERT INTO setting (name, workspace, value) VALUES ('WORKSPACE_PROFILE', $1, $2)`,
				ws.workspace, ws.profile)
			require.NoError(t, err)
		}
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

	// The connection's STATUS alone cannot separate "the ceiling parsed as
	// DISABLED" from "the ceiling could not be read": both are policy and both
	// answer 403. Its wording does, and TestMCPConnectionDenialEmission pins
	// that. The resolver is asserted directly here, on every workspace at once.
	//
	// That doubles as the composite-primary-key check the setting table needs.
	// It is keyed (workspace, name), and GetMCPSettingsUncached is a new
	// reader of it: a future edit that dropped the workspace predicate would
	// return whichever row the scan reached first and apply one tenant's MCP
	// ceiling to every other tenant — a kill switch answering for the wrong
	// workspace. Every workspace below sits in this one database and must still
	// resolve to its own stored value; the ws-readonly row must resolve as
	// itself rather than as a fail-closed error, which is what separates "the
	// ceiling says read-only" from "the ceiling could not be read".
	for workspace, want := range map[string]storepb.MCPSetting_Capability{
		"ws-readonly":            storepb.MCPSetting_READ_ONLY,
		"ws-open":                storepb.MCPSetting_READ_WRITE,
		"ws-other-unknown-field": storepb.MCPSetting_READ_WRITE,
		"ws-ignoring":            storepb.MCPSetting_READ_ONLY,
	} {
		got, err := s.GetMCPSettingsUncached(ctx, workspace)
		require.NoError(t, err, "%s stores a ceiling this build understands", workspace)
		require.Equal(t, want, got.Capability,
			"%s must resolve its OWN stored ceiling, not a neighbour's", workspace)
	}
	// The toggle comes off the same row read as the ceiling, which is what lets
	// one resolution per request hold for both. Asserting it here rather than in
	// a test of its own is how that stays provable: these rows are read once
	// each, so a reader that resolved the two fields separately would have to
	// show it.
	for workspace, want := range map[string]bool{
		"ws-ignoring":            true,
		"ws-readonly":            false,
		"ws-open":                false,
		"ws-other-unknown-field": false,
	} {
		settings, err := s.GetMCPSettingsUncached(ctx, workspace)
		require.NoError(t, err)
		require.Equal(t, want, settings.IgnoreMaskingExemptions,
			"%s must resolve its OWN toggle, and an unset one is false", workspace)
	}

	for _, workspace := range []string{"ws-absent", "ws-legacy"} {
		got, err := s.GetMCPSettingsUncached(ctx, workspace)
		require.Error(t, err, "%s has no mandatory MCP setting row", workspace)
		require.Nil(t, got, "an error must not carry a usable settings result")
	}

	for _, workspace := range []string{"ws-typo", "ws-reserved", "ws-explicit-unset", "ws-missing-capability", "ws-wrong-type"} {
		got, err := s.GetMCPSettingsUncached(ctx, workspace)
		if err == nil {
			// The reserved number parses; it is refused later because no
			// ceiling serves it. What must never happen is resolving to a
			// value some other workspace stored.
			require.NotEqual(t, storepb.MCPSetting_READ_WRITE, got.Capability,
				"%s must not resolve to a neighbour's permissive ceiling", workspace)
		} else {
			require.Nil(t, got, "an error must not carry a usable settings result")
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
