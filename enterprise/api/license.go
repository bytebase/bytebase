package api

// License is the API message for exterprise license.
type License struct {
	Audience      string `jsonapi:"attr,audience"`
	InstanceCount int    `jsonapi:"attr,instanceCount"`
	ExpiresTs     int64  `jsonapi:"attr,expiresTs"`
	Plan          string `jsonapi:"attr,plan"`
}

// LicenseService is the service for exterprise license.
type LicenseService interface {
	// StoreLicense will store license into file.
	StoreLicense(tokenString string) error
	// ParseLicense will valid and parse license from file.
	ParseLicense() (*License, error)
}
