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

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestNewIdentityProvider(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		config      IdentityProviderConfig
		containsErr string
	}{
		{
			name: "no issuer",
			config: IdentityProviderConfig{
				Issuer:       "",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "sub",
				},
			},
			containsErr: `the field "issuer" is empty but required`,
		},
		{
			name: "no clientId",
			config: IdentityProviderConfig{
				Issuer:       "https://oidc.example.com",
				ClientID:     "",
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "sub",
				},
			},
			containsErr: `the field "clientId" is empty but required`,
		},
		{
			name: "no clientSecret",
			config: IdentityProviderConfig{
				Issuer:       "https://oidc.example.com",
				ClientID:     "test-client-id",
				ClientSecret: "",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "sub",
				},
			},
			containsErr: `the field "clientSecret" is empty but required`,
		},
		{
			name: "no fieldMapping.identifier",
			config: IdentityProviderConfig{
				Issuer:       "https://oidc.example.com",
				ClientID:     "test-client-id",
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

func newMockServer(t *testing.T, tls bool, clientID, code, accessToken, nonce string, userinfo []byte) *httptest.Server {
	mux := http.NewServeMux()

	var openidConfig map[string]any
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
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

		require.Equal(t, code, vals.Get("code"))
		require.Equal(t, "authorization_code", vals.Get("grant_type"))
		require.Equal(t, "https://example.com/oidc/callback", vals.Get("redirect_uri"))

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"refresh_token": "test-refresh-token",
			"expires_in":    3600,
			"id_token":      rawIDToken,
		})
		require.NoError(t, err)
	})
	mux.HandleFunc("/oauth2/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(userinfo)
		require.NoError(t, err)
	})

	rs256, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	mux.HandleFunc("/oauth/discovery/keys", func(w http.ResponseWriter, r *http.Request) {
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
			"aud":   clientID,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"nonce": nonce,
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
			"sub":   testSubject,
			"name":  testName,
			"email": testEmail,
		},
	)
	require.NoError(t, err)

	s := newMockServer(t, false, testClientID, testCode, testAccessToken, testNonce, userinfo)
	oidc, err := NewIdentityProvider(
		ctx,
		IdentityProviderConfig{
			Issuer:       s.URL,
			ClientID:     testClientID,
			ClientSecret: "test-client-secret",
			FieldMapping: &storepb.FieldMapping{
				Identifier:  "sub",
				DisplayName: "name",
				Email:       "email",
			},
		},
	)
	require.NoError(t, err)

	oauthToken, err := oidc.ExchangeToken(ctx, "https://example.com/oidc/callback", testCode)
	require.NoError(t, err)
	require.Equal(t, testAccessToken, oauthToken.AccessToken)

	userInfo, err := oidc.UserInfo(ctx, oauthToken, testNonce)
	require.NoError(t, err)

	wantUserInfo := &storepb.IdentityProviderUserInfo{
		Identifier:  testSubject,
		DisplayName: testName,
		Email:       testEmail,
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
			"sub":   testSubject,
			"name":  testName,
			"email": testEmail,
		},
	)
	require.NoError(t, err)

	t.Run("verify TLS", func(t *testing.T) {
		s := newMockServer(t, true, testClientID, testCode, testAccessToken, testNonce, userinfo)
		_, err := NewIdentityProvider(
			ctx,
			IdentityProviderConfig{
				Issuer:       s.URL,
				ClientID:     testClientID,
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier:  "sub",
					DisplayName: "name",
					Email:       "email",
				},
			},
		)
		assert.ErrorContains(t, err, "x509: certificate signed by unknown authority")
	})

	t.Run("skip TLS verify", func(t *testing.T) {
		s := newMockServer(t, true, testClientID, testCode, testAccessToken, testNonce, userinfo)
		oidc, err := NewIdentityProvider(
			ctx,
			IdentityProviderConfig{
				Issuer:       s.URL,
				ClientID:     testClientID,
				ClientSecret: "test-client-secret",
				FieldMapping: &storepb.FieldMapping{
					Identifier:  "sub",
					DisplayName: "name",
					Email:       "email",
				},
				SkipTLSVerify: true,
			},
		)
		require.NoError(t, err)

		oauthToken, err := oidc.ExchangeToken(ctx, "https://example.com/oidc/callback", testCode)
		require.NoError(t, err)
		require.Equal(t, testAccessToken, oauthToken.AccessToken)

		userInfo, err := oidc.UserInfo(ctx, oauthToken, testNonce)
		require.NoError(t, err)

		wantUserInfo := &storepb.IdentityProviderUserInfo{
			Identifier:  testSubject,
			DisplayName: testName,
			Email:       testEmail,
		}
		assert.Equal(t, wantUserInfo, userInfo)
	})
}
