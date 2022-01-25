package api

type License struct {
	Audience      string `jsonapi:"attr,audience"`
	InstanceCount int    `jsonapi:"attr,instanceCount"`
	ExpiresAt     int64  `jsonapi:"attr,expiresAt"`
	Plan          string `jsonapi:"attr,plan"`
}

type LicenseService interface {
	StoreLicense(tokenString string) error
	ParseLicense() (*License, error)
}
