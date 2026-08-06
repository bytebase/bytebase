package oauth2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/api/mcp"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

const (
	testSecret      = "test-secret"
	testUserEmail   = "demo@example.com"
	testWorkspace   = "ws-test"
	testClientID    = "client-A"
	testRedirectURI = "http://localhost/cb"
	// 43-128 chars per RFC 7636.
	testCodeVerifier = "kVerifier_kVerifier_kVerifier_kVerifier_kVer"
	testResource     = "https://bb.example.com/mcp"
)

// TestResourceScopeGrantLifecycle drives the whole consent → token → refresh
// path against a real store. The pure rule space lives in resource_test.go;
// what this pins is the wiring, which is where a resource/scope binding
// silently disappears: the code row must carry what was consented, the token
// endpoint must compare against it, and every refresh must re-issue the same
// grant rather than whatever the client asks for.
func TestResourceScopeGrantLifecycle(t *testing.T) {
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

	configured := newTestService(st, "https://bb.example.com")

	t.Run("authorization starts without an existing browser session", func(t *testing.T) {
		entryService := newTestService(st, "")
		entryService.profile.SaaS = true
		challenge := sha256.Sum256([]byte(testCodeVerifier))
		query := url.Values{
			"response_type":         {"code"},
			"client_id":             {testClientID},
			"redirect_uri":          {testRedirectURI},
			"state":                 {"test-state"},
			"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
			"code_challenge_method": {"S256"},
			"resource":              {testResource},
		}
		req := httptest.NewRequest(http.MethodGet, "/api/oauth2/authorize?"+query.Encode(), nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, rec)
		require.NoError(t, entryService.handleAuthorizeGet(c))
		response, err := echo.UnwrapResponse(c.Response())
		require.NoError(t, err)
		require.Equal(t, http.StatusFound, response.Status)

		location, err := url.Parse(response.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, "/oauth2/consent", location.Path)
		require.Equal(t, testClientID, location.Query().Get("client_id"))
		require.Equal(t, testResource, location.Query().Get("resource"))
	})

	t.Run("configured external URL: matching resource and known scope are consented", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{"resource": {testResource}, "scope": {"mcp:read-only"}})

		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, testResource, got.Config.Resource)
		require.Equal(t, "mcp:read-only", got.Config.Scope)
	})

	t.Run("resource is stored canonicalized, not as sent", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{"resource": {"HTTPS://BB.Example.com/mcp/"}})

		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Equal(t, testResource, got.Config.Resource,
			"the token endpoint compares against this value, so it must already be canonical")
	})

	t.Run("a bare origin is consented as the canonical MCP resource", func(t *testing.T) {
		// Accepting the bare origin (our unsuffixed RFC 9728 document publishes
		// it) must not put a second spelling in the grant: PR 3 binds the token
		// audience to the stored value, and two spellings would mean two
		// audiences to honor at /mcp for as long as any grant lives.
		code := consentOK(t, configured, url.Values{"resource": {"https://bb.example.com"}})

		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Equal(t, testResource, got.Config.Resource)
	})

	t.Run("no resource requested: the grant is bound to this server's MCP resource", func(t *testing.T) {
		// Since PR 3 every new grant is resource-bound, because its tokens carry
		// the resource as their audience: an unbound grant would mint tokens no
		// audience check accepts once the legacy window closes, sending the
		// client into a re-auth loop. A client that never sends `resource`
		// (mandatory for MCP clients per RFC 8707, but not every DCR client is
		// one) is defaulted to the only resource this server would accept anyway.
		code := consentOK(t, configured, url.Values{})

		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Equal(t, testResource, got.Config.Resource)
		require.Empty(t, got.Config.Scope, "only the resource is defaulted; scope stays exactly as requested")
	})

	rejections := []struct {
		name     string
		params   url.Values
		wantCode string
	}{
		{"resource on another host", url.Values{"resource": {"https://evil.example.com/mcp"}}, "invalid_target"},
		{"resource parameter repeated", url.Values{"resource": {testResource, testResource}}, "invalid_target"},
		{"unknown scope", url.Values{"scope": {"mcp:admin"}}, "invalid_scope"},
		{"scope parameter repeated", url.Values{"scope": {"mcp:read-only", "mcp:read-write"}}, "invalid_scope"},
	}
	for _, tc := range rejections {
		t.Run("consent rejected: "+tc.name, func(t *testing.T) {
			redirect := consent(t, configured, tc.params)
			require.Equal(t, tc.wantCode, redirect.Query().Get("error"))
		})
	}

	t.Run("default port resource is consented as the canonical resource", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{"resource": {"https://bb.example.com:443/mcp"}})
		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Equal(t, testResource, got.Config.Resource)
	})

	t.Run("no configured external URL: a resource request gets the actionable setup error", func(t *testing.T) {
		_, err := st.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_WORKSPACE_PROFILE,
			Workspace: testWorkspace,
			Value:     &storepb.WorkspaceProfileSetting{},
		})
		require.NoError(t, err)

		unconfigured := newTestService(st, "")
		redirect := consent(t, unconfigured, url.Values{"resource": {testResource}})
		require.Equal(t, "server_error", redirect.Query().Get("error"))
		description := redirect.Query().Get("error_description")
		require.Contains(t, description, "external URL isn't setup yet")
	})

	t.Run("no configured external URL: consent without a resource gets the setup error too", func(t *testing.T) {
		// PR 2 let a resource-less request through on an unconfigured instance;
		// PR 3 binds every grant, and the binding needs the trusted URL. Failing
		// consent with the actionable setup pointer beats minting tokens that
		// can only ever 401 at /mcp.
		unconfigured := newTestService(st, "")
		redirect := consent(t, unconfigured, url.Values{})
		require.Equal(t, "server_error", redirect.Query().Get("error"))
		require.Contains(t, redirect.Query().Get("error_description"), "external URL isn't setup yet")
	})

	t.Run("the workspace setting is trusted too, not only the flag", func(t *testing.T) {
		// Self-hosted instances configure the external URL in Settings rather
		// than on the command line, so the DB tier has to validate resources
		// exactly like the flag does.
		_, err := st.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_WORKSPACE_PROFILE,
			Workspace: testWorkspace,
			Value:     &storepb.WorkspaceProfileSetting{ExternalUrl: "https://bb.example.com"},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := st.UpsertSetting(ctx, &store.SettingMessage{
				Name:      storepb.SettingName_WORKSPACE_PROFILE,
				Workspace: testWorkspace,
				Value:     &storepb.WorkspaceProfileSetting{},
			})
			require.NoError(t, err)
		})

		fromSetting := newTestService(st, "")
		code := consentOK(t, fromSetting, url.Values{"resource": {testResource}})
		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Equal(t, testResource, got.Config.Resource)

		redirect := consent(t, fromSetting, url.Values{"resource": {"https://evil.example.com/mcp"}})
		require.Equal(t, "invalid_target", redirect.Query().Get("error"))
	})

	t.Run("token exchange and refresh carry the grant forward unchanged", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{"resource": {testResource}, "scope": {"mcp:read-write"}})

		// Exchange, echoing the resource back the way RFC 8707 clients do.
		first := tokenOK(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
			"resource":      {testResource},
		})
		require.Equal(t, "mcp:read-write", first.Scope, "the consented scope must be echoed (RFC 6749 §5.1)")
		require.NotEmpty(t, first.RefreshToken)

		stored, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(first.RefreshToken))
		require.NoError(t, err)
		require.NotNil(t, stored)
		require.Equal(t, testResource, stored.Config.GetResource())
		require.Equal(t, "mcp:read-write", stored.Config.GetScope())

		// A refresh that names a wider scope is refused, and refusal happens
		// before consumption so the client can retry correctly.
		widened := postToken(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {first.RefreshToken},
			"client_id":     {testClientID},
			"scope":         {"mcp:read-only"},
		})
		require.Equal(t, http.StatusBadRequest, widened.Code)
		require.Equal(t, "invalid_scope", errorCode(t, widened))

		mismatched := postToken(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {first.RefreshToken},
			"client_id":     {testClientID},
			"resource":      {"https://evil.example.com/mcp"},
		})
		require.Equal(t, http.StatusBadRequest, mismatched.Code)
		require.Equal(t, "invalid_target", errorCode(t, mismatched))

		// The rejected attempts left the grant usable, and the honest refresh
		// re-issues the same resource and scope.
		second := tokenOK(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {first.RefreshToken},
			"client_id":     {testClientID},
		})
		require.Equal(t, "mcp:read-write", second.Scope)

		rotated, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(second.RefreshToken))
		require.NoError(t, err)
		require.NotNil(t, rotated)
		require.Equal(t, testResource, rotated.Config.GetResource())
		require.Equal(t, "mcp:read-write", rotated.Config.GetScope())
	})

	t.Run("a bare-origin grant refreshes as the canonical resource, named either way", func(t *testing.T) {
		// Consent with the bare origin, then drive the whole exchange naming the
		// bare origin at every step. The stored value stays the canonical MCP
		// URI throughout, and the same normalization applies at the token end so
		// the client is never told its own spelling is wrong.
		code := consentOK(t, configured, url.Values{"resource": {"https://bb.example.com/"}})

		first := tokenOK(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
			"resource":      {"https://bb.example.com"},
		})
		stored, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(first.RefreshToken))
		require.NoError(t, err)
		require.NotNil(t, stored)
		require.Equal(t, testResource, stored.Config.GetResource())

		second := tokenOK(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {first.RefreshToken},
			"client_id":     {testClientID},
			"resource":      {"https://bb.example.com"},
		})
		rotated, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(second.RefreshToken))
		require.NoError(t, err)
		require.NotNil(t, rotated)
		require.Equal(t, testResource, rotated.Config.GetResource(),
			"the canonical form must survive rotation, not drift back to what the client sent")
	})

	t.Run("legacy unbound grants are refused at the token endpoint with re-auth guidance", func(t *testing.T) {
		// Rows created before 3.22.1 carry no resource in Config. PR 2 kept them
		// redeemable as a deliberate interim; PR 3 retires them: tokens are now
		// audience-bound to the grant's stored resource, so an unbound grant has
		// nothing valid to mint. invalid_grant is what makes RFC-compliant
		// clients discard the grant and rerun the OAuth flow, and the
		// description points at the reauthorize tool as the in-band recovery.
		challenge := sha256.Sum256([]byte(testCodeVerifier))
		const legacyCode = "bb_code_legacy_unbound"
		_, err := st.CreateOAuth2AuthorizationCode(ctx, &store.OAuth2AuthorizationCodeMessage{
			Code:      legacyCode,
			ClientID:  testClientID,
			UserEmail: testUserEmail,
			Workspace: testWorkspace,
			Config: &storepb.OAuth2AuthorizationCodeConfig{
				RedirectUri:         testRedirectURI,
				CodeChallenge:       base64.RawURLEncoding.EncodeToString(challenge[:]),
				CodeChallengeMethod: "S256",
			},
			ExpiresAt: time.Now().Add(10 * time.Minute),
		})
		require.NoError(t, err)

		exchanged := postToken(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {legacyCode},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
		})
		require.Equal(t, http.StatusBadRequest, exchanged.Code)
		require.Equal(t, "invalid_grant", errorCode(t, exchanged))
		require.Contains(t, errorDescription(t, exchanged), "re-authorize")
		require.Contains(t, errorDescription(t, exchanged), "reauthorize tool")

		const legacyRefresh = "bb_refresh_legacy_unbound"
		_, err = st.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
			TokenHash: auth.HashToken(legacyRefresh),
			ClientID:  testClientID,
			UserEmail: testUserEmail,
			Workspace: testWorkspace,
			Config:    &storepb.OAuth2RefreshTokenConfig{},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		refreshed := postToken(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {legacyRefresh},
			"client_id":     {testClientID},
		})
		require.Equal(t, http.StatusBadRequest, refreshed.Code)
		require.Equal(t, "invalid_grant", errorCode(t, refreshed))
		require.Contains(t, errorDescription(t, refreshed), "reauthorize tool")

		// Refusal happens before consumption: the row is left for the expiry
		// sweep (or the user's reauthorize) rather than burned by a refresh
		// that issued nothing.
		row, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(legacyRefresh))
		require.NoError(t, err)
		require.NotNil(t, row)

		// A bound grant with no scope is not legacy — only a missing resource
		// marks the retired population.
		boundCode := consentOK(t, configured, url.Values{"resource": {testResource}})
		bound := tokenOK(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {boundCode},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
		})
		tokenOK(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {bound.RefreshToken},
			"client_id":     {testClientID},
		})
	})

	t.Run("rotating the external URL kills outstanding tokens at /mcp until re-consent", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{"resource": {testResource}})
		first := tokenOK(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
		})
		require.NotEqual(t, http.StatusUnauthorized, mcpStatus(t, st, "https://bb.example.com", first.AccessToken),
			"a token bound to the live resource must be admitted at /mcp")

		// Rotate the trusted external URL. The middleware computes its expected
		// audience from live config, so the outstanding token dies at use time
		// with a clean 401 — never silently rebinds.
		require.Equal(t, http.StatusUnauthorized, mcpStatus(t, st, "https://bb-new.example.com", first.AccessToken))

		// The token endpoint, by contrast, mints from the grant's STORED
		// resource, never live config: refresh succeeds but hands back a token
		// still bound to the consented (old) resource, equally dead at /mcp.
		rotated := newTestService(st, "https://bb-new.example.com")
		second := tokenOK(t, rotated, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {first.RefreshToken},
			"client_id":     {testClientID},
		})
		require.Equal(t, http.StatusUnauthorized, mcpStatus(t, st, "https://bb-new.example.com", second.AccessToken))

		// Recovery is a fresh consent under the new URL — here without a
		// resource parameter, so this also proves the defaulted binding works
		// end to end.
		codeB := consentOK(t, rotated, url.Values{})
		third := tokenOK(t, rotated, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {codeB},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
		})
		require.NotEqual(t, http.StatusUnauthorized, mcpStatus(t, st, "https://bb-new.example.com", third.AccessToken))
	})

	t.Run("the /mcp middleware trusts the workspace-setting external URL tier", func(t *testing.T) {
		// The audience matrix and the rotation chain drive the --external-url
		// flag tier only; the normal self-hosted shape configures the URL in
		// Settings. A regression that silently drops the setting tier from the
		// middleware would 401 every MCP request on such deployments while the
		// consent-side tests stay green.
		_, err := st.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_WORKSPACE_PROFILE,
			Workspace: testWorkspace,
			Value:     &storepb.WorkspaceProfileSetting{ExternalUrl: "https://bb.example.com"},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := st.UpsertSetting(ctx, &store.SettingMessage{
				Name:      storepb.SettingName_WORKSPACE_PROFILE,
				Workspace: testWorkspace,
				Value:     &storepb.WorkspaceProfileSetting{},
			})
			require.NoError(t, err)
		})

		fromSetting := newTestService(st, "")
		code := consentOK(t, fromSetting, url.Values{"resource": {testResource}})
		got := tokenOK(t, fromSetting, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
		})
		require.NotEqual(t, http.StatusUnauthorized, mcpStatus(t, st, "", got.AccessToken),
			"an empty flag must resolve the audience from the workspace setting")
	})

	t.Run("an MCP token still authenticates on the general API (audit-only this release)", func(t *testing.T) {
		// Until PR 4's private transport, /mcp tool calls forward the inbound
		// bearer to the general API, so this admission is what keeps every MCP
		// tool call working. PR 5 flips rejectMCPTokenOnGeneralAPI to retire it;
		// this test is the tripwire against flipping it early by accident.
		_, err := st.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Workspace: testWorkspace,
			Member:    common.FormatUserEmail(testUserEmail),
			Roles:     []string{"roles/workspaceMember"},
		})
		require.NoError(t, err)

		code := consentOK(t, configured, url.Values{"resource": {testResource}})
		got := tokenOK(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
		})

		interceptor := auth.New(st, testSecret, nil, nil, &config.Profile{})
		user, workspaceID, _, err := interceptor.AuthenticateToken(ctx, got.AccessToken)
		require.NoError(t, err)
		require.Equal(t, testUserEmail, user.Email)
		require.Equal(t, testWorkspace, workspaceID)
	})

	t.Run("a requested scope set is consented as its maximum", func(t *testing.T) {
		// Distinct from the repeated-parameter case above: one `scope` value
		// naming both tiers, which is what the v1 bootstrap produces — the 401
		// challenge advertises every mode pre-authentication, so clients ask for
		// all of them. The set resolves to one stored mode; a multi-mode string
		// must never reach the grant record.
		code := consentOK(t, configured, url.Values{"scope": {"mcp:read-only mcp:read-write"}})

		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Equal(t, "mcp:read-write", got.Config.Scope)

		// And the client can keep sending the set it asked for: it normalizes to
		// the same mode, so the exchange matches instead of failing.
		first := tokenOK(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
			"scope":         {"mcp:read-only mcp:read-write"},
		})
		require.Equal(t, "mcp:read-write", first.Scope)

		stored, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(first.RefreshToken))
		require.NoError(t, err)
		require.NotNil(t, stored)
		require.Equal(t, "mcp:read-write", stored.Config.GetScope())
	})
}

func newTestService(st *store.Store, externalURL string) *Service {
	return NewService(st, &config.Profile{ExternalURL: externalURL}, testSecret)
}

// consent posts an approved consent form and returns the callback URL the
// handler redirects the browser to (carrying either `code` or `error`).
func consent(t *testing.T, s *Service, extra url.Values) *url.URL {
	t.Helper()

	challenge := sha256.Sum256([]byte(testCodeVerifier))
	form := url.Values{
		"client_id":             {testClientID},
		"redirect_uri":          {testRedirectURI},
		"state":                 {"state-1"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"action":                {"allow"},
	}
	for name, values := range extra {
		form[name] = values
	}

	sessionToken, err := auth.GenerateAccessToken(testUserEmail, testWorkspace, testSecret, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/oauth2/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	rec := httptest.NewRecorder()

	e := echo.New()
	s.RegisterRoutes(e)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	return redirectTargetOf(t, rec.Body.String())
}

// consentOK asserts the consent succeeded and returns the issued code.
func consentOK(t *testing.T, s *Service, extra url.Values) string {
	t.Helper()
	redirect := consent(t, s, extra)
	require.Empty(t, redirect.Query().Get("error"), redirect.Query().Get("error_description"))
	code := redirect.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

func postToken(t *testing.T, s *Service, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/oauth2/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	e := echo.New()
	s.RegisterRoutes(e)
	e.ServeHTTP(rec, req)
	return rec
}

func tokenOK(t *testing.T, s *Service, form url.Values) tokenResponse {
	t.Helper()
	rec := postToken(t, s, form)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.NotEmpty(t, got.AccessToken)
	return got
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body["error"]
}

func errorDescription(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body["error_description"]
}

// mcpStatus posts an MCP initialize request to a freshly wired /mcp endpoint
// whose trusted external URL is externalURL, and returns the HTTP status. 401
// means the auth middleware refused the token; anything else means the token
// got through to the MCP handler.
func mcpStatus(t *testing.T, st *store.Store, externalURL, accessToken string) int {
	t.Helper()
	srv, err := mcp.NewServer(st, &config.Profile{ExternalURL: externalURL}, testSecret, nil)
	require.NoError(t, err)
	e := echo.New()
	srv.RegisterRoutes(e)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec.Code
}

// redirectTargetOf pulls the callback URL out of the meta-refresh page the
// authorize handler returns (it renders HTML rather than a 302 to sidestep CSP
// form-action restrictions).
func redirectTargetOf(t *testing.T, body string) *url.URL {
	t.Helper()
	matches := regexp.MustCompile(`content="0;url=([^"]+)"`).FindStringSubmatch(body)
	require.Len(t, matches, 2, "expected a meta-refresh redirect, got: %s", body)
	parsed, err := url.Parse(html.UnescapeString(matches[1]))
	require.NoError(t, err)
	return parsed
}
