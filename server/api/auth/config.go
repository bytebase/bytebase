package auth

import (
	"fmt"
	"strings"
)

const apiPackagePrefix = "/bytebase.v1."

var authenticationAllowlistMethods = []string{
	"AuthService/Login",
}

func isAuthenticationAllowed(fullMethod string) bool {
	for _, allow := range authenticationAllowlistMethods {
		if strings.HasPrefix(fullMethod, fmt.Sprintf("%s%s", apiPackagePrefix, allow)) {
			return true
		}
	}
	return false
}
