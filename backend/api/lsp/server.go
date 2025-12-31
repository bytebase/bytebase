package lsp

import (
	"sync/atomic"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
)

// Server is the Language Server Protocol service.
type Server struct {
	connectionCount atomic.Uint64

	store           *store.Store
	profile         *config.Profile
	secret          string
	bus             *bus.Bus
	iamManager      *iam.Manager
	licenseService  *enterprise.LicenseService
	authInterceptor *auth.APIAuthInterceptor
}

// NewServer creates a Language Server Protocol service.
func NewServer(
	store *store.Store,
	profile *config.Profile,
	secret string,
	bus *bus.Bus,
	iamManager *iam.Manager,
	licenseService *enterprise.LicenseService,
) *Server {
	return &Server{
		store:           store,
		profile:         profile,
		secret:          secret,
		bus:             bus,
		iamManager:      iamManager,
		licenseService:  licenseService,
		authInterceptor: auth.New(store, secret, licenseService, bus, profile),
	}
}
