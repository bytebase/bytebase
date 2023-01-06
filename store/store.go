// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import "fmt"

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache *CacheService

	userIDCache        map[int]*UserMessage
	environmentCache   map[string]*EnvironmentMessage
	environmentIDCache map[int]*EnvironmentMessage
	instanceCache      map[string]*InstanceMessage
	instanceIDCache    map[int]*InstanceMessage
	databaseCache      map[string]*DatabaseMessage
	databaseIDCache    map[int]*DatabaseMessage
	projectCache       map[string]*ProjectMessage
	projectIDCache     map[int]*ProjectMessage
	dbSchemaCache      map[int]*DBSchema
}

// New creates a new instance of Store.
func New(db *DB) *Store {
	return &Store{
		db:                 db,
		cache:              newCacheService(),
		userIDCache:        make(map[int]*UserMessage),
		environmentCache:   make(map[string]*EnvironmentMessage),
		environmentIDCache: make(map[int]*EnvironmentMessage),
		instanceCache:      make(map[string]*InstanceMessage),
		instanceIDCache:    make(map[int]*InstanceMessage),
		databaseCache:      make(map[string]*DatabaseMessage),
		databaseIDCache:    make(map[int]*DatabaseMessage),
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

func getDatabaseCacheKey(environmentID, instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s/%s", environmentID, instanceID, databaseName)
}
