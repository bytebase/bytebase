// Package gitops is the package for GitOps APIs.
package gitops

import (
	v1pb "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/sheet"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
)

// Service is the API endpoint for handling GitOps requests.
type Service struct {
	store          *store.Store
	licenseService enterprise.LicenseService
	planService    *v1pb.PlanService
	rolloutService *v1pb.RolloutService
	issueService   *v1pb.IssueService
	sqlService     *v1pb.SQLService
	sheetManager   *sheet.Manager
}

// NewService creates a GitOps service.
func NewService(
	store *store.Store,
	licenseService enterprise.LicenseService,
	planService *v1pb.PlanService,
	rolloutService *v1pb.RolloutService,
	issueService *v1pb.IssueService,
	sqlService *v1pb.SQLService,
	sheetManager *sheet.Manager,
) *Service {
	return &Service{
		store:          store,
		licenseService: licenseService,
		planService:    planService,
		rolloutService: rolloutService,
		issueService:   issueService,
		sqlService:     sqlService,
		sheetManager:   sheetManager,
	}
}
