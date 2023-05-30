package auth

var authenticationAllowlistMethods = map[string]bool{
	"/bytebase.v1.ActuatorService/GetActuatorInfo":               true,
	"/bytebase.v1.SubscriptionService/GetSubscription":           true,
	"/bytebase.v1.SubscriptionService/GetFeatureMatrix":          true,
	"/bytebase.v1.AuthService/Login":                             true,
	"/bytebase.v1.AuthService/Logout":                            true,
	"/bytebase.v1.AuthService/CreateUser":                        true,
	"/bytebase.v1.IdentityProviderService/ListIdentityProviders": true,
}

// IsAuthenticationAllowed returns whether the method is exempted from authentication.
func IsAuthenticationAllowed(fullMethodName string) bool {
	// TODO(d): skips OpenAPI SQL endpoint.
	return authenticationAllowlistMethods[fullMethodName]
}
