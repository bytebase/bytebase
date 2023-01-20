// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"fmt"
	"sync"

	"github.com/bytebase/bytebase/api"
)

// Store provides database access to all raw objects.
type Store struct {
	db    *DB
	cache *CacheService

	userIDCache          sync.Map // map[int]*UserMessage
	userEmailCache       sync.Map // map[string]*UserMessage
	environmentCache     sync.Map // map[string]*EnvironmentMessage
	environmentIDCache   sync.Map // map[int]*EnvironmentMessage
	instanceCache        sync.Map // map[string]*InstanceMessage
	instanceIDCache      sync.Map // map[int]*InstanceMessage
	databaseCache        sync.Map // map[string]*DatabaseMessage
	databaseIDCache      sync.Map // map[int]*DatabaseMessage
	projectCache         sync.Map // map[string]*ProjectMessage
	projectIDCache       sync.Map // map[int]*ProjectMessage
	projectPolicyCache   sync.Map // map[string]*IAMPolicyMessage
	projectIDPolicyCache sync.Map // map[int]*IAMPolicyMessage
	policyCache          sync.Map // map[string]*PolicyMessage
	dbSchemaCache        sync.Map // map[int]*DBSchema
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

func getPolicyCacheKey(resourceType api.PolicyResourceType, resourceUID int, policyType api.PolicyType) string {
	return fmt.Sprintf("policies/%s/%d/%s", resourceType, resourceUID, policyType)
}

func getDatabaseCacheKey(environmentID, instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s/%s", environmentID, instanceID, databaseName)
}
