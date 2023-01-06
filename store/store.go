// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"fmt"
	"sync"
)

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache *CacheService

	userIDCache        sync.Map // map[int]*UserMessage
	environmentCache   sync.Map // map[string]*EnvironmentMessage
	environmentIDCache sync.Map // map[int]*EnvironmentMessage
	instanceCache      sync.Map // map[string]*InstanceMessage
	instanceIDCache    sync.Map // map[int]*InstanceMessage
	databaseCache      sync.Map // map[string]*DatabaseMessage
	databaseIDCache    sync.Map // map[int]*DatabaseMessage
	projectCache       sync.Map // map[string]*ProjectMessage
	projectIDCache     sync.Map // map[int]*ProjectMessage
	dbSchemaCache      sync.Map // map[int]*DBSchema
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

func getInstanceCacheKey(environmentID, instanceID string) string {
	return fmt.Sprintf("%s/%s", environmentID, instanceID)
}

func getDatabaseCacheKey(environmentID, instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s/%s", environmentID, instanceID, databaseName)
}
