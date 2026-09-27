package oauth2

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testTrustedBase = "https://bb.example.com"

// TestValidateResource is the rule space of the resource binding: which values
// name this server's MCP endpoint, and which are refused. The whole point of the
// check is that the answer comes from the *configured* external URL, so the
// unconfigured case is a first-class outcome here rather than an edge case.
func TestValidateResource(t *testing.T) {
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
		{"default port is stripped", "https://bb.example.com:443/mcp", "https://bb.example.com/mcp"},
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
		})
	}

	t.Run("every accepted spelling collapses to one stored value", func(t *testing.T) {
		stored := map[string]struct{}{}
		for _, in := range []string{
			"https://bb.example.com/mcp",
			"https://bb.example.com/mcp/",
			"HTTPS://BB.Example.COM/mcp",
			"https://bb.example.com:443/mcp",
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

func TestCanonicalizeScope(t *testing.T) {
	// The v1 bootstrap has the 401 challenge advertise every mode, so clients
	// request the whole set by design. A set is normal input: it resolves to its
	// highest rung, which is the one mode the grant is bound to.
	accepted := []struct {
		name string
		in   string
		want string
	}{
		{"empty is allowed", "", ""},
		{"whitespace only is empty", " \t\n ", ""},
		{"read-only alone", "mcp:read-only", "mcp:read-only"},
		{"read-write alone", "mcp:read-write", "mcp:read-write"},
		{"both modes, narrow first", "mcp:read-only mcp:read-write", "mcp:read-write"},
		{"both modes, wide first", "mcp:read-write mcp:read-only", "mcp:read-write"},
		{"repeated token", "mcp:read-only mcp:read-only", "mcp:read-only"},
		// Whatever whitespace a client uses, the stored value is one bare token —
		// the grant record must never hold a padded or multi-token string.
		{"padded and tab-separated", "  mcp:read-only\tmcp:read-write  ", "mcp:read-write"},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			got, err := canonicalizeScope(tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, strings.TrimSpace(got), got, "a stored scope must never carry whitespace")
			require.NotContains(t, got, " ", "a stored scope must name exactly one mode")
		})
	}

	// Unknown tokens are rejected rather than dropped, including when mixed in
	// with recognized ones — otherwise a client would believe the unrecognized
	// part of its request was granted.
	for _, scope := range []string{
		"mcp:admin",
		"openid",
		"MCP:READ-ONLY",
		"mcp:readonly",
		"mcp:read-only extra",
		"mcp:read-only mcp:admin",
		"mcp:admin mcp:read-write",
	} {
		t.Run("unknown scope rejected: "+scope, func(t *testing.T) {
			_, err := canonicalizeScope(scope)
			require.Error(t, err)
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
		{"default port is equivalent", url.Values{"resource": {"https://bb.example.com:443/mcp"}}, consented, ""},
		{"malformed value", url.Values{"resource": {"not a uri"}}, consented, "invalid_target"},
		{"repeated value", url.Values{"resource": {consented, consented}}, consented, "invalid_target"},
		// A grant with no stored resource predates the column (or the client never
		// sent one). RFC 8707 clients keep sending `resource` on every exchange
		// and refresh, so rejecting here would fail live sessions across upgrade.
		// Accepted and ignored — an unbound grant stays unbound, which the
		// lifecycle test asserts end to end.
		{"resource on a grant that consented none is accepted, not bound", url.Values{"resource": {consented}}, "", ""},
		{"any resource on an unbound grant is accepted", url.Values{"resource": {"https://evil.example.com/mcp"}}, "", ""},
		// Shape is still checked, so a genuinely broken value is still reported.
		{"malformed value on an unbound grant", url.Values{"resource": {"urn:nope"}}, "", "invalid_target"},
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
		// The client keeps sending the set it originally requested; it has to
		// normalize to the same mode consent stored, or every refresh would fail.
		{"the original requested set matches the maximum it resolved to", url.Values{"scope": {"mcp:read-only mcp:read-write"}}, "mcp:read-write", ""},
		{"a set whose maximum exceeds the grant is rejected", url.Values{"scope": {"mcp:read-only mcp:read-write"}}, "mcp:read-only", "invalid_scope"},
		{"a set containing an unknown token is rejected", url.Values{"scope": {"mcp:read-write mcp:admin"}}, "mcp:read-write", "invalid_scope"},
		// Same upgrade path as the resource: RFC 6749 §6 permits `scope` on a
		// refresh, so a grant issued before the column must not start failing.
		// Ignored, not adopted — the grant keeps its empty scope.
		{"scope on a grant that consented none is accepted, not adopted", url.Values{"scope": {"mcp:read-only"}}, "", ""},
		// A legacy client may be sending a scope we never recognized; before this
		// change it was ignored, and it must stay ignored rather than become a
		// rejection. This is why the empty-grant case returns before the
		// vocabulary check.
		{"an unrecognized scope on an unbound grant stays ignored", url.Values{"scope": {"openid profile"}}, "", ""},
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
