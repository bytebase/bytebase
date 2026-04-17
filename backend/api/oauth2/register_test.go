package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAllowedDynamicClientRedirectURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{
			name: "allow localhost http",
			uri:  "http://localhost:3000/callback",
			want: true,
		},
		{
			name: "allow localhost https",
			uri:  "https://127.0.0.1:8443/callback",
			want: true,
		},
		{
			name: "allow cursor scheme",
			uri:  "cursor://anysphere.cursor-retrieval/callback",
			want: true,
		},
		{
			name: "allow vscode scheme",
			uri:  "vscode://bytebase.bytebase/callback",
			want: true,
		},
		{
			name: "allow vscode insiders scheme",
			uri:  "vscode-insiders://bytebase.bytebase/callback",
			want: true,
		},
		{
			name: "allow jetbrains gateway scheme",
			uri:  "jetbrains://gateway/bytebase/callback",
			want: true,
		},
		{
			name: "reject non-localhost https",
			uri:  "https://example.com/callback",
			want: false,
		},
		{
			name: "reject non-localhost http",
			uri:  "http://example.com/callback",
			want: false,
		},
		{
			name: "reject claude scheme",
			uri:  "claude://callback",
			want: false,
		},
		{
			name: "reject zed scheme",
			uri:  "zed://auth/callback",
			want: false,
		},
		{
			name: "reject jetbrains non-gateway path",
			uri:  "jetbrains://idea/oauth/callback",
			want: false,
		},
		{
			name: "reject malformed uri",
			uri:  "://bad",
			want: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.want, isAllowedDynamicClientRedirectURI(test.uri))
		})
	}
}
