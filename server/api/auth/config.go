package auth

var authenticationAllowlistMethods = map[string]bool{
	"/bytebase.v1.AuthService/Login":      true,
	"/bytebase.v1.AuthService/Logout":     true,
	"/bytebase.v1.AuthService/CreateUser": true,
}

// IsAuthenticationAllowed returns whether the method is exempted from authentication.
func IsAuthenticationAllowed(fullMethodName string) bool {
	return authenticationAllowlistMethods[fullMethodName]
}
