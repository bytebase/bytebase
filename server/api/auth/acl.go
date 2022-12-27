package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// ACLInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) ACLInterceptor(ctx context.Context, req interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	role, err := in.getWorkspaceRole(ctx, serverInfo.FullMethod)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	// Stores principalID into context.
	childCtx := context.WithValue(ctx, common.RoleContextKey, role)

	// TODO(d): implement the authorization.

	return handler(childCtx, req)
}

func (in *APIAuthInterceptor) getWorkspaceRole(ctx context.Context, fullMethod string) (api.Role, error) {
	if isAuthenticationAllowed(fullMethod) {
		return api.Owner, nil
	}
	// If RBAC feature is not enabled, all users are treated as OWNER.
	if !in.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
		return api.Owner, nil
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	member, err := in.store.GetMemberByPrincipalID(ctx, principalID)
	if err != nil {
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "failed to get member for principal %v in processing authorize request.", principalID)
	}
	if member == nil {
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "member not found for principal %v in processing authorize request.", principalID)
	}
	if member.RowStatus == api.Archived {
		return api.UnknownRole, status.Errorf(codes.PermissionDenied, "The principal %v has been deactivated by the admin.", principalID)
	}
	return member.Role, nil
}
