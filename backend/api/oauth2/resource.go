package oauth2

import (
	"context"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/utils"
)

// mcpResourcePath is the path of the MCP endpoint (registered as `/mcp` by the
// mcp package). Appended to the configured external URL it forms the canonical
// resource URI that RFC 8707 clients name in the `resource` parameter, and that
// our RFC 9728 metadata document publishes.
const mcpResourcePath = "/mcp"

// externalURLSetupError is returned when a client asks to bind a token to a
// resource but the deployment has no configured external URL. It names the two
// places an admin can set one, because the fix is entirely on the server side
// and the client operator can do nothing about it.
const externalURLSetupError = "MCP OAuth requires a configured external URL. Start the server with --external-url https://<your-bytebase-host>, or set Settings > General > External URL " +
	"(https://docs.bytebase.com/get-started/self-host/external-url), then reconnect. The request Host header is deliberately not trusted for resource binding."

// errExternalURLNotConfigured is returned by trustedExternalURL when no trusted
// external URL is available, so resource validation is never handed an empty base
// to interpret. Mapped to the actionable configuration error above at the call
// site; every other validation failure is a client error.
var errExternalURLNotConfigured = errors.New("no configured external URL to validate the resource against")

// scopeLadder is the P1a scope vocabulary — one scope per predefined permission
// set — ordered narrowest to widest. Vocabulary per proposal v2 §4; the
// Metadata-only tier is deferred, so there are two rungs.
//
// The scopes are tiers rather than additive permissions, which is why a request
// naming several of them reduces to the highest one instead of their union. P1a
// persists and echoes the selected mode; enforcement is P1b, and the consent-time
// picker that lets a user choose below the requested maximum lands with it.
var scopeLadder = []string{"mcp:read-only", "mcp:read-write"}

// grantParams is the validated resource/scope pair consented for one grant.
type grantParams struct {
	// resource is canonical (see canonicalizeResourceURI) or empty.
	resource string
	// scope is the single mode the requested scope set resolved to, or empty.
	scope string
}

// oauth2Failure carries an RFC 6749 error code plus a description so a helper
// can report why it refused without deciding how to render it — /authorize
// redirects its errors, /token returns a JSON body.
type oauth2Failure struct {
	code        string
	description string
}

// trustedExternalURL returns the canonical external URL configured for this
// deployment: the --external-url flag, else the workspace setting. It returns
// errExternalURLNotConfigured when neither is available, so callers never have to
// interpret an empty string.
//
// Unlike getBaseURL this deliberately has no request-derived tier. Host and
// X-Forwarded-Proto are client-controlled, so a header-derived value would let a
// caller choose the audience its own token is bound to — which is the entire
// property the resource binding exists to establish.
func (s *Service) trustedExternalURL(ctx context.Context) (string, error) {
	if s.profile.ExternalURL != "" {
		return strings.TrimSuffix(s.profile.ExternalURL, "/"), nil
	}
	// The workspace ID is only resolved on self-hosted, and that is a correctness
	// requirement rather than an assumption that SaaS passes the flag:
	//
	//   - GetWorkspaceID is a singleton lookup (`... WHERE deleted = FALSE LIMIT 1`),
	//     so on SaaS it returns an arbitrary workspace. Feeding that to
	//     GetEffectiveExternalURL would validate this grant's resource against an
	//     unrelated tenant's setting.
	//   - On SaaS the setting cannot hold a value anyway: writes to
	//     workspace_profile.external_url are rejected outright in SaaS mode
	//     (setting_service.go), so the flag is the only source that can exist.
	//
	// Same tier order as getBaseURL and mcp's buildResourceMetadataURL, minus
	// their request-derived tail. Unifying the three is BOT-32.
	workspaceID := ""
	if !s.profile.SaaS {
		if ws, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = ws
		}
	}
	// GetEffectiveExternalURL reports "not configured" as an error, never as an
	// empty string. A lookup failure is treated identically — fail closed, never
	// fall back to the request — with the cause logged rather than returned.
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		slog.Warn("failed to resolve trusted external URL for OAuth2 resource binding", log.BBError(err))
		return "", errExternalURLNotConfigured
	}
	return strings.TrimSuffix(externalURL, "/"), nil
}

// parseGrantParams extracts and validates the `resource` (RFC 8707) and `scope`
// (RFC 6749) parameters of an authorization request. Both are optional — clients
// that send neither still consent successfully — but a value that is present is
// validated strictly, and multiple occurrences of either are rejected.
func (s *Service) parseGrantParams(ctx context.Context, values url.Values) (grantParams, *oauth2Failure) {
	resource, err := singleValue(values, "resource")
	if err != nil {
		return grantParams{}, &oauth2Failure{code: "invalid_target", description: err.Error()}
	}
	requestedScope, err := singleValue(values, "scope")
	if err != nil {
		return grantParams{}, &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	scope, err := canonicalizeScope(requestedScope)
	if err != nil {
		return grantParams{}, &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	if resource == "" {
		return grantParams{scope: scope}, nil
	}

	trustedBaseURL, err := s.trustedExternalURL(ctx)
	if err != nil {
		slog.Error("rejected an MCP OAuth resource binding because no external URL is configured",
			slog.String("resource", resource))
		return grantParams{}, &oauth2Failure{code: "server_error", description: externalURLSetupError}
	}
	canonical, err := validateResource(resource, trustedBaseURL)
	if err != nil {
		return grantParams{}, &oauth2Failure{code: "invalid_target", description: err.Error()}
	}
	return grantParams{resource: canonical, scope: scope}, nil
}

// checkConsentedResource verifies a token request's `resource` against the value
// the grant was consented for. Omitting it is allowed (the consented value
// stands); naming a different one is not, because the token would then be bound
// somewhere the user never approved.
//
// The bare origin is accepted here too, matching what validateResource accepts at
// consent time. The equivalence is derived from the stored value rather than from
// the configured external URL on purpose: comparing against live config would make
// every outstanding grant unusable the moment an admin changes the external URL.
//
// A grant with no stored resource has no constraint to check, so the parameter is
// accepted and ignored rather than rejected — see the comment on the empty case
// below.
func checkConsentedResource(values url.Values, consented string) *oauth2Failure {
	requested, err := singleValue(values, "resource")
	if err != nil {
		return &oauth2Failure{code: "invalid_target", description: err.Error()}
	}
	if requested == "" {
		return nil
	}
	canonical, err := canonicalizeResourceURI(requested)
	if err != nil {
		return &oauth2Failure{code: "invalid_target", description: err.Error()}
	}
	// Codes and refresh tokens issued before 3.22.1 read back with no resource,
	// and RFC 8707 clients send `resource` on every exchange *and* every refresh.
	// Rejecting it would fail each in-flight code and every live session at its
	// next refresh, for up to a refresh-token lifetime after upgrade. Accept it
	// without binding: the stored value stays empty and is what gets carried
	// forward, so an unbound grant cannot acquire a binding here. Retiring that
	// population is PR 3's job (spec §"PR 3": legacy refresh grants require
	// re-consent), and it needs the audience change to do it coherently.
	if consented == "" {
		return nil
	}
	if canonical != consented && canonical+mcpResourcePath != consented {
		return &oauth2Failure{
			code:        "invalid_target",
			description: "resource does not match the resource this grant was authorized for",
		}
	}
	return nil
}

// checkConsentedScope verifies a token request's `scope` against the consented
// scope. Omitting it is allowed (the consented scope stands); anything else must
// match exactly. Narrowing is not supported here either: the grant record is the
// authority, and changing what a session may do happens on the grant page, never
// by asking for a different scope at the token endpoint.
//
// The requested value is normalized the same way consent normalized it, so a
// client that keeps sending its full requested set matches the single mode that
// set resolved to.
//
// A grant with no stored scope is treated the same way as one with no stored
// resource, and for the same upgrade reason: RFC 6749 §6 lets a client send
// `scope` on a refresh, so a pre-3.22.1 grant would start failing its refreshes.
// That case returns before normalization on purpose — a legacy client may be
// sending a custom scope that was simply ignored before this change, and running
// it through the vocabulary check would turn "ignored" into "rejected".
func checkConsentedScope(values url.Values, consented string) *oauth2Failure {
	requested, err := singleValue(values, "scope")
	if err != nil {
		return &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	if requested == "" || consented == "" {
		return nil
	}
	canonical, err := canonicalizeScope(requested)
	if err != nil {
		return &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	if canonical != consented {
		return &oauth2Failure{
			code:        "invalid_scope",
			description: "scope cannot be changed when exchanging or refreshing a token; reconnect to authorize a different scope",
		}
	}
	return nil
}

// singleValue returns the sole value of name, "" when absent, and an error when
// it appears more than once. A repeated parameter is ambiguous — one value would
// end up bound to the token and the other silently dropped — so RFC 6749 §3.1
// forbids it and we reject rather than pick.
func singleValue(values url.Values, name string) (string, error) {
	got := values[name]
	if len(got) > 1 {
		return "", errors.Errorf("%s must appear at most once, got %d values", name, len(got))
	}
	if len(got) == 0 {
		return "", nil
	}
	return got[0], nil
}

// validateResource checks that a client-supplied resource indicator names this
// server and returns the form to consent to. trustedBaseURL comes from
// trustedExternalURL, which reports an unconfigured deployment as an error, so it
// is always a real URL here.
//
// Accepted inputs: the canonical MCP resource URI (<base>/mcp) and the bare
// origin (<base>) — the two values our own RFC 9728 metadata documents publish,
// so a client that read either one must not be turned away.
//
// Stored form: always <base>/mcp, whichever of the two was sent. PR 3 binds the
// access token's audience to this stored value, and two accepted spellings of one
// resource would mean two audiences to accept at /mcp — forever, since grants are
// long-lived. Normalizing at the door keeps the audience single-valued.
func validateResource(resource, trustedBaseURL string) (string, error) {
	base, err := canonicalizeResourceURI(trustedBaseURL)
	if err != nil {
		return "", errors.Wrapf(err, "configured external URL %q is not a usable absolute URL", trustedBaseURL)
	}
	mcpResource := base + mcpResourcePath
	canonical, err := canonicalizeResourceURI(resource)
	if err != nil {
		return "", err
	}
	if canonical != base && canonical != mcpResource {
		return "", errors.Errorf("resource %q is not this server's MCP resource; expected %q", canonical, mcpResource)
	}
	return mcpResource, nil
}

// canonicalizeResourceURI normalizes the two differences a resource URI may
// carry without changing which resource it names: scheme and host case, and a
// trailing slash. Everything else is left exactly as sent — a port, a
// percent-encoded path, a different path — so a noncanonical value fails the
// comparison in validateResource instead of being quietly accepted as equal.
//
// Query strings, fragments, and userinfo are rejected outright: RFC 8707 §2
// resource URIs must not carry a fragment, and none of the three ever appear in
// a legitimate MCP resource identifier.
func canonicalizeResourceURI(resource string) (string, error) {
	u, err := url.Parse(resource)
	if err != nil {
		return "", errors.Wrap(err, "resource is not a valid URI")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.Errorf("resource must be an absolute http(s) URI, got %q", resource)
	}
	if u.Host == "" {
		return "", errors.Errorf("resource must name a host, got %q", resource)
	}
	if u.User != nil {
		return "", errors.New("resource must not carry userinfo")
	}
	if u.RawQuery != "" || u.ForceQuery {
		return "", errors.New("resource must not carry a query string")
	}
	if u.Fragment != "" || u.RawFragment != "" {
		return "", errors.New("resource must not carry a fragment")
	}
	return scheme + "://" + strings.ToLower(u.Host) + strings.TrimSuffix(u.EscapedPath(), "/"), nil
}

// canonicalizeScope reduces a requested scope set to the one mode a grant can be
// bound to: the requested maximum, the highest rung named. Returns "" for an
// empty or whitespace-only request.
//
// A set is normal input, not a malformed request. The v1 bootstrap has the 401
// challenge advertise every mode — it is pre-authentication, so it cannot be
// capped by a workspace ceiling — and clients therefore ask for everything; the
// consent picker then selects a mode at or below that maximum (research doc §5.4,
// "Scope normalization"). Two values are tracked separately: the client's
// requested maximum, which this produces, and the durable consented grant, which
// is always exactly one mode.
//
// Unknown tokens are rejected rather than dropped, so a client never believes it
// holds something we did not grant.
func canonicalizeScope(scope string) (string, error) {
	canonical, highest := "", -1
	for token := range strings.FieldsSeq(scope) {
		rung := slices.Index(scopeLadder, token)
		if rung < 0 {
			return "", errors.Errorf("unknown scope %q; supported scopes are %s", token, strings.Join(scopeLadder, " "))
		}
		if rung > highest {
			canonical, highest = token, rung
		}
	}
	return canonical, nil
}
