package v1

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/anypb"
)

type auditContextKey int

const (
	serviceDataKey auditContextKey = iota
	auditWorkspaceIDKey
	permissionDeniedKey
	handlerReachedKey
)

func withSetServiceData(ctx context.Context, setServiceData func(a *anypb.Any)) context.Context {
	return context.WithValue(ctx, serviceDataKey, setServiceData)
}

func getSetServiceDataFromContext(ctx context.Context) (func(a *anypb.Any), bool) {
	setServiceData, ok := ctx.Value(serviceDataKey).(func(*anypb.Any))
	return setServiceData, ok
}

// withSetAuditWorkspaceID registers a callback handlers can use to tell the
// audit interceptor which workspace a request should be audited against. This
// is needed for methods that run with allow_without_credential=true (e.g.
// Login/Signup/ExchangeToken): the workspace is unknown when the interceptor
// chain starts, but the handler learns it before returning.
func withSetAuditWorkspaceID(ctx context.Context, setAuditWorkspaceID func(workspaceID string)) context.Context {
	return context.WithValue(ctx, auditWorkspaceIDKey, setAuditWorkspaceID)
}

// setAuditWorkspaceID records the workspace that the current request should be
// audited against, if the audit interceptor registered a setter on the context.
// Safe to call even when auditing is disabled for the current method.
func setAuditWorkspaceID(ctx context.Context, workspaceID string) {
	if workspaceID == "" {
		return
	}
	setter, ok := ctx.Value(auditWorkspaceIDKey).(func(string))
	if !ok {
		return
	}
	setter(workspaceID)
}

// withSetPermissionDenied registers a callback that a permission check inside
// the audit interceptor uses to report that it refused the request. The check
// runs inside the interceptor, so a value it puts on the context cannot travel
// back out; withSetAuditWorkspaceID has the same shape for the same reason.
func withSetPermissionDenied(ctx context.Context, setPermissionDenied func()) context.Context {
	return context.WithValue(ctx, permissionDeniedKey, setPermissionDenied)
}

// setPermissionDenied reports that a permission check refused the current
// request. Mark only a verdict about the caller's permission, not an outage, a
// license gate, a workflow state or an ownership rule.
func setPermissionDenied(ctx context.Context) {
	if setter, ok := ctx.Value(permissionDeniedKey).(func()); ok {
		setter()
	}
}

// permissionDeniedError is the refusal a handler answers with when its own
// permission check turns the caller away. It marks the request, so the audit
// interceptor streams the refusal and stamps it WARNING. A license gate, a
// workflow state or an ownership rule answers connect.NewError directly: none
// of them is a verdict about a permission the caller holds.
func permissionDeniedError(ctx context.Context, err error) *connect.Error {
	setPermissionDenied(ctx)
	return connect.NewError(connect.CodePermissionDenied, err)
}

// withSetHandlerReached registers a callback the ACL interceptor uses to report
// that it admitted the request to its handler.
func withSetHandlerReached(ctx context.Context, setHandlerReached func()) context.Context {
	return context.WithValue(ctx, handlerReachedKey, setHandlerReached)
}

// setHandlerReached reports that the current request passed every interceptor
// check and is about to run its handler. The audit interceptor stores a row
// only for such a call.
func setHandlerReached(ctx context.Context) {
	if setter, ok := ctx.Value(handlerReachedKey).(func()); ok {
		setter()
	}
}
