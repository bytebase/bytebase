package store

import (
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// Store provides database access to all raw objects
type Store struct {
	Member MemberStore

	db    *DB
	cache api.CacheService
}

// NewStore creates a new instance of Store
func NewStore(l *zap.Logger, db *DB, cache api.CacheService) *Store {
	store := &Store{
		db:    db,
		cache: cache,
	}
	store.Member = NewMemberStore(l, db, cache, store)
	return store
}
