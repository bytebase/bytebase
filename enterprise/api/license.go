package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
)

// validPlans is a string array of valid plan types.
var validPlans = []string{
	api.TEAM.String(),
	api.ENTERPRISE.String(),
}

// License is the API message for enterprise license.
type License struct {
	Audience      string       `jsonapi:"attr,audience"`
	InstanceCount int          `jsonapi:"attr,instanceCount"`
	ExpiresTs     int64        `jsonapi:"attr,expiresTs"`
	Plan          api.PlanType `jsonapi:"attr,plan"`
}

// Valid will check if license expired or has correct plan type.
func (l *License) Valid() error {
	if l.ExpiresTs <= time.Now().Unix() {
		return fmt.Errorf("license has expired at %v", time.Unix(l.ExpiresTs, 0))
	}

	if err := l.validPlanType(); err != nil {
		return err
	}

	return nil
}

func (l *License) validPlanType() error {
	for _, plan := range validPlans {
		if plan == l.Plan.String() {
			return nil
		}
	}

	return fmt.Errorf("plan %q is not valid, expect one of %s",
		l.Plan.String(),
		strings.Join(validPlans, ", "),
	)
}

// LicensePatch is the API message for upload a enterprise license.
type LicensePatch struct {
	Token string `jsonapi:"attr,token"`
}

// LicenseService is the service for enterprise license.
type LicenseService interface {
	// StoreLicense will store license into file.
	StoreLicense(tokenString string) error
	// ParseLicense will valid and parse license from file.
	ParseLicense() (*License, error)
}
