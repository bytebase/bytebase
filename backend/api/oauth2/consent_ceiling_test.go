package oauth2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestConsentCeiling pins the ceiling check this flow did not have.
//
// Before it, the authorization flow knew nothing about the MCP ceiling: a user
// in a workspace with MCP turned off approved the consent, was issued a real
// authorization code, exchanged it for a real token, and only learned at /mcp
// that the door refuses them. The credential outlived the misunderstanding,
// and nothing recorded that anyone had tried.
//
// Two halves, and they are separable on purpose. The refusal is the
// enforcement and does not depend on the row; the row is what an operator
// sees and does not change the verdict. Removing the write leaves every
// refusal assertion below green and fails only the row assertions.
func TestConsentCeiling(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES
			('ws-disabled'), ('ws-readonly'), ('ws-unset'), ('ws-typo'), ('ws-reserved');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'demo@example.com', 'unused');
		INSERT INTO oauth2_client (client_id, workspace, client_secret_hash, config)
		VALUES ('client-A', NULL, 'unused-hash', '{"clientName":"test","redirectUris":["http://localhost/cb"],"grantTypes":["authorization_code","refresh_token"],"tokenEndpointAuthMethod":"none"}'::jsonb);
		INSERT INTO setting (name, workspace, value) VALUES
			('MCP', 'ws-disabled', '{"capability":"DISABLED"}'),
			('MCP', 'ws-readonly', '{"capability":"READ_ONLY"}'),
			('MCP', 'ws-typo', '{"capability":"READ_ONLYY"}'),
			('MCP', 'ws-reserved', '{"capability":2}');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	s := newTestService(st, "https://bb.example.com")

	auditRows := func(t *testing.T, workspace string) []*store.AuditLog {
		t.Helper()
		logs, err := st.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: workspace})
		require.NoError(t, err)
		return logs
	}

	t.Run("a disabled workspace refuses the consent and records the attempt", func(t *testing.T) {
		rec := consentTo(t, s, "ws-disabled")

		require.Equal(t, http.StatusForbidden, rec.Code)
		body := rec.Body.String()
		require.Contains(t, body, "MCP access is turned off")
		require.Contains(t, body, "turned MCP access off for this workspace")
		require.Contains(t, body, "in the workspace settings", "the page names what to ask an admin for")
		require.NotContains(t, body, "code=", "no authorization code may be issued")

		// Against the table, not only the page: a refusal that rendered but
		// still wrote a code would leave a usable credential behind.
		var codes int
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT count(*) FROM oauth2_authorization_code`).Scan(&codes))
		require.Zero(t, codes, "a refused consent must persist no authorization code")

		rows := auditRows(t, "ws-disabled")
		require.Len(t, rows, 1)
		row := rows[0].Payload
		require.Equal(t, "/bytebase.mcp.Consent/Approve", row.GetMethod())
		require.Equal(t, "workspaces/ws-disabled", row.GetParent())
		require.Equal(t, "users/demo@example.com", row.GetUser())
		require.EqualValues(t, 7, row.GetStatus().GetCode(), "PermissionDenied")
		require.Contains(t, row.GetStatus().GetMessage(), "turned MCP access off")

		// The MCP marker, so `mcp == "true"` finds this alongside the
		// connection denial the same user would have hit next.
		require.NotNil(t, row.GetMcpDelegation())
		require.Equal(t, testClientID, row.GetMcpDelegation().GetClientId())
		require.Equal(t, "mcp:read-only", row.GetMcpDelegation().GetScope())
		require.Equal(t, testResource, row.GetMcpDelegation().GetResource())
		require.Empty(t, row.GetMcpDelegation().GetCorrelationId(),
			"a consent never reached the /mcp boundary that mints one")
	})

	t.Run("a stored ceiling nobody can read is refused as a typo, not as policy", func(t *testing.T) {
		rec := consentTo(t, s, "ws-typo")

		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "not one this build understands")

		rows := auditRows(t, "ws-typo")
		require.Len(t, rows, 1)
		require.Contains(t, rows[0].Payload.GetStatus().GetMessage(), "not one this build understands")
	})

	t.Run("a ceiling this build does not serve is refused too", func(t *testing.T) {
		rec := consentTo(t, s, "ws-reserved")

		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "not one this build serves")
		require.Len(t, auditRows(t, "ws-reserved"), 1)
	})

	t.Run("a serving ceiling consents and records nothing", func(t *testing.T) {
		for _, workspace := range []string{"ws-readonly", "ws-unset"} {
			t.Run(workspace, func(t *testing.T) {
				rec := consentTo(t, s, workspace)

				require.Equal(t, http.StatusOK, rec.Code)
				redirect := redirectTargetOf(t, rec.Body.String())
				require.Empty(t, redirect.Query().Get("error"))
				require.NotEmpty(t, redirect.Query().Get("code"))

				require.Empty(t, auditRows(t, workspace),
					"this gate records refusals; a granted consent is the token endpoint's story")
			})
		}
	})
}

// TestConsentCeilingRefusalIsNotAWideningPath pins that the refusal cannot be
// walked around by asking for a different scope: the ceiling decides whether
// any grant is issued at all, before the requested mode is considered.
func TestConsentCeilingRefusalIsNotAWideningPath(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-disabled');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'demo@example.com', 'unused');
		INSERT INTO oauth2_client (client_id, workspace, client_secret_hash, config)
		VALUES ('client-A', NULL, 'unused-hash', '{"clientName":"test","redirectUris":["http://localhost/cb"],"grantTypes":["authorization_code","refresh_token"],"tokenEndpointAuthMethod":"none"}'::jsonb);
		INSERT INTO setting (name, workspace, value) VALUES ('MCP', 'ws-disabled', '{"capability":"DISABLED"}');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	s := newTestService(st, "https://bb.example.com")

	for _, scope := range []string{"mcp:read-only", "mcp:read-write", ""} {
		rec := consentTo(t, s, "ws-disabled", url.Values{"scope": {scope}})
		require.Equal(t, http.StatusForbidden, rec.Code, "scope %q", scope)
	}
	logs, err := st.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: "ws-disabled"})
	require.NoError(t, err)
	require.Len(t, logs, 3, "every attempt is recorded, not just the first")
}

// consentTo posts an approved consent form as a user whose session is in the
// given workspace, and returns the raw response so a caller can read either the
// refusal page or the meta-refresh carrying the code.
func consentTo(t *testing.T, s *Service, workspaceID string, extra ...url.Values) *httptest.ResponseRecorder {
	t.Helper()

	challenge := sha256.Sum256([]byte(testCodeVerifier))
	form := url.Values{
		"client_id":             {testClientID},
		"redirect_uri":          {testRedirectURI},
		"state":                 {"state-1"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"action":                {"allow"},
		"resource":              {testResource},
		"scope":                 {"mcp:read-only"},
	}
	for _, values := range extra {
		for name, value := range values {
			form[name] = value
		}
	}

	sessionToken, err := auth.GenerateAccessToken(testUserEmail, workspaceID, testSecret, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/oauth2/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	rec := httptest.NewRecorder()

	e := echo.New()
	s.RegisterRoutes(e)
	e.ServeHTTP(rec, req)
	return rec
}

// TestConsentCeilingOutageIsNotARefusal pins the arm a real store cannot be
// asked to take: the ceiling read itself fails.
//
// It is the one exit that returns a nil error while having written a response,
// which is the exact shape this flow already got wrong once — a caller reading
// the nil as "may proceed" mints an authorization code underneath a 200 the
// client reads as an error. And it must not read as a refusal: telling a user
// their admin disabled MCP during a database blip sends them to an admin with
// nothing to fix.
func TestConsentCeilingOutageIsNotARefusal(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-outage');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'demo@example.com', 'unused');
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

	s := newTestService(st, "https://bb.example.com")
	s.mcpCeiling = failingCeilingReader{err: errors.New("connection refused")}

	rec := consentTo(t, s, "ws-outage")

	// The client is told to retry, in its own vocabulary, and gets no code.
	require.Equal(t, http.StatusOK, rec.Code, "an error redirect is a 200 carrying the error")
	redirect := redirectTargetOf(t, rec.Body.String())
	require.Equal(t, "temporarily_unavailable", redirect.Query().Get("error"))
	require.Empty(t, redirect.Query().Get("code"), "an outage must not mint a credential either")

	var codes int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM oauth2_authorization_code`).Scan(&codes))
	require.Zero(t, codes)

	logs, err := st.SearchAuditLogs(ctx, &store.AuditLogFind{Workspace: "ws-outage"})
	require.NoError(t, err)
	require.Empty(t, logs, "an outage recorded as a policy denial is a decision nobody made")
}

// failingCeilingReader is a ceiling read that fails, which is what a real store
// cannot be made to do for one call.
type failingCeilingReader struct{ err error }

func (f failingCeilingReader) GetMCPSettingsUncached(context.Context, string) (store.MCPSettings, error) {
	return store.MCPSettings{}, f.err
}

// TestEveryPolicyVerdictHasAConsentSentence pins that this door has wording for every
// verdict that reaches it. The two doors word the same verdict differently on
// purpose — one answers an agent reading an HTTP error, the other a person
// reading a page — so what must not drift is coverage, not phrasing.
func TestEveryPolicyVerdictHasAConsentSentence(t *testing.T) {
	for _, verdict := range auth.PolicyMCPCeilingVerdicts() {
		require.NotEmpty(t, consentRefusals[verdict],
			"a consent refused for %v would say a policy refused it and nothing about which", verdict)
	}
	require.Len(t, consentRefusals, len(auth.PolicyMCPCeilingVerdicts()),
		"a sentence for a verdict that never reaches here is one nobody maintains")
}

// TestTokenIssuanceRechecksTheCeiling pins the half a consent-time check alone
// would miss.
//
// The consent decision is captured once and reused at every later issuance: an
// authorization code outlives the consent by ten minutes and a refresh token by
// thirty days, so a grant consented while MCP was on keeps minting access
// tokens long after an admin turns MCP off. The workspace membership check on
// this endpoint exists for the same reason, and says so in its own comment.
func TestTokenIssuanceRechecksTheCeiling(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-test');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'demo@example.com', 'unused');
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
	s := newTestService(st, "https://bb.example.com")

	// Consent while MCP is on, so the grant is genuine and only the ceiling
	// moves underneath it.
	code := consentOK(t, s, url.Values{"resource": {testResource}, "scope": {"mcp:read-only"}})
	tokens := tokenOK(t, s, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {testRedirectURI},
		"code_verifier": {testCodeVerifier},
		"client_id":     {testClientID},
	})
	require.NotEmpty(t, tokens.RefreshToken)

	_, err = db.ExecContext(ctx,
		`INSERT INTO setting (name, workspace, value) VALUES ('MCP', 'ws-test', '{"capability":"DISABLED"}')`)
	require.NoError(t, err)

	refresh := func() *httptest.ResponseRecorder {
		return postToken(t, s, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {tokens.RefreshToken},
			"client_id":     {testClientID},
		})
	}

	t.Run("a refresh stops re-issuing", func(t *testing.T) {
		rec := refresh()
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		require.Equal(t, "invalid_grant", errorCode(t, rec))
		require.Contains(t, errorDescription(t, rec), "turned MCP access off")
	})

	// The refusal must not burn the credential it refuses. The ceiling is a
	// toggle, and its own message tells the user raising it restores service —
	// so an hour of MCP being off must not cost every client its refresh token
	// and force a fresh interactive consent. The consume is single-use and
	// irreversible, so this only holds while the check runs before it.
	t.Run("the refused refresh survives the policy being reversed", func(t *testing.T) {
		var left int
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT count(*) FROM oauth2_refresh_token`).Scan(&left))
		require.Equal(t, 1, left, "a refused refresh must not consume the token")

		_, err := db.ExecContext(ctx,
			`UPDATE setting SET value = '{"capability":"READ_WRITE"}' WHERE workspace = 'ws-test' AND name = 'MCP'`)
		require.NoError(t, err)

		rec := refresh()
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// Put it back so the subtest below still meets a refusing ceiling.
		_, err = db.ExecContext(ctx,
			`UPDATE setting SET value = '{"capability":"DISABLED"}' WHERE workspace = 'ws-test' AND name = 'MCP'`)
		require.NoError(t, err)
	})

	t.Run("an unexchanged code stops exchanging, and is not burned either", func(t *testing.T) {
		// A code minted before the ceiling moved. The consent that produced it
		// was allowed; the exchange is not.
		s.mcpCeiling = servingCeilingReader{}
		fresh := consentOK(t, s, url.Values{"resource": {testResource}, "scope": {"mcp:read-only"}})
		s.mcpCeiling = st

		exchange := func() *httptest.ResponseRecorder {
			return postToken(t, s, url.Values{
				"grant_type":    {"authorization_code"},
				"code":          {fresh},
				"redirect_uri":  {testRedirectURI},
				"code_verifier": {testCodeVerifier},
				"client_id":     {testClientID},
			})
		}

		rec := exchange()
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		require.Equal(t, "invalid_grant", errorCode(t, rec))

		var left int
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT count(*) FROM oauth2_authorization_code`).Scan(&left))
		require.Equal(t, 1, left, "a refused exchange must not consume the code")

		_, err := db.ExecContext(ctx,
			`UPDATE setting SET value = '{"capability":"READ_WRITE"}' WHERE workspace = 'ws-test' AND name = 'MCP'`)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, exchange().Code, "the same code exchanges once the ceiling is raised")
	})
}

// servingCeilingReader lets a consent through so the test can mint a grant that
// the live ceiling would now refuse.
type servingCeilingReader struct{}

func (servingCeilingReader) GetMCPSettingsUncached(context.Context, string) (store.MCPSettings, error) {
	return store.MCPSettings{Capability: storepb.MCPSetting_READ_WRITE}, nil
}
