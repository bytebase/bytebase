//nolint:revive
package common

import (
	"context"

	"google.golang.org/protobuf/types/known/anypb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ContextKey is the key type of context value.
type ContextKey int

const (
	// UserContextKey is the key name used to store user message in the context.
	UserContextKey ContextKey = iota
	AuthContextKey
	ServiceDataKey
	WorkspaceIDContextKey
	AuditWorkspaceIDKey
	MCPPolicyDenialKey
)

func WithSetServiceData(ctx context.Context, setServiceData func(a *anypb.Any)) context.Context {
	return context.WithValue(ctx, ServiceDataKey, setServiceData)
}

func GetSetServiceDataFromContext(ctx context.Context) (func(a *anypb.Any), bool) {
	setServiceData, ok := ctx.Value(ServiceDataKey).(func(*anypb.Any))
	return setServiceData, ok
}

// WithSetAuditWorkspaceID registers a callback handlers can use to tell the
// audit interceptor which workspace a request should be audited against. This
// is needed for methods that run with allow_without_credential=true (e.g.
// Login/Signup/ExchangeToken): the workspace is unknown when the interceptor
// chain starts, but the handler learns it before returning.
func WithSetAuditWorkspaceID(ctx context.Context, setAuditWorkspaceID func(workspaceID string)) context.Context {
	return context.WithValue(ctx, AuditWorkspaceIDKey, setAuditWorkspaceID)
}

// SetAuditWorkspaceID records the workspace that the current request should be
// audited against, if the audit interceptor registered a setter on the context.
// Safe to call even when auditing is disabled for the current method.
func SetAuditWorkspaceID(ctx context.Context, workspaceID string) {
	if workspaceID == "" {
		return
	}
	setter, ok := ctx.Value(AuditWorkspaceIDKey).(func(string))
	if !ok {
		return
	}
	setter(workspaceID)
}

// WithSetMCPPolicyDenied registers a callback the MCP ceiling gate uses to tell
// the audit interceptor that it refused this request. The gate runs inside the
// audit interceptor, so a value it puts on the context cannot travel back out;
// this is the same setter shape WithSetAuditWorkspaceID already uses for the
// same reason.
//
// The signal is needed because the audit interceptor otherwise writes a row
// only when the method's own audit annotation asks for one, and 47 of the
// methods the gate refuses carry no such annotation. A denial nobody can see is
// the outcome an operator most needs to see.
func WithSetMCPPolicyDenied(ctx context.Context, setMCPPolicyDenied func()) context.Context {
	return context.WithValue(ctx, MCPPolicyDenialKey, setMCPPolicyDenied)
}

// SetMCPPolicyDenied records that the MCP ceiling gate refused the current
// request, if the audit interceptor registered a setter on the context. Safe to
// call when it did not: the public chain never runs the gate at all.
func SetMCPPolicyDenied(ctx context.Context) {
	setter, ok := ctx.Value(MCPPolicyDenialKey).(func())
	if !ok {
		return
	}
	setter()
}

// GetWorkspaceIDFromContext returns the workspace ID from the request context.
func GetWorkspaceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(WorkspaceIDContextKey).(string); ok {
		return v
	}
	return ""
}

type AuthMethod int

const (
	AuthMethodUnspecified AuthMethod = iota
	AuthMethodIAM
	AuthMethodCustom
)

// ResourceType indicates whether a resource is workspace-scoped or project-scoped.
type ResourceType int

const (
	ResourceTypeWorkspace ResourceType = iota
	ResourceTypeProject
)

// Resource represents a resource extracted from an API request for authorization and audit.
type Resource struct {
	Type ResourceType
	// ID is the resource identifier:
	// - For workspace: the workspace resource ID
	// - For project: the project resource ID
	ID string
}

// DelegatedGrant is the OAuth2 grant state of the delegated MCP credential a
// request arrived with on the internal MCP transport. It is grant identity,
// not authorization state: roles and membership are re-resolved live per
// request, and nothing enforces on these values yet — P1b's capability gate is
// the consumer this shape is the contract for.
//
// Scope and Resource are the grant's STORED values, copied verbatim from the
// verified credential, and their empty states are load-bearing:
//
//   - Scope and Resource both empty: a genuinely pre-grant legacy session — a
//     plain web-session token at /mcp, or an OAuth2 token from before grants
//     recorded scope and resource.
//   - Scope empty, Resource present: a grant that recorded no scope. Two
//     indistinguishable origins: a client that omitted `scope` at consent — a
//     permanent, steady-state population — or, transiently, a token minted by
//     a PR-3-era replica during a rolling upgrade, whose grant DID record a
//     consented scope that the claim predates. Neither may be collapsed into
//     the pre-grant case: that could widen a consented read-only session to
//     full legacy semantics. P1b resolves this state most-restrictively.
//
// No store lookup recovers a missing scope, deliberately: the steady-state
// population has nothing to recover (the grant stored none), the transient
// one self-drains within an access-token lifetime and its refresh-token row
// is keyed by a (client_id, token_hash) the credential does not carry, and
// nothing enforces on the value yet.
type DelegatedGrant struct {
	// Scope is the consented permission-set name (e.g. "mcp:read-only");
	// empty in the two states above.
	Scope string
	// Resource is the grant's stored MCP resource URI.
	Resource string
	// ClientID is the OAuth2 client the grant was consented to.
	ClientID string
	// CorrelationID identifies the delegation this request rode in on. It is
	// minted at the /mcp boundary and session-scoped in practice: tool
	// handlers run on their session's initialize-time identity.
	CorrelationID string
}

type AuthContext struct {
	Audit                  bool
	AllowWithoutCredential bool
	Permission             string
	AuthMethod             AuthMethod
	// MCPMethodClass is the method's bytebase.v1.mcp_method_class annotation.
	// The effective authorization of an MCP session is this classification
	// intersected with the caller's own RBAC — it only ever narrows. The MCP
	// gate enforces every value: READ and WRITE are the serving classes the
	// workspace ceiling selects between, EXCLUDED and FORBIDDEN are served by
	// no ceiling, and UNSPECIFIED means "not yet classified" rather than
	// "safe", so it is refused too.
	MCPMethodClass v1pb.MCPMethodClass
	// MCPDenialReason is the method's bytebase.v1.mcp_denial_reason
	// annotation: which mechanism or scope decision refuses it, so the denial
	// can say why. Meaningful only when MCPMethodClass is FORBIDDEN or
	// EXCLUDED, and UNSPECIFIED costs wording rather than enforcement — the
	// class is what denies.
	MCPDenialReason v1pb.MCPDenialReason
	Resources       []*Resource
	// DelegatedGrant carries the grant state of the delegated MCP credential
	// on internal-chain requests; nil on the public chain. Presence, not any
	// field value, marks a request as MCP-originated.
	DelegatedGrant *DelegatedGrant
}

func GetAuthContextFromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx, ok
}
