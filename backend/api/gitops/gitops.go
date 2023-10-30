// Package gitops is the package for GitOps APIs.
package gitops

import (
	v1pb "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseapi "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
)

// Service is the API endpoint for handling GitOps requests.
type Service struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	stateCfg        *state.State
	licenseService  enterpriseapi.LicenseService
	rolloutService  *v1pb.RolloutService
	issueService    *v1pb.IssueService
}

// NewService creates a GitOps service.
func NewService(
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	activityManager *activity.Manager,
	stateCfg *state.State,
	licenseService enterpriseapi.LicenseService,
	rolloutService *v1pb.RolloutService,
	issueService *v1pb.IssueService,
) *Service {
	return &Service{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		stateCfg:        stateCfg,
		licenseService:  licenseService,
		rolloutService:  rolloutService,
		issueService:    issueService,
	}
}
