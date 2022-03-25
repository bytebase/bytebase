package store

import (
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// Store provides database access to all raw objects
type Store struct {
	l     *zap.Logger
	db    *DB
	cache api.CacheService
}

// NewStore creates a new instance of Store
func NewStore(l *zap.Logger, db *DB, cache api.CacheService) *Store {
	return &Store{
		l:     l,
		db:    db,
		cache: cache,
	}
}
