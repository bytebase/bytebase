// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"sync"

	"github.com/bytebase/bytebase/api"
)

// Store provides database access to all raw objects.
type Store struct {
	db *DB
	// Cache for
	cache           api.CacheService
	dataSourceCache sync.Map
}

// New creates a new instance of Store.
func New(db *DB, cache api.CacheService) *Store {
	return &Store{
		db: db,
		// data cache.
		cache:           cache,
		dataSourceCache: sync.Map{},
	}
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.db.Close()
}
