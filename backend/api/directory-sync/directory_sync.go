package directorysync

import (
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
)

// Service is the API endpoint for handling SCIM requests.
type Service struct {
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewService creates a SCIM service.
func NewService(
	store *store.Store,
	licenseService enterprise.LicenseService,
) *Service {
	return &Service{
		store:          store,
		licenseService: licenseService,
	}
}
