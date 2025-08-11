package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestNewIdentityProvider(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		config      *storepb.OIDCIdentityProviderConfig
		containsErr string
	}{
		{
			name: "no issuer",
			config: &storepb.OIDCIdentityProviderConfig{
				Issuer:       "",
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "sub",
				},
			},
			containsErr: `the field "issuer" is empty but required`,
		},
		{
			name: "no clientId",
			config: &storepb.OIDCIdentityProviderConfig{
				Issuer:       "https://oidc.example.com",
				ClientId:     "",
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "sub",
				},
			},
			containsErr: `the field "clientId" is empty but required`,
		},
		{
			name: "no clientSecret",
			config: &storepb.OIDCIdentityProviderConfig{
				Issuer:       "https://oidc.example.com",
				ClientId:     "test-client-id",
				ClientSecret: "",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "sub",
				},
			},
			containsErr: `the field "clientSecret" is empty but required`,
		},
		{
			name: "no fieldMapping.identifier",
			config: &storepb.OIDCIdentityProviderConfig{
				Issuer:       "https://oidc.example.com",
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					DisplayName: "name",
				},
			},
			containsErr: `the field "fieldMapping.identifier" is empty but required`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewIdentityProvider(ctx, test.config)
			assert.ErrorContains(t, err, test.containsErr)
		})
	}
}

func newMockServer(t *testing.T, tls bool, userinfo []byte) *httptest.Server {
	const (
		testClientID    = "test-client-id"
		testNonce       = "test-nonce"
		testCode        = "test-code"
		testAccessToken = "test-access-token"
	)
	mux := http.NewServeMux()

	var openidConfig map[string]any
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(openidConfig)
		require.NoError(t, err)
	})

	var rawIDToken string
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)

		require.Equal(t, testCode, vals.Get("code"))
		require.Equal(t, "authorization_code", vals.Get("grant_type"))
		require.Equal(t, "https://example.com/oidc/callback", vals.Get("redirect_uri"))

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  testAccessToken,
			"token_type":    "Bearer",
			"refresh_token": "test-refresh-token",
			"expires_in":    3600,
			"id_token":      rawIDToken,
		})
		require.NoError(t, err)
	})
	mux.HandleFunc("/oauth2/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(userinfo)
		require.NoError(t, err)
	})

	rs256, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	mux.HandleFunc("/oauth/discovery/keys", func(w http.ResponseWriter, _ *http.Request) {
		key, err := jwk.FromRaw(rs256.PublicKey)
		require.NoError(t, err)

		err = key.Set(jwk.KeyUsageKey, "sig")
		require.NoError(t, err)
		err = key.Set(jwk.AlgorithmKey, "RS256")
		require.NoError(t, err)
		marshalledKey, err := json.Marshal(key)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]any{
			"keys": []json.RawMessage{marshalledKey},
		})
		require.NoError(t, err)
	})

	var s *httptest.Server
	if tls {
		s = httptest.NewTLSServer(mux)
	} else {
		s = httptest.NewServer(mux)
	}

	openidConfig = map[string]any{
		"issuer":                                s.URL,
		"authorization_endpoint":                s.URL + "/oauth2/authorize",
		"token_endpoint":                        s.URL + "/oauth2/token",
		"userinfo_endpoint":                     s.URL + "/oauth2/userinfo",
		"jwks_uri":                              s.URL + "/oauth/discovery/keys",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		jwt.MapClaims{
			"iss":   s.URL,
			"sub":   "123456789",
			"aud":   testClientID,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"nonce": testNonce,
		},
	)
	rawIDToken, err = token.SignedString(rs256)
	require.NoError(t, err)
	return s
}

func TestIdentityProvider(t *testing.T) {
	ctx := context.Background()

	const (
		testClientID    = "test-client-id"
		testNonce       = "test-nonce"
		testCode        = "test-code"
		testAccessToken = "test-access-token"
		testSubject     = "123456789"
		testName        = "John Doe"
		testEmail       = "john.doe@example.com"
	)
	userinfo, err := json.Marshal(
		map[string]any{
			"sub":    testSubject,
			"name":   testName,
			"email":  testEmail,
			"groups": []any{"Dev", "Admin"},
		},
	)
	require.NoError(t, err)

	s := newMockServer(t, false, userinfo)
	oidc, err := NewIdentityProvider(
		ctx,
		&storepb.OIDCIdentityProviderConfig{
			Issuer:       s.URL,
			ClientId:     testClientID,
			ClientSecret: "test-client-secret",
			FieldMapping: &storepb.FieldMapping{
				Identifier:  "sub",
				DisplayName: "name",
				Groups:      "groups",
			},
		},
	)
	require.NoError(t, err)

	oauthToken, err := oidc.ExchangeToken(ctx, "https://example.com/oidc/callback", testCode)
	require.NoError(t, err)
	require.Equal(t, testAccessToken, oauthToken.AccessToken)

	userInfo, _, err := oidc.UserInfo(ctx, oauthToken, testNonce)
	require.NoError(t, err)

	wantUserInfo := &storepb.IdentityProviderUserInfo{
		Identifier:  testSubject,
		DisplayName: testName,
		Groups:      []string{"Dev", "Admin"},
		HasGroups:   true,
	}
	assert.Equal(t, wantUserInfo, userInfo)
}

func TestIdentityProvider_SelfSigned(t *testing.T) {
	ctx := context.Background()

	const (
		testClientID    = "test-client-id"
		testNonce       = "test-nonce"
		testCode        = "test-code"
		testAccessToken = "test-access-token"
		testSubject     = "123456789"
		testName        = "John Doe"
		testEmail       = "john.doe@example.com"
	)
	userinfo, err := json.Marshal(
		map[string]any{
			"sub":    testSubject,
			"name":   testName,
			"email":  testEmail,
			"groups": []any{"Dev", "Admin"},
		},
	)
	require.NoError(t, err)

	t.Run("verify TLS", func(t *testing.T) {
		s := newMockServer(t, true, userinfo)
		_, err := NewIdentityProvider(
			ctx,
			&storepb.OIDCIdentityProviderConfig{
				Issuer:       s.URL,
				ClientId:     testClientID,
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier:  "sub",
					DisplayName: "name",
					Groups:      "groups",
				},
			},
		)
		assert.ErrorContains(t, err, "x509: certificate signed by unknown authority")
	})

	t.Run("skip TLS verify", func(t *testing.T) {
		s := newMockServer(t, true, userinfo)
		oidc, err := NewIdentityProvider(
			ctx,
			&storepb.OIDCIdentityProviderConfig{
				Issuer:       s.URL,
				ClientId:     testClientID,
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier:  "sub",
					DisplayName: "name",
					Groups:      "groups",
				},
				SkipTlsVerify: true,
			},
		)
		require.NoError(t, err)

		oauthToken, err := oidc.ExchangeToken(ctx, "https://example.com/oidc/callback", testCode)
		require.NoError(t, err)
		require.Equal(t, testAccessToken, oauthToken.AccessToken)

		userInfo, _, err := oidc.UserInfo(ctx, oauthToken, testNonce)
		require.NoError(t, err)

		wantUserInfo := &storepb.IdentityProviderUserInfo{
			Identifier:  testSubject,
			DisplayName: testName,
			Groups:      []string{"Dev", "Admin"},
			HasGroups:   true,
		}
		assert.Equal(t, wantUserInfo, userInfo)
	})
}

func TestIdentityProvider_GroupsParsing(t *testing.T) {
	ctx := context.Background()

	const (
		testClientID    = "test-client-id"
		testNonce       = "test-nonce"
		testCode        = "test-code"
		testAccessToken = "test-access-token"
		testSubject     = "123456789"
		testName        = "John Doe"
		testEmail       = "john.doe@example.com"
	)

	tests := []struct {
		name           string
		groupsClaim    any
		expectedGroups []string
	}{
		{
			name:           "normal array groups",
			groupsClaim:    []any{"Dev", "Admin"},
			expectedGroups: []string{"Dev", "Admin"},
		},
		{
			name:           "2D array groups (JSON-encoded string in array)",
			groupsClaim:    []any{`["a1b2c3d4-e5f6-7890-abcd-ef1234567890","f9e8d7c6-b5a4-3210-9876-543210fedcba"]`},
			expectedGroups: []string{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", "f9e8d7c6-b5a4-3210-9876-543210fedcba"},
		},
		{
			name:           "mixed groups (normal strings and JSON)",
			groupsClaim:    []any{"Admin", `["group1","group2"]`, "Dev"},
			expectedGroups: []string{"Admin", "group1", "group2", "Dev"},
		},
		{
			name:           "malformed JSON (should be treated as string)",
			groupsClaim:    []any{`["invalid json"`},
			expectedGroups: []string{`["invalid json"`},
		},
		{
			name:           "empty JSON array",
			groupsClaim:    []any{`[]`},
			expectedGroups: nil,
		},
		{
			name:           "non-JSON string starting with bracket",
			groupsClaim:    []any{"[not-json]"},
			expectedGroups: []string{"[not-json]"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userinfo, err := json.Marshal(
				map[string]any{
					"sub":    testSubject,
					"name":   testName,
					"email":  testEmail,
					"groups": test.groupsClaim,
				},
			)
			require.NoError(t, err)

			s := newMockServer(t, false, userinfo)
			oidc, err := NewIdentityProvider(
				ctx,
				&storepb.OIDCIdentityProviderConfig{
					Issuer:       s.URL,
					ClientId:     testClientID,
					ClientSecret: "test-client-secret",
					FieldMapping: &storepb.FieldMapping{
						Identifier:  "sub",
						DisplayName: "name",
						Groups:      "groups",
					},
				},
			)
			require.NoError(t, err)

			oauthToken, err := oidc.ExchangeToken(ctx, "https://example.com/oidc/callback", testCode)
			require.NoError(t, err)

			userInfo, _, err := oidc.UserInfo(ctx, oauthToken, testNonce)
			require.NoError(t, err)

			assert.Equal(t, test.expectedGroups, userInfo.Groups)
			// HasGroups is true whenever groups field exists in claims, regardless of content
			assert.True(t, userInfo.HasGroups)
		})
	}
}

func TestGetOpenIDConfigration(t *testing.T) {
	tests := []struct {
		issuer   string
		response *OpenIDConfigurationResponse
	}{
		{
			issuer: "https://accounts.google.com",
			response: &OpenIDConfigurationResponse{
				AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.issuer, func(t *testing.T) {
			response, err := GetOpenIDConfiguration(test.issuer, false)
			require.NoError(t, err)
			assert.Equal(t, test.response, response)
		})
	}
}
