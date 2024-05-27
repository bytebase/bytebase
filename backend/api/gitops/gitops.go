// Package gitops is the package for GitOps APIs.
package gitops

import (
	v1pb "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
)

// Service is the API endpoint for handling GitOps requests.
type Service struct {
	store          *store.Store
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	licenseService enterprise.LicenseService
	planService    *v1pb.PlanService
	rolloutService *v1pb.RolloutService
	issueService   *v1pb.IssueService
}

// NewService creates a GitOps service.
func NewService(
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	stateCfg *state.State,
	licenseService enterprise.LicenseService,
	planService *v1pb.PlanService,
	rolloutService *v1pb.RolloutService,
	issueService *v1pb.IssueService,
) *Service {
	return &Service{
		store:          store,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		licenseService: licenseService,
		planService:    planService,
		rolloutService: rolloutService,
		issueService:   issueService,
	}
}
