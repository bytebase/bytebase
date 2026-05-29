package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAllowedDynamicClientRedirectURI(t *testing.T) {
	testCases := []struct {
		name string
		uri  string
		want bool
	}{
		// Localhost (loopback) callbacks — covers Claude Code, Gemini CLI, etc.
		{name: "localhost http", uri: "http://localhost:8080/callback", want: true},
		{name: "127.0.0.1 http", uri: "http://127.0.0.1:53682/callback", want: true},
		{name: "localhost https", uri: "https://localhost/callback", want: true},

		// Native app custom schemes already supported.
		{name: "cursor scheme", uri: "cursor://anysphere.cursor-mcp/oauth/callback", want: true},
		{name: "vscode scheme", uri: "vscode://anysphere.cursor/callback", want: true},
		{name: "vscode-insiders scheme", uri: "vscode-insiders://foo/callback", want: true},
		{name: "jetbrains gateway", uri: "jetbrains://gateway/auth/callback", want: true},
		{name: "jetbrains no path", uri: "jetbrains://gateway", want: false},

		// Hosted MCP client callbacks — Claude (web, Desktop, mobile, Cowork).
		{name: "claude.ai exact", uri: "https://claude.ai/api/mcp/auth_callback", want: true},
		{name: "claude.ai wrong path", uri: "https://claude.ai/evil", want: false},
		{name: "claude.ai over http", uri: "http://claude.ai/api/mcp/auth_callback", want: false},
		{name: "claude.ai subdomain spoof", uri: "https://claude.ai.evil.com/api/mcp/auth_callback", want: false},
		// Host should match case-insensitively and ignore the default port.
		{name: "claude.ai uppercase host", uri: "https://CLAUDE.AI/api/mcp/auth_callback", want: true},
		{name: "claude.ai explicit port", uri: "https://claude.ai:443/api/mcp/auth_callback", want: true},
		// Userinfo spoof must resolve to the real (non-allowlisted) host and be rejected.
		{name: "claude.ai userinfo spoof", uri: "https://claude.ai@evil.com/api/mcp/auth_callback", want: false},

		// ChatGPT connectors — current per-connector path + legacy fixed path.
		{name: "chatgpt connector prefix", uri: "https://chatgpt.com/connector/oauth/abc123", want: true},
		{name: "chatgpt legacy exact", uri: "https://chatgpt.com/connector_platform_oauth_redirect", want: true},
		{name: "chatgpt wrong path", uri: "https://chatgpt.com/evil", want: false},

		// VS Code for the Web.
		{name: "vscode.dev redirect", uri: "https://vscode.dev/redirect", want: true},
		{name: "insiders.vscode.dev redirect", uri: "https://insiders.vscode.dev/redirect", want: true},
		{name: "vscode.dev wrong path", uri: "https://vscode.dev/evil", want: false},

		// Antigravity.
		{name: "antigravity callback", uri: "https://antigravity.google/oauth-callback", want: true},
		{name: "antigravity wrong path", uri: "https://antigravity.google/evil", want: false},

		// Arbitrary HTTPS / HTTP must stay rejected (the #20033 protection).
		{name: "arbitrary https", uri: "https://evil.com/callback", want: false},
		{name: "non-localhost http", uri: "http://evil.com/callback", want: false},
		{name: "unknown scheme", uri: "ftp://example.com/callback", want: false},
		{name: "malformed", uri: "://nonsense", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isAllowedDynamicClientRedirectURI(tc.uri))
		})
	}
}
