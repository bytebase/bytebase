// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"github.com/bytebase/bytebase/api"
)

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache api.CacheService
}

// New creates a new instance of Store.
func New(db *DB, cache api.CacheService) *Store {
	return &Store{
		db:    db,
		cache: cache,
	}
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.db.Close()
}
