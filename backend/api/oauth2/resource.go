package oauth2

import (
	"context"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/utils"
)

// mcpResourcePath is the path of the MCP endpoint (registered as `/mcp` by the
// mcp package). Appended to the configured external URL it forms the canonical
// resource URI that RFC 8707 clients name in the `resource` parameter, and that
// our RFC 9728 metadata document publishes.
const mcpResourcePath = "/mcp"

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
	// resource is canonical or empty.
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

// parseGrantParams extracts and validates the `resource` (RFC 8707) and `scope`
// (RFC 6749) parameters of an authorization request. Both parameters are
// optional, and a value that is present is validated strictly; multiple
// occurrences of either are rejected.
//
// The resulting grant is always resource-bound (P1a PR 3): the stored resource
// becomes the access token's audience, so a client that omits `resource`
// (mandatory for MCP clients per RFC 8707, but not every DCR client is one) is
// bound to the canonical MCP resource — the only value validateResource would
// accept anyway. Leaving such a grant unbound instead would mint tokens no
// audience check accepts once the legacy migration window closes.
func (s *Service) parseGrantParams(ctx context.Context, values url.Values, workspaceID string) (grantParams, *oauth2Failure) {
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

	// Every consent needs the trusted external URL now that every grant is
	// bound. GetEffectiveExternalURL reports "not configured" as an error, never
	// as an empty string. Fail closed with the actionable setup error, never
	// fall back to the request.
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		slog.Error("rejected an MCP OAuth consent because no external URL is configured",
			slog.String("resource", resource))
		return grantParams{}, &oauth2Failure{code: "server_error", description: err.Error()}
	}

	if resource == "" {
		return grantParams{resource: externalURL + mcpResourcePath, scope: scope}, nil
	}
	canonical, err := validateResource(resource, externalURL)
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
	canonical, err := common.NormalizeExternalURL(requested)
	if err != nil {
		return &oauth2Failure{code: "invalid_target", description: err.Error()}
	}
	// Codes and refresh tokens issued before 3.22.1 read back with no resource.
	// The parameter check accepts anything for them so the refusal such a grant
	// gets is the deliberate one: the legacy gate right after these checks
	// (legacyGrantFailure) refuses the grant itself with re-auth guidance,
	// rather than a confusing mismatch error against a value that was never
	// consented. An unbound grant still cannot acquire a binding here.
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
// server and returns the form to consent to. externalURL comes from deployment
// config and is already normalized.
//
// Accepted inputs: the canonical MCP resource URI (<base>/mcp) and the bare
// origin (<base>) — the two values our own RFC 9728 metadata documents publish,
// so a client that read either one must not be turned away.
//
// Stored form: always <base>/mcp, whichever of the two was sent. PR 3 binds the
// access token's audience to this stored value, and two accepted spellings of one
// resource would mean two audiences to accept at /mcp — forever, since grants are
// long-lived. Normalizing at the door keeps the audience single-valued.
func validateResource(resource, externalURL string) (string, error) {
	mcpResource := externalURL + mcpResourcePath
	canonical, err := common.NormalizeExternalURL(resource)
	if err != nil {
		return "", err
	}
	if canonical != externalURL && canonical != mcpResource {
		return "", errors.Errorf("resource %q is not this server's MCP resource; expected %q", canonical, mcpResource)
	}
	return mcpResource, nil
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
