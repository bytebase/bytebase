package command

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
)

func envOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestIDTokenSource(t *testing.T) {
	t.Parallel()

	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer request-token" {
			http.Error(w, "bad request token", http.StatusUnauthorized)
			return
		}
		// The audience is echoed so a case can assert how it was encoded.
		_, _ = w.Write([]byte(`{"value":"github:` + r.URL.Query().Get("api-version") + `:` + r.URL.Query().Get("audience") + `"}`))
	}))
	t.Cleanup(github.Close)
	githubEnv := map[string]string{
		"ACTIONS_ID_TOKEN_REQUEST_URL":   github.URL + "/token?api-version=2.0",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "request-token",
	}

	tests := []struct {
		name      string
		env       map[string]string
		platform  world.JobPlatform
		audience  string
		want      string
		wantError string
	}{
		{
			name:     "the variable is used on any CI",
			env:      map[string]string{"BYTEBASE_ID_TOKEN": "gitlab-token"},
			platform: world.GitLab,
			audience: "bytebase",
			want:     "gitlab-token",
		},
		{
			name: "the variable wins on GitHub",
			env: map[string]string{
				"BYTEBASE_ID_TOKEN":              "from-variable",
				"ACTIONS_ID_TOKEN_REQUEST_URL":   githubEnv["ACTIONS_ID_TOKEN_REQUEST_URL"],
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": githubEnv["ACTIONS_ID_TOKEN_REQUEST_TOKEN"],
			},
			platform: world.GitHub,
			audience: "bytebase",
			want:     "from-variable",
		},
		{
			name:     "GitHub is asked with the audience, keeping the URL's own query",
			env:      githubEnv,
			platform: world.GitHub,
			audience: "https://bytebase.example.com/?a=b&c",
			want:     "github:2.0:https://bytebase.example.com/?a=b&c",
		},
		{
			name:      "no variable off GitHub",
			platform:  world.GitLab,
			audience:  "bytebase",
			wantError: "set BYTEBASE_ID_TOKEN",
		},
		{
			name:      "GitHub job without the permission",
			platform:  world.GitHub,
			audience:  "bytebase",
			wantError: "id-token: write",
		},
		{
			name: "GitHub refuses the request",
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   githubEnv["ACTIONS_ID_TOKEN_REQUEST_URL"],
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "wrong",
			},
			platform:  world.GitHub,
			audience:  "bytebase",
			wantError: "401 Unauthorized: bad request token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, err := newIDTokenSource(envOf(tt.env), tt.platform, tt.audience, 5*time.Second)(t.Context())
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, token)
		})
	}
}
