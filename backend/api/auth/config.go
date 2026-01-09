package auth

import (
	"strings"

	"github.com/bytebase/bytebase/backend/common"
)

// IsAuthenticationSkipped returns whether the method is exempted from authentication.
func IsAuthenticationSkipped(fullMethodName string, authContext *common.AuthContext) bool {
	// "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo" is used
	//  for reflection.
	if strings.HasPrefix(fullMethodName, "/grpc.reflection") {
		return true
	}
	return authContext.AllowWithoutCredential
}
