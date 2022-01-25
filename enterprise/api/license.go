package api

type License struct {
	Audience      string `jsonapi:"attr,audience"`
	InstanceCount int    `jsonapi:"attr,instanceCount"`
	ExpiresTs     int64  `jsonapi:"attr,expiresTs"`
	Plan          string `jsonapi:"attr,plan"`
}

type LicenseService interface {
	StoreLicense(tokenString string) error
	ParseLicense() (*License, error)
}
