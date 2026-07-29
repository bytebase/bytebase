package oauth2

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
)

const testTrustedBase = "https://bb.example.com"

// TestValidateResource is the rule space of the resource binding: which values
// name this server's MCP endpoint, and which are refused. The whole point of the
// check is that the answer comes from the *configured* external URL, so the
// unconfigured case is a first-class outcome here rather than an edge case.
func TestValidateResource(t *testing.T) {
	t.Run("no configured external URL is a configuration failure, not a client error", func(t *testing.T) {
		_, err := validateResource("https://bb.example.com/mcp", "")
		require.ErrorIs(t, err, errExternalURLNotConfigured,
			"callers key on this sentinel to return the actionable setup error instead of a generic OAuth failure")
	})

	// Every accepted input resolves to the same stored form. PR 3 binds the token
	// audience to what is stored, so a second accepted spelling would become a
	// second audience to honor at /mcp for the life of every grant.
	accepted := []struct {
		name string
		in   string
		want string
	}{
		{"canonical MCP resource URI", "https://bb.example.com/mcp", "https://bb.example.com/mcp"},
		{"trailing slash is stripped", "https://bb.example.com/mcp/", "https://bb.example.com/mcp"},
		{"scheme and host case are normalized", "HTTPS://BB.Example.COM/mcp", "https://bb.example.com/mcp"},
		// Our own RFC 9728 document at the unsuffixed well-known path publishes
		// the bare origin as `resource`, so a client that read it must not be
		// rejected for using the value we advertised — but it is stored as the
		// canonical MCP URI, not echoed back as the origin.
		{"bare origin normalizes to the MCP resource", "https://bb.example.com", "https://bb.example.com/mcp"},
		{"bare origin with trailing slash", "https://bb.example.com/", "https://bb.example.com/mcp"},
	}
	for _, tc := range accepted {
		t.Run("accepted: "+tc.name, func(t *testing.T) {
			got, err := validateResource(tc.in, testTrustedBase)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	rejected := []struct {
		name string
		in   string
	}{
		{"different host", "https://evil.example.com/mcp"},
		{"different scheme", "http://bb.example.com/mcp"},
		{"different path", "https://bb.example.com/v1/sql"},
		{"path case differs (paths are case-sensitive)", "https://bb.example.com/MCP"},
		// Not normalized on purpose: adding :443 to a URI we never published is
		// a noncanonical value, and quietly treating it as equal would widen
		// what counts as "this server".
		{"explicit default port", "https://bb.example.com:443/mcp"},
		{"nondefault port", "https://bb.example.com:8443/mcp"},
		{"percent-encoded path separator", "https://bb.example.com%2Fmcp"},
		{"query string", "https://bb.example.com/mcp?x=1"},
		{"fragment", "https://bb.example.com/mcp#frag"},
		{"userinfo", "https://user@bb.example.com/mcp"},
		{"relative reference", "/mcp"},
		{"non-http scheme", "urn:example:mcp"},
		{"no host", "https:///mcp"},
		{"empty is not valid once present", " "},
	}
	for _, tc := range rejected {
		t.Run("rejected: "+tc.name, func(t *testing.T) {
			_, err := validateResource(tc.in, testTrustedBase)
			require.Error(t, err)
			require.NotErrorIs(t, err, errExternalURLNotConfigured,
				"a client-supplied bad value must not be reported as a server misconfiguration")
		})
	}

	t.Run("a trailing slash on the configured URL does not change what is accepted", func(t *testing.T) {
		got, err := validateResource("https://bb.example.com/mcp", "https://bb.example.com/")
		require.NoError(t, err)
		require.Equal(t, "https://bb.example.com/mcp", got)
	})

	t.Run("a mixed-case configured URL still matches a canonical resource", func(t *testing.T) {
		got, err := validateResource("https://bb.example.com/mcp", "https://BB.Example.com")
		require.NoError(t, err)
		require.Equal(t, "https://bb.example.com/mcp", got)
	})

	t.Run("every accepted spelling collapses to one stored value", func(t *testing.T) {
		stored := map[string]struct{}{}
		for _, in := range []string{
			"https://bb.example.com/mcp",
			"https://bb.example.com/mcp/",
			"HTTPS://BB.Example.COM/mcp",
			"https://bb.example.com",
			"https://bb.example.com/",
		} {
			got, err := validateResource(in, testTrustedBase)
			require.NoError(t, err)
			stored[got] = struct{}{}
		}
		require.Len(t, stored, 1, "more than one stored spelling would force PR 3 to accept multiple audiences")
	})
}

func TestValidateScope(t *testing.T) {
	t.Run("empty scope is allowed", func(t *testing.T) {
		// The consent-time picker is P1b; until it exists every real client
		// sends no scope at all, and rejecting that would break MCP entirely.
		require.NoError(t, validateScope(""))
	})
	for _, scope := range knownScopes {
		t.Run("known scope "+scope, func(t *testing.T) {
			require.NoError(t, validateScope(scope))
		})
	}
	for _, scope := range []string{"mcp:admin", "openid", "mcp:read-only extra", "MCP:READ-ONLY", "mcp:readonly"} {
		t.Run("unknown scope "+scope, func(t *testing.T) {
			require.Error(t, validateScope(scope),
				"an unrecognized scope must be refused, not silently dropped — a client would otherwise believe it holds something we never granted")
		})
	}
}

func TestSingleValue(t *testing.T) {
	t.Run("absent is empty", func(t *testing.T) {
		got, err := singleValue(url.Values{}, "resource")
		require.NoError(t, err)
		require.Empty(t, got)
	})
	t.Run("single value is returned", func(t *testing.T) {
		got, err := singleValue(url.Values{"resource": {"https://bb.example.com/mcp"}}, "resource")
		require.NoError(t, err)
		require.Equal(t, "https://bb.example.com/mcp", got)
	})
	t.Run("repeated value is rejected", func(t *testing.T) {
		_, err := singleValue(url.Values{"resource": {"https://bb.example.com/mcp", "https://evil.example.com/mcp"}}, "resource")
		require.Error(t, err,
			"picking one of two values would bind the token to a resource the caller may not have meant")
	})
}

// TestCheckConsentedResource covers the token endpoint's half: the request may
// omit the resource, but it may never name a different one than was consented.
func TestCheckConsentedResource(t *testing.T) {
	const consented = "https://bb.example.com/mcp"

	cases := []struct {
		name      string
		values    url.Values
		consented string
		wantCode  string
	}{
		{"omitted keeps the consented resource", url.Values{}, consented, ""},
		{"exact match", url.Values{"resource": {consented}}, consented, ""},
		{"noncanonical but equivalent match", url.Values{"resource": {"HTTPS://BB.example.com/mcp/"}}, consented, ""},
		// Same normalization at both ends: consent stores <base>/mcp for a bare
		// origin, so the token endpoint must accept a bare origin against it.
		{"bare origin matches a /mcp-stored grant", url.Values{"resource": {"https://bb.example.com"}}, consented, ""},
		{"bare origin with trailing slash matches", url.Values{"resource": {"https://bb.example.com/"}}, consented, ""},
		{"bare origin of a different host still fails", url.Values{"resource": {"https://evil.example.com"}}, consented, "invalid_target"},
		{"different resource", url.Values{"resource": {"https://evil.example.com/mcp"}}, consented, "invalid_target"},
		{"noncanonical value that is not equivalent", url.Values{"resource": {"https://bb.example.com:443/mcp"}}, consented, "invalid_target"},
		{"malformed value", url.Values{"resource": {"not a uri"}}, consented, "invalid_target"},
		{"repeated value", url.Values{"resource": {consented, consented}}, consented, "invalid_target"},
		// A grant consented without a resource cannot acquire one at the token
		// endpoint — that would be a resource binding the user never approved.
		{"resource on a grant that consented none", url.Values{"resource": {consented}}, "", "invalid_target"},
		{"omitted on a grant that consented none", url.Values{}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := checkConsentedResource(tc.values, tc.consented)
			if tc.wantCode == "" {
				require.Nil(t, failure)
				return
			}
			require.NotNil(t, failure)
			require.Equal(t, tc.wantCode, failure.code)
		})
	}
}

// TestCheckConsentedScope pins the never-widened rule: a refresh re-issues the
// consented scope, and asking for anything else fails instead of being honored.
func TestCheckConsentedScope(t *testing.T) {
	cases := []struct {
		name      string
		values    url.Values
		consented string
		wantCode  string
	}{
		{"omitted keeps the consented scope", url.Values{}, "mcp:read-only", ""},
		{"exact match", url.Values{"scope": {"mcp:read-only"}}, "mcp:read-only", ""},
		{"widening is rejected", url.Values{"scope": {"mcp:read-write"}}, "mcp:read-only", "invalid_scope"},
		{"narrowing is rejected too (the grant record is the authority)", url.Values{"scope": {"mcp:read-only"}}, "mcp:read-write", "invalid_scope"},
		{"scope on a grant that consented none", url.Values{"scope": {"mcp:read-only"}}, "", "invalid_scope"},
		{"repeated value", url.Values{"scope": {"mcp:read-only", "mcp:read-write"}}, "mcp:read-only", "invalid_scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := checkConsentedScope(tc.values, tc.consented)
			if tc.wantCode == "" {
				require.Nil(t, failure)
				return
			}
			require.NotNil(t, failure)
			require.Equal(t, tc.wantCode, failure.code)
		})
	}
}

// TestTrustedExternalURLUsesTheFlagOnly is the negative control for the whole
// PR: getBaseURL falls back to the request Host, and that fallback must never
// reach resource validation, because a client controls it and could then name
// the audience of its own token. trustedExternalURL takes no request at all —
// there is nothing to fall back to.
func TestTrustedExternalURLUsesTheFlagOnly(t *testing.T) {
	s := &Service{profile: &config.Profile{ExternalURL: "https://bb.example.com/"}}
	require.Equal(t, "https://bb.example.com", s.trustedExternalURL(t.Context()),
		"the flag is the trusted tier and its trailing slash is normalized away")
}
