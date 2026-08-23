package mcp

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestMCPConnectionDenialEmission pins which refusals at /mcp become audit rows
// and which do not.
//
// The line is what an operator can act on. A ceiling that refuses the workspace
// is a decision somebody made and somebody else has to change, and it is
// invisible everywhere else: the refusal happens in echo middleware, so neither
// connect chain audits it, and the session it refused never existed to leave
// any other trace. A rejected credential is not that. Tokens expire and get
// replaced constantly, by design, and a row per expiry would bury the denials
// this exists to surface under ordinary churn — while telling an operator
// nothing they can do anything about.
//
// The refusal comes first either way. The row is best effort; the 403 is not.
func TestMCPConnectionDenialEmission(t *testing.T) {
	const secret = "test-secret-key"

	newServer := func(t *testing.T, st serverStore) *Server {
		t.Helper()
		profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
		srv, err := newServerWithStore(st, profile, secret, nil)
		require.NoError(t, err)
		return srv
	}

	connect := func(t *testing.T, srv *Server, token string) int {
		t.Helper()
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", "TestAgent/1.0")
		req.Header.Set(common.HeaderRealIP, "10.0.1.50")
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

	t.Run("a ceiling that refuses the workspace writes one row", func(t *testing.T) {
		st := newTestServerStore()
		st.capability = storepb.MCPSetting_DISABLED
		srv := newServer(t, st)

		require.Equal(t, http.StatusForbidden, connect(t, srv, mcpToken(t, secret, tokenOptions{})))

		require.Len(t, st.auditRows, 1)
		row := st.auditRows[0]
		require.Equal(t, common.AuditMethodMCPSessionAuthorize, row.Method)
		require.Equal(t, "workspaces/ws-test", row.Parent)
		require.Equal(t, "workspaces/ws-test", row.Resource)
		require.Equal(t, "users/test@example.com", row.User)
		require.EqualValues(t, 7, row.Status.GetCode(), "PermissionDenied")
		require.Contains(t, row.Status.GetMessage(), "turned MCP access off")
		require.Equal(t, "10.0.1.50", row.RequestMetadata.GetCallerIp())
		require.Equal(t, "TestAgent/1.0", row.RequestMetadata.GetCallerSuppliedUserAgent())

		// The marker, so the row answers `mcp == true` and wears the MCP badge
		// alongside the calls a session that got in would have made.
		require.NotNil(t, row.McpDelegation, "a denial at the MCP door is an MCP row")
		// But no correlation ID. It is session-scoped and this refusal is
		// decided before the SDK resolves a session, so any value here would be
		// a session ID that names one row — and a mid-session refusal would get
		// one different from the rows that session already wrote, hiding the
		// denial from the pivot an operator would use to find it.
		require.Empty(t, row.McpDelegation.GetCorrelationId())
	})

	t.Run("an unreadable ceiling writes a row that names the typo, not the policy", func(t *testing.T) {
		st := newTestServerStore()
		st.capabilityErr = errors.Wrapf(store.ErrMCPCapabilityUnreadable, "READ_ONLYY is not a value this build understands")
		srv := newServer(t, st)

		require.Equal(t, http.StatusForbidden, connect(t, srv, mcpToken(t, secret, tokenOptions{})))

		require.Len(t, st.auditRows, 1)
		require.Contains(t, st.auditRows[0].Status.GetMessage(), "not one this build understands",
			"an admin fixes a broken stored value by rewriting it, not by turning MCP back on")
	})

	t.Run("a ceiling this build does not serve writes its own row", func(t *testing.T) {
		st := newTestServerStore()
		// The reserved number 2, or a ceiling a newer release wrote.
		st.capability = storepb.MCPSetting_Capability(2)
		srv := newServer(t, st)

		require.Equal(t, http.StatusForbidden, connect(t, srv, mcpToken(t, secret, tokenOptions{})))

		require.Len(t, st.auditRows, 1)
		require.Contains(t, st.auditRows[0].Status.GetMessage(), "not one this build serves")
	})

	t.Run("a failed ceiling read is an outage, not a denial", func(t *testing.T) {
		st := newTestServerStore()
		st.capabilityErr = errors.New("connection refused")
		srv := newServer(t, st)

		require.Equal(t, http.StatusServiceUnavailable, connect(t, srv, mcpToken(t, secret, tokenOptions{})))
		require.Empty(t, st.auditRows,
			"recording an outage as a policy denial would put a decision in the log that nobody made")
	})

	t.Run("credential-lifecycle refusals write nothing", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			token string
		}{
			{"expired", mcpToken(t, secret, tokenOptions{expired: true})},
			{"signed with the wrong secret", mcpToken(t, "not-the-secret", tokenOptions{})},
			{"no workspace claim", mcpToken(t, secret, tokenOptions{noWorkspace: true})},
			{"audience bound elsewhere", mcpToken(t, secret, tokenOptions{audience: "https://other.example.com/mcp"})},
		} {
			t.Run(tc.name, func(t *testing.T) {
				st := newTestServerStore()
				st.capability = storepb.MCPSetting_DISABLED
				srv := newServer(t, st)

				require.Equal(t, http.StatusUnauthorized, connect(t, srv, tc.token))
				require.Empty(t, st.auditRows, "token churn is not a denial an operator acts on")
			})
		}
	})

	t.Run("an admitted connection writes nothing", func(t *testing.T) {
		st := newTestServerStore()
		srv := newServer(t, st)

		require.Equal(t, http.StatusOK, connect(t, srv, mcpToken(t, secret, tokenOptions{})))
		require.Empty(t, st.auditRows,
			"the interceptor audits what an admitted session then does; this gate only records refusals")
	})

	t.Run("the row carries the whole grant, not just its presence", func(t *testing.T) {
		const resource = "https://bb.example.com/mcp"
		st := newTestServerStore()
		st.capability = storepb.MCPSetting_DISABLED
		srv := newServer(t, st)

		token := mcpToken(t, secret, tokenOptions{
			audience:    resource,
			clientID:    "client-A",
			scope:       "mcp:read-only",
			mcpTokenUse: true,
		})
		require.Equal(t, http.StatusForbidden, connect(t, srv, token))

		require.Len(t, st.auditRows, 1)
		grant := st.auditRows[0].McpDelegation
		require.Equal(t, "client-A", grant.GetClientId())
		require.Equal(t, "mcp:read-only", grant.GetScope())
		require.Equal(t, resource, grant.GetResource())
	})

	t.Run("the row survives a client that hung up on its own refusal", func(t *testing.T) {
		st := newTestServerStore()
		st.capability = storepb.MCPSetting_DISABLED
		srv := newServer(t, st)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+mcpToken(t, secret, tokenOptions{}))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		handler := srv.authMiddleware(func(c *echo.Context) error { return c.String(http.StatusOK, "success") })
		if err := handler(c); err != nil {
			echo.DefaultHTTPErrorHandler(true)(c, err)
		}

		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Len(t, st.auditRows, 1)
		require.NoError(t, st.writeCtxErr,
			"the write must not inherit the cancellation it is recording")
		require.True(t, st.writeCtxHasDeadline,
			"detaching from the request drops its deadline too, and this write is on the "+
				"synchronous path of a refusal already decided — an unbounded insert holds the 403 open")
	})

	t.Run("every refused request writes its own row", func(t *testing.T) {
		st := newTestServerStore()
		st.capability = storepb.MCPSetting_DISABLED
		srv := newServer(t, st)
		token := mcpToken(t, secret, tokenOptions{})

		for range 3 {
			require.Equal(t, http.StatusForbidden, connect(t, srv, token))
		}
		require.Len(t, st.auditRows, 3,
			"/mcp is Streamable HTTP, so this gate is per request; nothing collapses repeats")
	})

	t.Run("the stdout mirror runs even when the row cannot be persisted", func(t *testing.T) {
		st := newTestServerStore()
		st.capability = storepb.MCPSetting_DISABLED
		st.auditErr = errors.New("audit_log insert failed")
		srv := newServer(t, st)
		srv.profile.RuntimeEnableAuditLogStdout.Store(true)

		var buf bytes.Buffer
		prev := slog.Default()
		t.Cleanup(func() { slog.SetDefault(prev) })
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))

		require.Equal(t, http.StatusForbidden, connect(t, srv, mcpToken(t, secret, tokenOptions{})))
		require.Contains(t, buf.String(), `"log_type":"audit"`,
			"a database failure is exactly when the stream is the surface that still works")
		require.Contains(t, buf.String(), common.AuditMethodMCPSessionAuthorize)
	})

	t.Run("a failed write does not admit the connection", func(t *testing.T) {
		st := newTestServerStore()
		st.capability = storepb.MCPSetting_DISABLED
		st.auditErr = errors.New("audit_log insert failed")
		srv := newServer(t, st)

		require.Equal(t, http.StatusForbidden, connect(t, srv, mcpToken(t, secret, tokenOptions{})),
			"an audit row that cannot be written must never turn a refusal into an admission")
	})
}

type tokenOptions struct {
	expired     bool
	noWorkspace bool
	audience    string
	clientID    string
	scope       string
	mcpTokenUse bool
}

// mcpToken mints the bearer a client presents at /mcp, with one thing wrong at
// a time.
func mcpToken(t *testing.T, secret string, opts tokenOptions) string {
	t.Helper()
	audience := any(auth.OAuth2AccessTokenAudience)
	if opts.audience != "" {
		audience = opts.audience
	}
	expiry := time.Now().Add(time.Hour)
	if opts.expired {
		expiry = time.Now().Add(-time.Hour)
	}
	claims := jwt.MapClaims{
		"iss": "bytebase",
		"sub": "test@example.com",
		"aud": audience,
		"exp": expiry.Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	if !opts.noWorkspace {
		claims["workspace_id"] = "ws-test"
	}
	if opts.clientID != "" {
		claims["client_id"] = opts.clientID
	}
	if opts.scope != "" {
		claims["scope"] = opts.scope
	}
	if opts.mcpTokenUse {
		claims["token_use"] = auth.TokenUseMCP
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

// TestCeilingRefusalsAreRecognizedAsPolicy runs the real IsPolicyRefusal over
// the ceiling refusals that can reach a tool, rather than restating its phrase
// list somewhere it would drift.
//
// Only these two reach it. A 403 is what the predicate is consulted for, and
// the per-request gate answers CodePermissionDenied for exactly these; an
// outage answers CodeUnavailable, and DISABLED is refused at this door before
// any tool call and by the serving table at the per-request one. A refusal it
// stops recognizing gets "request a project role" appended — advice that cannot
// lift a workspace setting.
func TestCeilingRefusalsAreRecognizedAsPolicy(t *testing.T) {
	for _, verdict := range []auth.MCPCeilingVerdict{
		auth.MCPCeilingUnreadable,
		auth.MCPCeilingUnserved,
	} {
		require.True(t, IsPolicyRefusal(verdict.Refusal()),
			"%v reaches a tool as a 403 and must carry its own way out", verdict)
	}
}
