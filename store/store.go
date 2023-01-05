// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import "fmt"

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache *CacheService

	environmentCache   map[string]*EnvironmentMessage
	environmentIDCache map[int]*EnvironmentMessage
	instanceCache      map[string]*InstanceMessage
	projectCache       map[string]*ProjectMessage
	projectIDCache     map[int]*ProjectMessage
	dbSchemaCache      map[int]*DBSchema
}

// New creates a new instance of Store.
func New(db *DB) *Store {
	return &Store{
		db:                 db,
		cache:              newCacheService(),
		environmentCache:   make(map[string]*EnvironmentMessage),
		environmentIDCache: make(map[int]*EnvironmentMessage),
		instanceCache:      make(map[string]*InstanceMessage),
		projectCache:       make(map[string]*ProjectMessage),
		projectIDCache:     make(map[int]*ProjectMessage),
		dbSchemaCache:      make(map[int]*DBSchema),
	}
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.db.Close()
}

func getInstanceCacheKey(environmentID, instanceID string) string {
	return fmt.Sprintf("%s/%s", environmentID, instanceID)
}
