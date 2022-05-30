package store

import (
	"github.com/bytebase/bytebase/api"
)

// Store provides database access to all raw objects
type Store struct {
	db    *DB
	cache api.CacheService
}

// New creates a new instance of Store
func New(db *DB, cache api.CacheService) *Store {
	return &Store{
		db:    db,
		cache: cache,
	}
}
