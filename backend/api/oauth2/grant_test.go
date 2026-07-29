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

	t.Run("no resource and no scope still consents (clients that predate both)", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{})

		got, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.Empty(t, got.Config.Resource)
		require.Empty(t, got.Config.Scope)
	})

	rejections := []struct {
		name     string
		params   url.Values
		wantCode string
	}{
		{"resource on another host", url.Values{"resource": {"https://evil.example.com/mcp"}}, "invalid_target"},
		{"noncanonical resource", url.Values{"resource": {"https://bb.example.com:443/mcp"}}, "invalid_target"},
		{"multiple resources", url.Values{"resource": {testResource, testResource}}, "invalid_target"},
		{"unknown scope", url.Values{"scope": {"mcp:admin"}}, "invalid_scope"},
		{"multiple scopes", url.Values{"scope": {"mcp:read-only", "mcp:read-write"}}, "invalid_scope"},
	}
	for _, tc := range rejections {
		t.Run("consent rejected: "+tc.name, func(t *testing.T) {
			redirect := consent(t, configured, tc.params)
			require.Equal(t, tc.wantCode, redirect.Query().Get("error"))
		})
	}

	t.Run("no configured external URL: a resource request gets the actionable setup error", func(t *testing.T) {
		// The ship gate of proposal v2 §6.2 — self-hosted instances running on
		// the request-Host fallback break here, and the error has to tell the
		// admin exactly what to set rather than fail generically.
		unconfigured := newTestService(st, "")
		redirect := consent(t, unconfigured, url.Values{"resource": {testResource}})
		require.Equal(t, "server_error", redirect.Query().Get("error"))
		description := redirect.Query().Get("error_description")
		require.Contains(t, description, "--external-url")
		require.Contains(t, description, "External URL")
	})

	t.Run("no configured external URL: a request without a resource still works", func(t *testing.T) {
		unconfigured := newTestService(st, "")
		require.NotEmpty(t, consentOK(t, unconfigured, url.Values{}),
			"resource binding is opt-in, so today's clients must not break on an unconfigured instance")
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
		require.Equal(t, testResource, stored.Resource)
		require.Equal(t, "mcp:read-write", stored.Scope)

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
		require.Equal(t, testResource, rotated.Resource)
		require.Equal(t, "mcp:read-write", rotated.Scope)
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
		require.Equal(t, testResource, stored.Resource)

		second := tokenOK(t, configured, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {first.RefreshToken},
			"client_id":     {testClientID},
			"resource":      {"https://bb.example.com"},
		})
		rotated, err := st.GetOAuth2RefreshToken(ctx, testClientID, auth.HashToken(second.RefreshToken))
		require.NoError(t, err)
		require.NotNil(t, rotated)
		require.Equal(t, testResource, rotated.Resource,
			"the canonical form must survive rotation, not drift back to what the client sent")
	})

	t.Run("token exchange rejects a resource the grant never consented to", func(t *testing.T) {
		code := consentOK(t, configured, url.Values{})

		rec := postToken(t, configured, url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {testRedirectURI},
			"code_verifier": {testCodeVerifier},
			"client_id":     {testClientID},
			"resource":      {testResource},
		})
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "invalid_target", errorCode(t, rec))

		// Rejected before consumption: the code survives for a correct retry.
		still, err := st.GetOAuth2AuthorizationCode(ctx, testClientID, code)
		require.NoError(t, err)
		require.NotNil(t, still, "a rejected exchange must not burn the authorization code")
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
