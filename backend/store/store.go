// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"fmt"
	"sync"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// Store provides database access to all raw objects.
type Store struct {
	db *DB

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
	issueCache           sync.Map // map[int]*IssueMessage
	issueByPipelineCache sync.Map // map[int]*IssueMessage
	pipelineCache        sync.Map // map[int]*PipelineMessage
	dbSchemaCache        sync.Map // map[int]*DBSchema
	settingCache         sync.Map // map[string]*SettingMessage
}

// New creates a new instance of Store.
func New(db *DB) *Store {
	return &Store{
		db: db,
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
