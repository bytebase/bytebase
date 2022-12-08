// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache *CacheService
}

// New creates a new instance of Store.
func New(db *DB) *Store {
	return &Store{
		db:    db,
		cache: newCacheService(),
	}
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.db.Close()
}
