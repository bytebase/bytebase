package store

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// Store provides database access to all raw objects
type Store struct {
	db      *DB
	cache   api.CacheService
	profile common.Profile
}

// New creates a new instance of Store
func New(db *DB, cache api.CacheService, profile common.Profile) *Store {
	return &Store{
		db:      db,
		cache:   cache,
		profile: profile,
	}
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.db.Close()
}
