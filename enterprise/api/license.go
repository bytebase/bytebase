package api

type PlanType string

const (
	FREE       PlanType = "FREE"
	TEAM       PlanType = "TEAM"
	ENTERPRISE PlanType = "ENTERPRISE"
)

func (p PlanType) String() string {
	switch p {
	case FREE:
		return "FREE"
	case TEAM:
		return "TEAM"
	case ENTERPRISE:
		return "ENTERPRISE"
	}
	return ""
}

type License struct {
	Audience      string   `jsonapi:"attr,audience"`
	InstanceCount int      `jsonapi:"attr,instanceCount"`
	ExpiresAt     int64    `jsonapi:"attr,expiresAt"`
	Plan          PlanType `jsonapi:"attr,plan"`
}

type LicenseService interface {
	StoreLicense(tokenString string) error
	ParseLicense() (*License, error)
}
