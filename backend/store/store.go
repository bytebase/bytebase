// Package store is the implementation for managing Bytebase's own metadata in a PostgreSQL database.
package store

import (
	"context"
	"fmt"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	protojsonUnmarshaler = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
)

// Store provides database access to all raw objects.
type Store struct {
	db *DB

	userIDCache                    sync.Map // map[int]*UserMessage
	environmentCache               sync.Map // map[string]*EnvironmentMessage
	environmentIDCache             sync.Map // map[int]*EnvironmentMessage
	instanceCache                  sync.Map // map[string]*InstanceMessage
	instanceIDCache                sync.Map // map[int]*InstanceMessage
	databaseCache                  sync.Map // map[string]*DatabaseMessage
	databaseIDCache                sync.Map // map[int]*DatabaseMessage
	projectCache                   sync.Map // map[string]*ProjectMessage
	projectIDCache                 sync.Map // map[int]*ProjectMessage
	projectPolicyCache             sync.Map // map[string]*IAMPolicyMessage
	projectIDPolicyCache           sync.Map // map[int]*IAMPolicyMessage
	policyCache                    sync.Map // map[string]*PolicyMessage
	issueCache                     sync.Map // map[int]*IssueMessage
	issueByPipelineCache           sync.Map // map[int]*IssueMessage
	pipelineCache                  sync.Map // map[int]*PipelineMessage
	settingCache                   sync.Map // map[string]*SettingMessage
	idpCache                       sync.Map // map[string]*IdentityProvider
	projectIDDeploymentConfigCache sync.Map // map[int]*DeploymentConfigMessage
	risksCache                     sync.Map // []*RiskMessage, use 0 as the key
	databaseGroupCache             *lru.Cache[string, *DatabaseGroupMessage]
	databaseGroupIDCache           *lru.Cache[int64, *DatabaseGroupMessage]
	schemaGroupCache               *lru.Cache[string, *SchemaGroupMessage]
	vcsIDCache                     *lru.Cache[int, *ExternalVersionControlMessage]

	// Large objects.
	sheetCache    *lru.Cache[int, string]
	dbSchemaCache *lru.Cache[int, *model.DBSchema]
}

// New creates a new instance of Store.
func New(db *DB) (*Store, error) {
	databaseGroupCache, err := lru.New[string, *DatabaseGroupMessage](10)
	if err != nil {
		return nil, err
	}
	databaseGroupIDCache, err := lru.New[int64, *DatabaseGroupMessage](10)
	if err != nil {
		return nil, err
	}
	schemaGroupCache, err := lru.New[string, *SchemaGroupMessage](10)
	if err != nil {
		return nil, err
	}
	vcsIDCache, err := lru.New[int, *ExternalVersionControlMessage](10)
	if err != nil {
		return nil, err
	}

	sheetCache, err := lru.New[int, string](10)
	if err != nil {
		return nil, err
	}
	dbSchemaCache, err := lru.New[int, *model.DBSchema](100)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: db,

		databaseGroupCache:   databaseGroupCache,
		databaseGroupIDCache: databaseGroupIDCache,
		schemaGroupCache:     schemaGroupCache,
		vcsIDCache:           vcsIDCache,

		sheetCache:    sheetCache,
		dbSchemaCache: dbSchemaCache,
	}, nil
}

// Close closes underlying db.
func (s *Store) Close(ctx context.Context) error {
	return s.db.Close(ctx)
}

func getInstanceCacheKey(instanceID string) string {
	return instanceID
}

func getPolicyCacheKey(resourceType api.PolicyResourceType, resourceUID int, policyType api.PolicyType) string {
	return fmt.Sprintf("policies/%s/%d/%s", resourceType, resourceUID, policyType)
}

func getDatabaseCacheKey(instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s", instanceID, databaseName)
}

func getDatabaseGroupCacheKey(projectUID int, databaseGroupResourceID string) string {
	return fmt.Sprintf("%d/%s", projectUID, databaseGroupResourceID)
}

func getSchemaGroupCacheKey(databaseGroupUID int64, schemaGroupResourceID string) string {
	return fmt.Sprintf("%d/%s", databaseGroupUID, schemaGroupResourceID)
}

func getPlaceholders(start int, count int) string {
	var placeholders []string
	for i := start; i < start+count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
	}
	return fmt.Sprintf("(%s)", strings.Join(placeholders, ","))
}
