package store

import (
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// Store provides database access to all raw objects
type Store struct {
	db    *DB
	cache api.CacheService
}

// NewStore creates a new instance of Store
func NewStore(l *zap.Logger, db *DB, cache api.CacheService) *Store {
	return &Store{
		db:    db,
		cache: cache,
	}
}
