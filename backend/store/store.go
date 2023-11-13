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

	userIDCache            sync.Map // map[int]*UserMessage
	environmentCache       sync.Map // map[string]*EnvironmentMessage
	environmentIDCache     sync.Map // map[int]*EnvironmentMessage
	instanceCache          sync.Map // map[string]*InstanceMessage
	instanceIDCache        sync.Map // map[int]*InstanceMessage
	databaseCache          sync.Map // map[string]*DatabaseMessage
	databaseIDCache        sync.Map // map[int]*DatabaseMessage
	projectCache           sync.Map // map[string]*ProjectMessage
	projectIDCache         sync.Map // map[int]*ProjectMessage
	projectPolicyCache     *lru.Cache[string, *IAMPolicyMessage]
	projectIDPolicyCache   *lru.Cache[int, *IAMPolicyMessage]
	projectDeploymentCache *lru.Cache[int, *DeploymentConfigMessage]
	policyCache            *lru.Cache[string, *PolicyMessage]
	issueCache             *lru.Cache[int, *IssueMessage]
	issueByPipelineCache   *lru.Cache[int, *IssueMessage]
	pipelineCache          *lru.Cache[int, *PipelineMessage]
	settingCache           *lru.Cache[api.SettingName, *SettingMessage]
	idpCache               *lru.Cache[string, *IdentityProviderMessage]
	risksCache             *lru.Cache[int, []*RiskMessage] // Use 0 as the key.
	databaseGroupCache     *lru.Cache[string, *DatabaseGroupMessage]
	databaseGroupIDCache   *lru.Cache[int64, *DatabaseGroupMessage]
	schemaGroupCache       *lru.Cache[string, *SchemaGroupMessage]
	vcsIDCache             *lru.Cache[int, *ExternalVersionControlMessage]

	// Large objects.
	sheetCache    *lru.Cache[int, string]
	dbSchemaCache *lru.Cache[int, *model.DBSchema]
}

// New creates a new instance of Store.
func New(db *DB) (*Store, error) {
	projectPolicyCache, err := lru.New[string, *IAMPolicyMessage](128)
	if err != nil {
		return nil, err
	}
	projectIDPolicyCache, err := lru.New[int, *IAMPolicyMessage](128)
	if err != nil {
		return nil, err
	}
	projectDeploymentCache, err := lru.New[int, *DeploymentConfigMessage](128)
	if err != nil {
		return nil, err
	}
	policyCache, err := lru.New[string, *PolicyMessage](128)
	if err != nil {
		return nil, err
	}
	issueCache, err := lru.New[int, *IssueMessage](256)
	if err != nil {
		return nil, err
	}
	issueByPipelineCache, err := lru.New[int, *IssueMessage](256)
	if err != nil {
		return nil, err
	}
	pipelineCache, err := lru.New[int, *PipelineMessage](256)
	if err != nil {
		return nil, err
	}
	settingCache, err := lru.New[api.SettingName, *SettingMessage](64)
	if err != nil {
		return nil, err
	}
	idpCache, err := lru.New[string, *IdentityProviderMessage](4)
	if err != nil {
		return nil, err
	}
	risksCache, err := lru.New[int, []*RiskMessage](1)
	if err != nil {
		return nil, err
	}
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

		projectPolicyCache:     projectPolicyCache,
		projectIDPolicyCache:   projectIDPolicyCache,
		projectDeploymentCache: projectDeploymentCache,
		policyCache:            policyCache,
		issueCache:             issueCache,
		issueByPipelineCache:   issueByPipelineCache,
		pipelineCache:          pipelineCache,
		settingCache:           settingCache,
		idpCache:               idpCache,
		risksCache:             risksCache,
		databaseGroupCache:     databaseGroupCache,
		databaseGroupIDCache:   databaseGroupIDCache,
		schemaGroupCache:       schemaGroupCache,
		vcsIDCache:             vcsIDCache,
		sheetCache:             sheetCache,
		dbSchemaCache:          dbSchemaCache,
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
