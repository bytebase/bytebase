// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import "fmt"

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache *CacheService

	environmentCache map[string]*EnvironmentMessage
	instanceCache    map[string]*InstanceMessage
}

// New creates a new instance of Store.
func New(db *DB) *Store {
	return &Store{
		db:               db,
		cache:            newCacheService(),
		environmentCache: make(map[string]*EnvironmentMessage),
		instanceCache:    make(map[string]*InstanceMessage),
	}
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.db.Close()
}

func getInstanceCacheKey(environmentID, instanceID string) string {
	return fmt.Sprintf("%s/%s", environmentID, instanceID)
}
