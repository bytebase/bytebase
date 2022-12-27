package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"

	"github.com/bytebase/bytebase/api"
)

// ACLInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) ACLInterceptor(ctx context.Context, req interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	for _, allow := range authenticationAllowlistMethods {
		if strings.HasPrefix(serverInfo.FullMethod, fmt.Sprintf("%s%s", apiPackagePrefix, allow)) {
			return handler(ctx, req)
		}
	}

	// If RBAC feature is not enabled, all users are treated as OWNER.
	if !in.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
		return handler(ctx, req)
	}

	return handler(ctx, req)
}
