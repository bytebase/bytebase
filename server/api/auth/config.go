package auth

import (
	"strings"
)

const (
	apiPackagePrefix = "/bytebase.v1."
)

var authenticationAllowlistMethods = map[string]bool{
	"AuthService/Login":      true,
	"AuthService/Logout":     true,
	"AuthService/CreateUser": true,
}

var ownerAndDBAMethods = map[string]bool{
	"EnvironmentService/CreateEnvironment":   true,
	"EnvironmentService/UpdateEnvironment":   true,
	"EnvironmentService/DeleteEnvironment":   true,
	"EnvironmentService/UndeleteEnvironment": true,
	"InstanceService/CreateInstance":         true,
	"InstanceService/UpdateInstance":         true,
	"InstanceService/DeleteInstance":         true,
	"InstanceService/UndeleteInstance":       true,
}

func getShortMethodName(fullMethod string) string {
	return strings.TrimPrefix(fullMethod, apiPackagePrefix)
}

func isAuthenticationAllowed(methodName string) bool {
	return authenticationAllowlistMethods[methodName]
}

func isOwnerAndDBAMethod(methodName string) bool {
	return ownerAndDBAMethods[methodName]
}
