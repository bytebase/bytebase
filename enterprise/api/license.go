package api

type License struct {
	Audience      string `jsonapi:"attr,audience"`
	InstanceCount int    `jsonapi:"attr,instanceCount"`
	ExpiresTs     int64  `jsonapi:"attr,expiresTs"`
	Plan          string `jsonapi:"attr,plan"`
}

type LicenseService interface {
	// Store license into file
	StoreLicense(tokenString string) error
	// Valid and parse license from file
	ParseLicense() (*License, error)
}
