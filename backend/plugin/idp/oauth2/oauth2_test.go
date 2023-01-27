package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestNewIdentityProvider(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		config      *storepb.OAuth2IdentityProviderConfig
		containsErr string
	}{
		{
			name: "no tokenUrl",
			config: &storepb.OAuth2IdentityProviderConfig{
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthUrl:      "",
				TokenUrl:     "",
				UserInfoUrl:  "https://example.com/api/user",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "login",
				},
			},
			containsErr: `the field "tokenUrl" is empty but required`,
		},
		{
			name: "no userInfoUrl",
			config: &storepb.OAuth2IdentityProviderConfig{
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthUrl:      "",
				TokenUrl:     "https://example.com/token",
				UserInfoUrl:  "",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "login",
				},
			},
			containsErr: `the field "userInfoUrl" is empty but required`,
		},
		{
			name: "no field mapping identifier",
			config: &storepb.OAuth2IdentityProviderConfig{
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthUrl:      "",
				TokenUrl:     "https://example.com/token",
				UserInfoUrl:  "https://example.com/api/user",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "",
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

func newMockServer(t *testing.T, code, accessToken string, userinfo []byte) *httptest.Server {
	mux := http.NewServeMux()

	var rawIDToken string
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		vals, err := url.ParseQuery(string(body))
		require.NoError(t, err)

		require.Equal(t, code, vals.Get("code"))
		require.Equal(t, "authorization_code", vals.Get("grant_type"))

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

	s := httptest.NewServer(mux)

	return s
}

func TestIdentityProvider(t *testing.T) {
	ctx := context.Background()

	const (
		testClientID    = "test-client-id"
		testCode        = "test-code"
		testAccessToken = "test-access-token"
		testSubject     = "123456789"
		testName        = "John Doe"
		testEmail       = "john.doe@example.com"
	)
	userInfo, err := json.Marshal(
		map[string]any{
			"sub":   testSubject,
			"name":  testName,
			"email": testEmail,
		},
	)
	require.NoError(t, err)

	s := newMockServer(t, testCode, testAccessToken, userInfo)

	oauth2, err := NewIdentityProvider(
		ctx,
		&storepb.OAuth2IdentityProviderConfig{
			ClientId:     testClientID,
			ClientSecret: "test-client-secret",
			TokenUrl:     fmt.Sprintf("%s/oauth2/token", s.URL),
			UserInfoUrl:  fmt.Sprintf("%s/oauth2/userinfo", s.URL),
			FieldMapping: &storepb.FieldMapping{
				Identifier:  "sub",
				DisplayName: "name",
				Email:       "email",
			},
		},
	)
	require.NoError(t, err)

	oauthToken, err := oauth2.ExchangeToken(ctx, testCode)
	require.NoError(t, err)
	require.Equal(t, testAccessToken, oauthToken)

	userInfoResult, err := oauth2.UserInfo(ctx, oauthToken)
	require.NoError(t, err)

	wantUserInfo := &storepb.IdentityProviderUserInfo{
		Identifier:  testSubject,
		DisplayName: testName,
		Email:       testEmail,
		Raw:         string(json.RawMessage(userInfo)),
	}
	assert.Equal(t, wantUserInfo, userInfoResult)
}
