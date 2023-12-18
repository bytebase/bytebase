package auth

import (
	"strings"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var authenticationAllowlistMethods = map[string]bool{
	v1pb.ActuatorService_GetActuatorInfo_FullMethodName:               true,
	v1pb.ActuatorService_DeleteCache_FullMethodName:                   true,
	v1pb.SubscriptionService_GetSubscription_FullMethodName:           true,
	v1pb.SubscriptionService_GetFeatureMatrix_FullMethodName:          true,
	v1pb.AuthService_Login_FullMethodName:                             true,
	v1pb.AuthService_Logout_FullMethodName:                            true,
	v1pb.AuthService_CreateUser_FullMethodName:                        true,
	v1pb.IdentityProviderService_ListIdentityProviders_FullMethodName: true,
}

// IsAuthenticationAllowed returns whether the method is exempted from authentication.
func IsAuthenticationAllowed(fullMethodName string) bool {
	// TODO(d): skips OpenAPI SQL endpoint.
	// "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo" is used
	//  for reflection, but we'd rather to allow the whole reflection endpoint.
	if strings.HasPrefix(fullMethodName, "/grpc.reflection") {
		return true
	}
	return authenticationAllowlistMethods[fullMethodName]
}
