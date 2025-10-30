package lsp

import (
	"sync/atomic"

	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/store"
)

// Server is the Language Server Protocol service.
type Server struct {
	connectionCount atomic.Uint64

	store      *store.Store
	profile    *config.Profile
	secret     string
	stateCfg   *state.State
	iamManager *iam.Manager
}

// NewServer creates a Language Server Protocol service.
func NewServer(
	store *store.Store,
	profile *config.Profile,
	secret string,
	stateCfg *state.State,
	iamManager *iam.Manager,
) *Server {
	return &Server{
		store:      store,
		profile:    profile,
		secret:     secret,
		stateCfg:   stateCfg,
		iamManager: iamManager,
	}
}
