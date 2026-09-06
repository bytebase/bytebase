package tests

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	oauth2TestRedirectURI = "http://localhost/cb"
	// 43-128 chars per RFC 7636.
	oauth2TestVerifier = "e2eVerifier_e2eVerifier_e2eVerifier_e2eVerifier"
)

// registerOAuth2Client registers a public client through RFC 7591 dynamic
// registration, allowed both grant types, and returns its ID.
func registerOAuth2Client(t *testing.T, ctl *controller) string {
	t.Helper()
	resp, err := (&http.Client{}).Post(ctl.rootURL+"/api/oauth2/register", "application/json",
		strings.NewReader(`{"client_name":"bb-e2e","redirect_uris":["http://localhost/cb"],"grant_types":["authorization_code","refresh_token"],"token_endpoint_auth_method":"none"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	var reg struct {
		ClientID string `json:"client_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reg))
	require.NotEmpty(t, reg.ClientID)
	return reg.ClientID
}

// oauth2ConsentForm is an approved consent for clientID with PKCE filled in.
// extra adds or overrides parameters; a repeated parameter is sent repeated.
func oauth2ConsentForm(clientID string, extra url.Values) url.Values {
	challenge := sha256.Sum256([]byte(oauth2TestVerifier))
	form := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {oauth2TestRedirectURI},
		"state":                 {"e2e-state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"action":                {"allow"},
	}
	for name, values := range extra {
		form[name] = values
	}
	return form
}

// postOAuth2Consent posts a consent form as the bearer's user and returns the
// raw answer, so a caller can read either a refusal page or the meta-refresh
// carrying the callback.
func postOAuth2Consent(t *testing.T, ctl *controller, bearer string, form url.Values) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ctl.rootURL+"/api/oauth2/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

// oauth2Callback pulls the callback URL out of the meta-refresh page the
// authorize handler answers with (a page rather than a 302, to sidestep CSP
// form-action restrictions); it carries either `code` or `error`.
func oauth2Callback(t *testing.T, body string) *url.URL {
	t.Helper()
	matches := regexp.MustCompile(`content="0;url=([^"]+)"`).FindStringSubmatch(body)
	require.Len(t, matches, 2, "expected a meta-refresh redirect, got: %s", body)
	callback, err := url.Parse(html.UnescapeString(matches[1]))
	require.NoError(t, err)
	return callback
}

// consentOAuth2 posts an approved consent and returns its callback.
func consentOAuth2(t *testing.T, ctl *controller, bearer, clientID string, extra url.Values) *url.URL {
	t.Helper()
	status, body := postOAuth2Consent(t, ctl, bearer, oauth2ConsentForm(clientID, extra))
	require.Equal(t, http.StatusOK, status, body)
	return oauth2Callback(t, body)
}

// consentOAuth2Code asserts the consent was granted and returns the code.
func consentOAuth2Code(t *testing.T, ctl *controller, bearer, clientID string, extra url.Values) string {
	t.Helper()
	callback := consentOAuth2(t, ctl, bearer, clientID, extra)
	require.Empty(t, callback.Query().Get("error"), callback.Query().Get("error_description"))
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

func oauth2ExchangeForm(clientID, code string, extra url.Values) url.Values {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {oauth2TestRedirectURI},
		"code_verifier": {oauth2TestVerifier},
		"client_id":     {clientID},
	}
	for name, values := range extra {
		form[name] = values
	}
	return form
}

func oauth2RefreshForm(clientID, refreshToken string, extra url.Values) url.Values {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	for name, values := range extra {
		form[name] = values
	}
	return form
}

// postOAuth2Token posts to the token endpoint and returns the status and the
// decoded JSON body, which carries either the tokens or an RFC 6749 error.
func postOAuth2Token(t *testing.T, ctl *controller, form url.Values) (int, map[string]any) {
	t.Helper()
	resp, err := (&http.Client{}).PostForm(ctl.rootURL+"/api/oauth2/token", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &body), string(raw))
	return resp.StatusCode, body
}

// oauth2TokenOK asserts the token endpoint issued and returns the body.
func oauth2TokenOK(t *testing.T, ctl *controller, form url.Values) map[string]any {
	t.Helper()
	status, body := postOAuth2Token(t, ctl, form)
	require.Equal(t, http.StatusOK, status, body)
	require.NotEmpty(t, body["access_token"])
	return body
}

func oauth2Error(t *testing.T, ctl *controller, form url.Values, wantStatus int) (string, string) {
	t.Helper()
	status, body := postOAuth2Token(t, ctl, form)
	require.Equal(t, wantStatus, status, body)
	code := stringField(t, body, "error")
	description := stringField(t, body, "error_description")
	return code, description
}

// stringField reads a string out of a decoded token response, failing the
// test if it is missing or not a string.
func stringField(t *testing.T, body map[string]any, key string) string {
	t.Helper()
	value, ok := body[key].(string)
	require.True(t, ok, "%s is not a string in %v", key, body)
	return value
}

// storedGrant is the consented state a code or refresh row carries.
type storedGrant struct {
	Resource string `json:"resource"`
	Scope    string `json:"scope"`
}

func storedAuthorizationCode(ctx context.Context, t *testing.T, db *sql.DB, clientID, code string) *storedGrant {
	t.Helper()
	return storedGrantRow(ctx, t, db, `SELECT config FROM oauth2_authorization_code WHERE client_id = $1 AND code = $2`, clientID, code)
}

func storedRefreshToken(ctx context.Context, t *testing.T, db *sql.DB, clientID, refreshToken string) *storedGrant {
	t.Helper()
	return storedGrantRow(ctx, t, db, `SELECT config FROM oauth2_refresh_token WHERE client_id = $1 AND token_hash = $2`, clientID, auth.HashToken(refreshToken))
}

func storedGrantRow(ctx context.Context, t *testing.T, db *sql.DB, query string, args ...any) *storedGrant {
	t.Helper()
	var raw []byte
	err := db.QueryRowContext(ctx, query, args...).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil
	}
	require.NoError(t, err)
	grant := &storedGrant{}
	require.NoError(t, json.Unmarshal(raw, grant))
	return grant
}

// TestOAuth2GrantLifecycle drives the whole consent → token → refresh path on a
// live server. The pure rule space — resource canonicalization, scope
// canonicalization, consented-value checks — is table-tested in
// backend/api/oauth2; what this pins is the wiring, which is where a
// resource/scope binding silently disappears: the code row must carry what was
// consented, the token endpoint must compare against it, and every refresh
// must re-issue the same grant rather than whatever the client asks for.
//
//nolint:tparallel // Subtests share one server lifecycle.
func TestOAuth2GrantLifecycle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{Name: "workspaces/-"}))
	a.NoError(err)
	workspaceID := strings.TrimPrefix(workspace.Msg.Name, "workspaces/")

	bearer := ctl.authInterceptor.token
	clientID := registerOAuth2Client(t, ctl)
	canonical := ctl.rootURL + "/mcp"

	t.Run("matching resource and known scope are consented", func(t *testing.T) {
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"resource": {canonical}, "scope": {"mcp:read-only"}})
		require.Equal(t, &storedGrant{Resource: canonical, Scope: "mcp:read-only"}, storedAuthorizationCode(ctx, t, db, clientID, code))
	})

	t.Run("resource is stored canonicalized, not as sent", func(t *testing.T) {
		shouted := strings.ToUpper(strings.TrimPrefix(ctl.rootURL, "http://"))
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"resource": {"HTTP://" + shouted + "/mcp/"}})
		require.Equal(t, canonical, storedAuthorizationCode(ctx, t, db, clientID, code).Resource,
			"the token endpoint compares against this value, so it must already be canonical")
	})

	t.Run("a bare origin is consented as the canonical MCP resource", func(t *testing.T) {
		// Accepting the bare origin (our unsuffixed RFC 9728 document publishes
		// it) must not put a second spelling in the grant: the token audience
		// is bound to the stored value, and two spellings would mean two
		// audiences to honor at /mcp for as long as any grant lives.
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"resource": {ctl.rootURL}})
		require.Equal(t, canonical, storedAuthorizationCode(ctx, t, db, clientID, code).Resource)
	})

	t.Run("no resource requested: the grant is bound to this server's MCP resource", func(t *testing.T) {
		// Every grant is resource-bound, because its tokens carry the resource
		// as their audience: an unbound grant would mint tokens no audience
		// check accepts. A client that never sends `resource` is defaulted to
		// the only resource this server would accept anyway.
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{})
		got := storedAuthorizationCode(ctx, t, db, clientID, code)
		require.Equal(t, canonical, got.Resource)
		require.Empty(t, got.Scope, "only the resource is defaulted; scope stays exactly as requested")
	})

	for _, tc := range []struct {
		name     string
		params   url.Values
		wantCode string
	}{
		{"resource on another host", url.Values{"resource": {"https://evil.example.com/mcp"}}, "invalid_target"},
		{"resource parameter repeated", url.Values{"resource": {canonical, canonical}}, "invalid_target"},
		{"unknown scope", url.Values{"scope": {"mcp:admin"}}, "invalid_scope"},
		{"scope parameter repeated", url.Values{"scope": {"mcp:read-only", "mcp:read-write"}}, "invalid_scope"},
	} {
		t.Run("consent rejected: "+tc.name, func(t *testing.T) {
			callback := consentOAuth2(t, ctl, bearer, clientID, tc.params)
			require.Equal(t, tc.wantCode, callback.Query().Get("error"))
			require.Empty(t, callback.Query().Get("code"))
		})
	}

	t.Run("token exchange and refresh carry the grant forward unchanged", func(t *testing.T) {
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"resource": {canonical}, "scope": {"mcp:read-write"}})

		// Exchange, echoing the resource back the way RFC 8707 clients do.
		first := oauth2TokenOK(t, ctl, oauth2ExchangeForm(clientID, code, url.Values{"resource": {canonical}}))
		require.Equal(t, "mcp:read-write", first["scope"], "the consented scope must be echoed (RFC 6749 §5.1)")
		refreshToken := stringField(t, first, "refresh_token")
		require.NotEmpty(t, refreshToken)
		require.Equal(t, &storedGrant{Resource: canonical, Scope: "mcp:read-write"}, storedRefreshToken(ctx, t, db, clientID, refreshToken))

		// A refresh that names a different scope or resource is refused, and
		// refusal happens before consumption so the client can retry correctly.
		errCode, _ := oauth2Error(t, ctl, oauth2RefreshForm(clientID, refreshToken, url.Values{"scope": {"mcp:read-only"}}), http.StatusBadRequest)
		require.Equal(t, "invalid_scope", errCode)
		errCode, _ = oauth2Error(t, ctl, oauth2RefreshForm(clientID, refreshToken, url.Values{"resource": {"https://evil.example.com/mcp"}}), http.StatusBadRequest)
		require.Equal(t, "invalid_target", errCode)

		// The rejected attempts left the grant usable, and the honest refresh
		// re-issues the same resource and scope.
		second := oauth2TokenOK(t, ctl, oauth2RefreshForm(clientID, refreshToken, nil))
		require.Equal(t, "mcp:read-write", second["scope"])
		rotated := stringField(t, second, "refresh_token")
		require.Equal(t, &storedGrant{Resource: canonical, Scope: "mcp:read-write"}, storedRefreshToken(ctx, t, db, clientID, rotated))
	})

	t.Run("a bare-origin grant refreshes as the canonical resource, named either way", func(t *testing.T) {
		// Consent with the bare origin, then drive the whole exchange naming
		// the bare origin at every step. The stored value stays the canonical
		// MCP URI throughout, the token's audience is that stored value, and
		// the client is never told its own spelling is wrong.
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"resource": {ctl.rootURL + "/"}})
		first := oauth2TokenOK(t, ctl, oauth2ExchangeForm(clientID, code, url.Values{"resource": {ctl.rootURL}}))
		refreshToken := stringField(t, first, "refresh_token")
		require.Equal(t, canonical, storedRefreshToken(ctx, t, db, clientID, refreshToken).Resource)
		accessToken := stringField(t, first, "access_token")
		require.Contains(t, jwtClaims(t, accessToken)["aud"], canonical, "the token is minted from the grant's stored resource")

		second := oauth2TokenOK(t, ctl, oauth2RefreshForm(clientID, refreshToken, url.Values{"resource": {ctl.rootURL}}))
		rotated := stringField(t, second, "refresh_token")
		require.Equal(t, canonical, storedRefreshToken(ctx, t, db, clientID, rotated).Resource,
			"the canonical form must survive rotation, not drift back to what the client sent")
	})

	t.Run("legacy unbound grants are refused at the token endpoint with re-auth guidance", func(t *testing.T) {
		// Rows created before 3.22.1 carry no resource in their config. Tokens
		// are audience-bound to the grant's stored resource, so an unbound
		// grant has nothing valid to mint. invalid_grant is what makes
		// RFC-compliant clients discard the grant and rerun the OAuth flow,
		// and the description points at the reauthorize tool as the in-band
		// recovery.
		challenge := sha256.Sum256([]byte(oauth2TestVerifier))
		const legacyCode = "bb_code_legacy_unbound"
		_, err := db.ExecContext(ctx, `
			INSERT INTO oauth2_authorization_code (code, client_id, user_email, workspace, config, expires_at)
			VALUES ($1, $2, 'demo@example.com', $3, $4, now() + interval '10 minutes')`,
			legacyCode, clientID, workspaceID,
			`{"redirectUri":"http://localhost/cb","codeChallenge":"`+base64.RawURLEncoding.EncodeToString(challenge[:])+`","codeChallengeMethod":"S256"}`)
		require.NoError(t, err)

		code, description := oauth2Error(t, ctl, oauth2ExchangeForm(clientID, legacyCode, nil), http.StatusBadRequest)
		require.Equal(t, "invalid_grant", code)
		require.Contains(t, description, "re-authorize")
		require.Contains(t, description, "reauthorize tool")

		const legacyRefresh = "bb_refresh_legacy_unbound"
		_, err = db.ExecContext(ctx, `
			INSERT INTO oauth2_refresh_token (token_hash, client_id, user_email, workspace, config, expires_at)
			VALUES ($1, $2, 'demo@example.com', $3, '{}', now() + interval '1 day')`,
			auth.HashToken(legacyRefresh), clientID, workspaceID)
		require.NoError(t, err)

		code, description = oauth2Error(t, ctl, oauth2RefreshForm(clientID, legacyRefresh, nil), http.StatusBadRequest)
		require.Equal(t, "invalid_grant", code)
		require.Contains(t, description, "reauthorize tool")
		// Refusal happens before consumption: the row is left for the expiry
		// sweep (or the user's reauthorize) rather than burned by a refresh
		// that issued nothing.
		require.NotNil(t, storedRefreshToken(ctx, t, db, clientID, legacyRefresh))

		// A bound grant with no scope is not legacy — only a missing resource
		// marks the retired population.
		_, boundRefresh, boundClient := mintMCPOAuthTokenWithScope(t, ctl, bearer, "")
		oauth2TokenOK(t, ctl, oauth2RefreshForm(boundClient, boundRefresh, nil))
	})

	t.Run("the reauthorize tool revokes the grant it is called under", func(t *testing.T) {
		// backend/api/mcp only records that the tool asked its store to delete
		// the caller's refresh grants; this is the contract that the real
		// store deletes them, so the token endpoint stops refreshing, and that
		// the bearer the tool was called with is refused from then on.
		accessToken, refreshToken, toolClient := mintMCPOAuthTokenWithScope(t, ctl, bearer, "mcp:read-only")
		session := openMCPSession(ctx, t, ctl, accessToken)
		defer session.Close()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "reauthorize"})
		require.NoError(t, err)
		require.False(t, result.IsError)

		require.Nil(t, storedRefreshToken(ctx, t, db, toolClient, refreshToken))
		code, _ := oauth2Error(t, ctl, oauth2RefreshForm(toolClient, refreshToken, nil), http.StatusBadRequest)
		require.Equal(t, "invalid_grant", code)
		status, _ := postMCP(t, ctl, accessToken)
		require.Equal(t, http.StatusUnauthorized, status)
	})

	t.Run("a requested scope set is consented as its maximum", func(t *testing.T) {
		// One `scope` value naming both tiers, which is what the v1 bootstrap
		// produces — the 401 challenge advertises every mode
		// pre-authentication, so clients ask for all of them. The set resolves
		// to one stored mode; a multi-mode string must never reach the grant
		// record. And the client can keep sending the set it asked for: it
		// normalizes to the same mode, so the exchange matches.
		code := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"scope": {"mcp:read-only mcp:read-write"}})
		require.Equal(t, "mcp:read-write", storedAuthorizationCode(ctx, t, db, clientID, code).Scope)

		first := oauth2TokenOK(t, ctl, oauth2ExchangeForm(clientID, code, url.Values{"scope": {"mcp:read-only mcp:read-write"}}))
		require.Equal(t, "mcp:read-write", first["scope"])
		refreshToken := stringField(t, first, "refresh_token")
		require.Equal(t, "mcp:read-write", storedRefreshToken(ctx, t, db, clientID, refreshToken).Scope)
	})
}

// TestOAuth2IssuanceRechecksTheCeiling pins the half a consent-time check alone
// would miss. The consent decision is captured once and reused at every later
// issuance: an authorization code outlives the consent by ten minutes and a
// refresh token by thirty days, so a grant consented while MCP was on would
// keep minting access tokens long after an admin turned MCP off.
//
// The refusal must read as reversible, not as a dead grant: invalid_grant makes
// a compliant client discard the credential and rerun the OAuth flow, exactly
// what this path retains the credential to avoid. And it must not burn the
// credential it refuses — the ceiling is a toggle, and raising it again must
// restore service without a fresh interactive consent.
func TestOAuth2IssuanceRechecksTheCeiling(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()

	bearer := ctl.authInterceptor.token
	canonical := ctl.rootURL + "/mcp"
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	// Consent while MCP is on, so the grant is genuine and only the ceiling
	// moves underneath it.
	_, refreshToken, clientID := mintMCPOAuthTokenWithScope(t, ctl, bearer, "mcp:read-only")
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_DISABLED))

	code, description := oauth2Error(t, ctl, oauth2RefreshForm(clientID, refreshToken, nil), http.StatusServiceUnavailable)
	a.Equal("temporarily_unavailable", code, "a compliant client discards the grant on invalid_grant, and this one must keep it")
	a.Contains(description, "turned MCP access off")
	a.NotNil(storedRefreshToken(ctx, t, db, clientID, refreshToken), "a refused refresh must not consume the token")

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	oauth2TokenOK(t, ctl, oauth2RefreshForm(clientID, refreshToken, nil))

	// A code minted before the ceiling moved. The consent that produced it was
	// allowed; the exchange is not, and is not burned either.
	fresh := consentOAuth2Code(t, ctl, bearer, clientID, url.Values{"resource": {canonical}, "scope": {"mcp:read-only"}})
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_DISABLED))
	code, _ = oauth2Error(t, ctl, oauth2ExchangeForm(clientID, fresh, nil), http.StatusServiceUnavailable)
	a.Equal("temporarily_unavailable", code)
	a.NotNil(storedAuthorizationCode(ctx, t, db, clientID, fresh), "a refused exchange must not consume the code")

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	oauth2TokenOK(t, ctl, oauth2ExchangeForm(clientID, fresh, nil))
}
