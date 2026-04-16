//nolint:revive
package common

import (
	"context"

	"google.golang.org/protobuf/types/known/anypb"
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

type AuthContext struct {
	Audit                  bool
	AllowWithoutCredential bool
	Permission             string
	AuthMethod             AuthMethod
	Resources              []*Resource
}

func GetAuthContextFromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx, ok
}
