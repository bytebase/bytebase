package store

import (
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// Store provides database access to all raw objects
type Store struct {
	l               *zap.Logger
	db              *DB
	cache           api.CacheService
	DatabaseService api.DatabaseService
}

// New creates a new instance of Store
func New(l *zap.Logger, db *DB, cache api.CacheService) *Store {
	return &Store{
		l:     l,
		db:    db,
		cache: cache,
	}
}
