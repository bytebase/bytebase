package lsp

import (
	"sync/atomic"

	"github.com/bytebase/bytebase/backend/store"
)

// Server is the Language Server Protocol service.
type Server struct {
	connectionCount atomic.Uint64

	store *store.Store
}

// NewServer creates a Language Server Protocol service.
func NewServer(
	store *store.Store,
) *Server {
	return &Server{
		store: store,
	}
}
