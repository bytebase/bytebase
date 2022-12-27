package auth

import (
	"context"

	"google.golang.org/grpc"

	"github.com/bytebase/bytebase/api"
)

// ACLInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) ACLInterceptor(ctx context.Context, req interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if isAuthenticationAllowed(serverInfo.FullMethod) {
		return handler(ctx, req)
	}
	// If RBAC feature is not enabled, all users are treated as OWNER.
	if !in.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
		return handler(ctx, req)
	}

	return handler(ctx, req)
}
