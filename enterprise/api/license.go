package api

import (
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
)

// validPlans is a string array of valid plan types.
var validPlans = []api.PlanType{
	api.TEAM,
	api.ENTERPRISE,
}

// License is the API message for enterprise license.
type License struct {
	Subject       string
	InstanceCount int
	ExpiresTs     int64
	IssuedTs      int64
	Plan          api.PlanType
	Trialing      bool
}

// Valid will check if license expired or has correct plan type.
func (l *License) Valid() error {
	if expireTime := time.Unix(l.ExpiresTs, 0); expireTime.Before(time.Now()) {
		return fmt.Errorf("license has expired at %v", expireTime)
	}

	if err := l.validPlanType(); err != nil {
		return err
	}

	return nil
}

func (l *License) validPlanType() error {
	for _, plan := range validPlans {
		if plan == l.Plan {
			return nil
		}
	}

	return fmt.Errorf("plan %q is not valid, expect %s or %s",
		l.Plan.String(),
		api.TEAM.String(),
		api.ENTERPRISE.String(),
	)
}

// LicenseService is the service for enterprise license.
type LicenseService interface {
	// StoreLicense will store license into file.
	StoreLicense(patch *SubscriptionPatch) error
	// LoadLicense will load license from file and validate it.
	LoadLicense() (*License, error)
}
