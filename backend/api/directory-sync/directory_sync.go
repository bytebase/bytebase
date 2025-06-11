package directorysync

import (
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
)

// Service is the API endpoint for handling SCIM requests.
type Service struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
	iamManager     *iam.Manager
}

// NewService creates a SCIM service.
func NewService(
	store *store.Store,
	licenseService *enterprise.LicenseService,
	iamManager *iam.Manager,
) *Service {
	return &Service{
		store:          store,
		licenseService: licenseService,
		iamManager:     iamManager,
	}
}
