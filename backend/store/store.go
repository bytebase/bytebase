// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// Store provides database access to all raw objects.
type Store struct {
	dbConnManager *DBConnectionManager
	enableCache   bool

	// Cache.
	Secret            string
	userEmailCache    *lru.Cache[string, *UserMessage]
	instanceCache     *lru.Cache[string, *InstanceMessage]
	databaseCache     *lru.Cache[string, *DatabaseMessage]
	projectCache      *lru.Cache[string, *ProjectMessage]
	policyCache       *lru.Cache[string, *PolicyMessage]
	settingCache      *lru.Cache[storepb.SettingName, *SettingMessage]
	rolesCache        *expirable.LRU[string, *RoleMessage]
	groupCache        *expirable.LRU[string, *GroupMessage]
	groupMembersCache *expirable.LRU[string, map[string]bool]
	memberGroupsCache *expirable.LRU[string, []string]
	dbSchemaCache     *expirable.LRU[string, *model.DatabaseMetadata]

	// Large objects.
	sheetFullCache *lru.Cache[string, *SheetMessage]
}

// New creates a new instance of Store.
// pgURL can be either a direct PostgreSQL URL or a file path containing the URL.
func New(ctx context.Context, pgURL string, enableCache bool) (*Store, error) {
	userEmailCache, err := lru.New[string, *UserMessage](32768)
	if err != nil {
		return nil, err
	}
	instanceCache, err := lru.New[string, *InstanceMessage](32768)
	if err != nil {
		return nil, err
	}
	databaseCache, err := lru.New[string, *DatabaseMessage](32768)
	if err != nil {
		return nil, err
	}
	projectCache, err := lru.New[string, *ProjectMessage](32768)
	if err != nil {
		return nil, err
	}
	policyCache, err := lru.New[string, *PolicyMessage](128)
	if err != nil {
		return nil, err
	}
	settingCache, err := lru.New[storepb.SettingName, *SettingMessage](64)
	if err != nil {
		return nil, err
	}
	rolesCache := expirable.NewLRU[string, *RoleMessage](128, nil, time.Minute)
	sheetFullCache, err := lru.New[string, *SheetMessage](10)
	if err != nil {
		return nil, err
	}
	groupCache := expirable.NewLRU[string, *GroupMessage](1024, nil, time.Minute)
	groupMembersCache := expirable.NewLRU[string, map[string]bool](1024, nil, time.Minute)
	memberGroupsCache := expirable.NewLRU[string, []string](4096, nil, time.Minute)
	dbSchemaCache := expirable.NewLRU[string, *model.DatabaseMetadata](128, nil, 5*time.Minute)

	// Initialize database connection (handles both direct URL and file-based)
	dbConnManager := NewDBConnectionManager(pgURL)
	if err := dbConnManager.Initialize(ctx); err != nil {
		return nil, err
	}

	s := &Store{
		dbConnManager: dbConnManager,
		enableCache:   enableCache,

		// Cache.
		userEmailCache:    userEmailCache,
		instanceCache:     instanceCache,
		databaseCache:     databaseCache,
		projectCache:      projectCache,
		policyCache:       policyCache,
		settingCache:      settingCache,
		rolesCache:        rolesCache,
		sheetFullCache:    sheetFullCache,
		groupCache:        groupCache,
		groupMembersCache: groupMembersCache,
		memberGroupsCache: memberGroupsCache,
		dbSchemaCache:     dbSchemaCache,
	}

	return s, nil
}

// Close closes underlying db.
func (s *Store) Close() error {
	return s.dbConnManager.Close()
}

func (s *Store) GetDB() *sql.DB {
	return s.dbConnManager.GetDB()
}

// DeleteCache deletes the cache.
func (s *Store) DeleteCache() {
	s.settingCache.Purge()
	s.policyCache.Purge()
	s.userEmailCache.Purge()
}

// PurgeGroupCaches purges all group-related caches.
func (s *Store) PurgeGroupCaches() {
	s.groupCache.Purge()
	s.groupMembersCache.Purge()
	s.memberGroupsCache.Purge()
}

func getInstanceCacheKey(instanceID string) string {
	return instanceID
}

func getPolicyCacheKey(resourceType storepb.Policy_Resource, resource string, policyType storepb.Policy_Type) string {
	return fmt.Sprintf("policies/%s/%s/%s", resourceType, resource, policyType)
}

func getDatabaseCacheKey(instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s", instanceID, databaseName)
}

func getDBSchemaCacheKey(instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s", instanceID, databaseName)
}
