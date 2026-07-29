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

// errExternalURLNotConfigured signals that resource validation could not run
// because no trusted external URL is configured. Mapped to the actionable
// configuration error at the call site; every other validation failure is a
// client error.
var errExternalURLNotConfigured = errors.New("no configured external URL to validate the resource against")

// knownScopes is the P1a scope vocabulary: one scope per predefined permission
// set. P1a persists and echoes the consented scope; enforcement is P1b, and the
// consent-time picker that lets a user choose between them lands with it. Until
// then an unknown scope is rejected rather than silently dropped, so a client
// never believes it holds something we did not grant.
var knownScopes = []string{"mcp:read-only", "mcp:read-write"}

// grantParams is the validated resource/scope pair consented for one grant.
type grantParams struct {
	// resource is canonical (see canonicalizeResourceURI) or empty.
	resource string
	// scope is the client's scope string, stored verbatim, or empty.
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
// "" when neither is configured.
//
// Unlike getBaseURL this deliberately has no request-derived tier. Host and
// X-Forwarded-Proto are client-controlled, so a header-derived value would let a
// caller choose the audience its own token is bound to — which is the entire
// property the resource binding exists to establish.
func (s *Service) trustedExternalURL(ctx context.Context) string {
	if s.profile.ExternalURL != "" {
		return strings.TrimSuffix(s.profile.ExternalURL, "/")
	}
	// On self-hosted, resolve the singleton workspace so the DB-backed
	// workspace_profile.external_url setting can be found. On SaaS there is no
	// singleton — the CLI flag is required.
	workspaceID := ""
	if !s.profile.SaaS {
		if ws, err := s.store.GetWorkspaceID(ctx); err == nil {
			workspaceID = ws
		}
	}
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		// Not configured is the common case here and is reported to the caller
		// as the actionable setup error; a lookup failure is logged and treated
		// the same way (fail closed, never fall back to the request).
		slog.Warn("failed to resolve trusted external URL for OAuth2 resource binding", log.BBError(err))
		return ""
	}
	return strings.TrimSuffix(externalURL, "/")
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
	scope, err := singleValue(values, "scope")
	if err != nil {
		return grantParams{}, &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	if err := validateScope(scope); err != nil {
		return grantParams{}, &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	if resource == "" {
		return grantParams{scope: scope}, nil
	}

	canonical, err := validateResource(resource, s.trustedExternalURL(ctx))
	if errors.Is(err, errExternalURLNotConfigured) {
		slog.Error("rejected an MCP OAuth resource binding because no external URL is configured",
			slog.String("resource", resource))
		return grantParams{}, &oauth2Failure{code: "server_error", description: externalURLSetupError}
	}
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
	// Codes and refresh tokens issued before 3.21.5 read back with no resource,
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
// A grant with no stored scope is treated the same way as one with no stored
// resource, and for the same upgrade reason: RFC 6749 §6 lets a client send
// `scope` on a refresh, so a pre-3.21.5 grant would start failing its refreshes.
// The requested value is ignored rather than adopted, so the grant keeps the
// (empty) scope it was issued with.
func checkConsentedScope(values url.Values, consented string) *oauth2Failure {
	requested, err := singleValue(values, "scope")
	if err != nil {
		return &oauth2Failure{code: "invalid_scope", description: err.Error()}
	}
	if requested == "" || requested == consented || consented == "" {
		return nil
	}
	return &oauth2Failure{
		code:        "invalid_scope",
		description: "scope cannot be changed when exchanging or refreshing a token; reconnect to authorize a different scope",
	}
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
// server and returns the form to consent to. trustedBaseURL must come from the
// configured external URL; an empty one means we cannot prove anything about the
// value and yields errExternalURLNotConfigured.
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
	if trustedBaseURL == "" {
		return "", errExternalURLNotConfigured
	}
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

// validateScope accepts an empty scope (the common case today — no client asks
// for one) and otherwise requires exactly one token from the known vocabulary.
//
// Exactly one, not a set of them: a grant is bound to a single predefined
// permission set (proposal v2 §4 — "consent binds the grant to a predefined
// set", stored as one set reference). The scopes are a ladder, not additive
// permissions, so `mcp:read-only mcp:read-write` names no set that can be
// resolved. Accepting it would persist and echo a scope string with no
// representation in the model P1b enforces against.
func validateScope(scope string) error {
	tokens := strings.Fields(scope)
	if len(tokens) == 0 {
		return nil
	}
	if len(tokens) > 1 {
		return errors.Errorf("scope must name exactly one permission set, got %d tokens; supported scopes are %s", len(tokens), strings.Join(knownScopes, " "))
	}
	if !slices.Contains(knownScopes, tokens[0]) {
		return errors.Errorf("unknown scope %q; supported scopes are %s", tokens[0], strings.Join(knownScopes, " "))
	}
	return nil
}
