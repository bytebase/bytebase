package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type IamPolicyMessage struct {
	Policy *storepb.IamPolicy
	Etag   string
}

func getIamPolicyCacheKey(workspace string, resourceType storepb.Policy_Resource, resourceID string) string {
	return fmt.Sprintf("iam/%s/%s/%s", workspace, resourceType, resourceID)
}

// generateEtag generates etag for the given body.
func generateEtag(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixMilli())
}

func (s *Store) GetWorkspaceIamPolicy(ctx context.Context, workspaceID string) (*IamPolicyMessage, error) {
	return s.getIamPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: new(storepb.Policy_WORKSPACE),
		Resource:     new(common.FormatWorkspace(workspaceID)),
	})
}

type PatchIamPolicyMessage struct {
	Workspace string
	Member    string
	Roles     []string
}

// PatchWorkspaceIamPolicy will set or remove the member for the workspace role.
func (s *Store) PatchWorkspaceIamPolicy(ctx context.Context, patch *PatchIamPolicyMessage) (*IamPolicyMessage, error) {
	workspaceResource := common.FormatWorkspace(patch.Workspace)

	workspaceIamPolicy, err := s.GetWorkspaceIamPolicy(ctx, patch.Workspace)
	if err != nil {
		return nil, err
	}

	roleMap := map[string]bool{}
	for _, role := range patch.Roles {
		roleMap[role] = true
	}

	for _, binding := range workspaceIamPolicy.Policy.Bindings {
		index := slices.Index(binding.Members, patch.Member)
		if !roleMap[binding.Role] {
			if index >= 0 {
				binding.Members = slices.Delete(binding.Members, index, index+1)
			}
		} else {
			if index < 0 {
				binding.Members = append(binding.Members, patch.Member)
			}
		}

		delete(roleMap, binding.Role)
	}

	for role := range roleMap {
		workspaceIamPolicy.Policy.Bindings = append(workspaceIamPolicy.Policy.Bindings, &storepb.Binding{
			Role: role,
			Members: []string{
				patch.Member,
			},
		})
	}

	policyPayload, err := protojson.Marshal(workspaceIamPolicy.Policy)
	if err != nil {
		return nil, err
	}

	if _, err := s.CreatePolicy(ctx, &PolicyMessage{
		Workspace:         patch.Workspace,
		ResourceType:      storepb.Policy_WORKSPACE,
		Resource:          workspaceResource,
		Payload:           string(policyPayload),
		Type:              storepb.Policy_IAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}); err != nil {
		return nil, err
	}

	// Invalidate caches after mutation.
	s.iamPolicyCache.Remove(getIamPolicyCacheKey(patch.Workspace, storepb.Policy_WORKSPACE, workspaceResource))

	return s.GetWorkspaceIamPolicy(ctx, patch.Workspace)
}

func (s *Store) GetProjectIamPolicy(ctx context.Context, workspaceID string, projectID string) (*IamPolicyMessage, error) {
	return s.getIamPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: new(storepb.Policy_PROJECT),
		Resource:     new(common.FormatProject(projectID)),
	})
}

func (s *Store) GetWorkspaceIamPolicySnapshot(ctx context.Context, workspaceID string) (*IamPolicyMessage, error) {
	workspaceResource := common.FormatWorkspace(workspaceID)
	key := getIamPolicyCacheKey(workspaceID, storepb.Policy_WORKSPACE, workspaceResource)
	if v, ok := s.iamPolicyCache.Get(key); ok {
		return v, nil
	}
	return s.GetWorkspaceIamPolicy(ctx, workspaceID)
}

func (s *Store) GetProjectIamPolicySnapshot(ctx context.Context, workspaceID string, projectID string) (*IamPolicyMessage, error) {
	key := getIamPolicyCacheKey(workspaceID, storepb.Policy_PROJECT, projectID)
	if v, ok := s.iamPolicyCache.Get(key); ok {
		return v, nil
	}
	return s.GetProjectIamPolicy(ctx, workspaceID, projectID)
}

func (s *Store) getIamPolicy(ctx context.Context, find *FindPolicyMessage) (*IamPolicyMessage, error) {
	find.Type = new(storepb.Policy_IAM)
	policy, err := s.GetPolicy(ctx, find)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return &IamPolicyMessage{
			Policy: &storepb.IamPolicy{},
		}, nil
	}

	p := &storepb.IamPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal iam policy for %v", policy.Resource)
	}

	return &IamPolicyMessage{
		Policy: p,
		Etag:   generateEtag(policy.UpdatedAt),
	}, nil
}

// GetDefaultRolloutPolicy returns the default rollout policy when no custom policy exists.
// This is used as a fallback for both API and store layers to ensure consistent defaults.
// Default values:
// - automatic: false (manual rollout required)
// - roles: [] (no role restrictions)
// - requiredIssueApproval: true (issue must be approved before rollout)
// - planCheckEnforcement: ERROR_ONLY (block rollout only on errors, not warnings)
func GetDefaultRolloutPolicy() *storepb.RolloutPolicy {
	return &storepb.RolloutPolicy{
		Automatic: false,
		Roles:     []string{},
	}
}

func (s *Store) GetRolloutPolicy(ctx context.Context, workspaceID string, environment string) (*storepb.RolloutPolicy, error) {
	policy, err := s.GetPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: new(storepb.Policy_ENVIRONMENT),
		Resource:     new(common.FormatEnvironment(environment)),
		Type:         new(storepb.Policy_ROLLOUT),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy")
	}
	if policy == nil {
		return GetDefaultRolloutPolicy(), nil
	}

	p := &storepb.RolloutPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal rollout policy")
	}

	return p, nil
}

type EffectiveQueryDataPolicy struct {
	MaximumResultSize        int64
	MaximumResultRows        int32
	DisableCopyData          bool
	DisableExport            bool
	MaxQueryTimeoutInSeconds int64
	AllowAdminDataSource     bool
}

func formatEffectiveQueryDataPolicy(policy *storepb.QueryDataPolicy) *EffectiveQueryDataPolicy {
	maximumResultRows := policy.GetMaximumResultRows()
	if maximumResultRows <= 0 {
		maximumResultRows = math.MaxInt32
	}

	return &EffectiveQueryDataPolicy{
		MaximumResultRows:    maximumResultRows,
		DisableCopyData:      policy.GetDisableCopyData(),
		DisableExport:        policy.GetDisableExport(),
		AllowAdminDataSource: policy.GetAllowAdminDataSource(),
	}
}

func (s *Store) GetEffectiveQueryDataPolicy(ctx context.Context, workspaceID string, projectFullName string) (*EffectiveQueryDataPolicy, error) {
	workspaceResource := common.FormatWorkspace(workspaceID)
	workspacePolicy, err := s.getQueryDataPolicy(ctx, workspaceID, workspaceResource)
	if err != nil {
		return nil, err
	}
	projectPolicy, err := s.getQueryDataPolicy(ctx, workspaceID, projectFullName)
	if err != nil {
		return nil, err
	}

	formatWorkspacePolicy := formatEffectiveQueryDataPolicy(workspacePolicy)
	formatProjectPolicy := formatEffectiveQueryDataPolicy(projectPolicy)

	maximumResultSize, err := s.GetSQLResultSize(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	queryTimeout, err := s.GetQueryTimeoutInSeconds(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return &EffectiveQueryDataPolicy{
		DisableCopyData:          formatWorkspacePolicy.DisableCopyData,
		DisableExport:            formatWorkspacePolicy.DisableExport,
		MaximumResultRows:        min(formatWorkspacePolicy.MaximumResultRows, formatProjectPolicy.MaximumResultRows),
		MaximumResultSize:        maximumResultSize,
		MaxQueryTimeoutInSeconds: queryTimeout,
		AllowAdminDataSource:     formatWorkspacePolicy.AllowAdminDataSource,
	}, nil
}

func (s *Store) getQueryDataPolicy(ctx context.Context, workspaceID string, resource string) (*storepb.QueryDataPolicy, error) {
	resourceType, _, err := common.GetPolicyResourceTypeAndResource(resource)
	if err != nil {
		return nil, err
	}
	policy, err := s.GetPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         new(storepb.Policy_QUERY_DATA),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy")
	}
	if policy == nil {
		return &storepb.QueryDataPolicy{}, nil
	}

	p := &storepb.QueryDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal query data policy")
	}
	return p, nil
}

type reviewConfigResource struct {
	resourceType storepb.Policy_Resource
	resource     string
}

// GetReviewConfigForDatabase will get the review config for a database.
func (s *Store) GetReviewConfigForDatabase(ctx context.Context, workspaceID string, database *DatabaseMessage) (*storepb.ReviewConfigPayload, error) {
	resources := []*reviewConfigResource{}
	if database.EffectiveEnvironmentID != nil {
		resources = append(resources, &reviewConfigResource{
			resourceType: storepb.Policy_ENVIRONMENT,
			resource:     common.FormatEnvironment(*database.EffectiveEnvironmentID),
		})
	}
	resources = append(resources, &reviewConfigResource{
		resourceType: storepb.Policy_PROJECT,
		resource:     common.FormatProject(database.ProjectID),
	})
	for _, v := range resources {
		reviewConfig, err := s.getReviewConfigByResource(ctx, workspaceID, v.resourceType, v.resource)
		if err != nil {
			slog.Debug("failed to get review config", slog.String("resource_type", v.resourceType.String()), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		if reviewConfig == nil {
			slog.Debug("review config is empty", slog.String("resource_type", v.resourceType.String()), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		return reviewConfig, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy for database %s not found", database.DatabaseName)}
}

func (s *Store) getReviewConfigByResource(ctx context.Context, workspaceID string, resourceType storepb.Policy_Resource, resource string) (*storepb.ReviewConfigPayload, error) {
	policy, err := s.GetPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         new(storepb.Policy_TAG),
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("tag policy for resource %v/%s not found", resourceType, resource)}
	}
	if !policy.Enforce {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("tag policy is not enforced for resource %v/%s", resourceType, resource)}
	}

	payload := &storepb.TagPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal tag policy payload")
	}

	reviewConfigName, ok := payload.Tags[common.ReservedTagReviewConfig]
	if !ok {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config tag for resource %v/%s not found", resourceType, resource)}
	}
	reviewConfigID, err := common.GetReviewConfigID(reviewConfigName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract review config %s", reviewConfigName)
	}

	reviewConfig, err := s.GetReviewConfig(ctx, workspaceID, reviewConfigID)
	if err != nil {
		return nil, err
	}
	if reviewConfig == nil {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config for resource %v/%s not found", resourceType, resource)}
	}
	if !reviewConfig.Enforce {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config is not enforced for resource %v/%s", resourceType, resource)}
	}

	return reviewConfig.Payload, nil
}

// GetMaskingRulePolicy will get the masking rule policy.
func (s *Store) GetMaskingRulePolicy(ctx context.Context, workspaceID string) (*storepb.MaskingRulePolicy, error) {
	policy, err := s.GetPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: new(storepb.Policy_WORKSPACE),
		Resource:     new(common.FormatWorkspace(workspaceID)),
		Type:         new(storepb.Policy_MASKING_RULE),
	})
	if err != nil {
		return nil, err
	}

	if policy == nil {
		return &storepb.MaskingRulePolicy{}, nil
	}

	p := new(storepb.MaskingRulePolicy)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, err
	}

	return p, nil
}

// GetMaskingExemptionPolicyByProject gets the masking exemption policy for a project.
func (s *Store) GetMaskingExemptionPolicyByProject(ctx context.Context, workspaceID string, projectID string) (*storepb.MaskingExemptionPolicy, error) {
	policy, err := s.GetPolicy(ctx, &FindPolicyMessage{
		Workspace:    workspaceID,
		ResourceType: new(storepb.Policy_PROJECT),
		Resource:     new(common.FormatProject(projectID)),
		Type:         new(storepb.Policy_MASKING_EXEMPTION),
	})
	if err != nil {
		return nil, err
	}

	if policy == nil {
		return &storepb.MaskingExemptionPolicy{}, nil
	}

	p := new(storepb.MaskingExemptionPolicy)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, err
	}

	return p, nil
}

// PolicyMessage is the mssage for policy.
type PolicyMessage struct {
	Workspace         string
	Resource          string
	ResourceType      storepb.Policy_Resource
	Payload           string
	InheritFromParent bool
	Type              storepb.Policy_Type
	Enforce           bool

	UpdatedAt time.Time
}

// FindPolicyMessage is the message for finding policies.
type FindPolicyMessage struct {
	Workspace    string
	ResourceType *storepb.Policy_Resource
	Resource     *string
	Type         *storepb.Policy_Type
	// ShowAll will show all policies regardless of the enforce status.
	ShowAll bool
}

// UpdatePolicyMessage is the message for updating a policy.
type UpdatePolicyMessage struct {
	ResourceType      storepb.Policy_Resource
	Resource          string
	Type              storepb.Policy_Type
	Workspace         string
	InheritFromParent *bool
	Payload           *string
	Enforce           *bool
}

// GetPolicy gets a policy.
func (s *Store) GetPolicy(ctx context.Context, find *FindPolicyMessage) (*PolicyMessage, error) {
	if find.ResourceType != nil && find.Resource != nil && find.Type != nil {
		if v, ok := s.policyCache.Get(getPolicyCacheKey(find.Workspace, *find.ResourceType, *find.Resource, *find.Type)); ok && s.enableCache {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowAll = true
	policies, err := s.ListPolicies(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		// Cache the policy for not found as well to reduce the look up latency.
		if find.ResourceType != nil && find.Resource != nil && find.Type != nil {
			s.policyCache.Add(getPolicyCacheKey(find.Workspace, *find.ResourceType, *find.Resource, *find.Type), nil)
		}
		return nil, nil
	}
	if len(policies) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d policies with filter %+v, expect 1", len(policies), find)}
	}
	policy := policies[0]

	s.policyCache.Add(getPolicyCacheKey(policy.Workspace, policy.ResourceType, policy.Resource, policy.Type), policy)

	return policy, nil
}

// ListPolicies lists all policies.
func (s *Store) ListPolicies(ctx context.Context, find *FindPolicyMessage) ([]*PolicyMessage, error) {
	q := qb.Q().Space(`
		SELECT
			workspace,
			updated_at,
			resource_type,
			resource,
			inherit_from_parent,
			type,
			payload,
			enforce
		FROM policy
		WHERE workspace = ?
	`, find.Workspace)

	if v := find.ResourceType; v != nil {
		q.And("resource_type = ?", v.String())
	}
	if v := find.Resource; v != nil {
		q.And("resource = ?", *v)
	}
	if v := find.Type; v != nil {
		q.And("type = ?", v.String())
	}
	if !find.ShowAll {
		q.And("enforce = ?", true)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policyList []*PolicyMessage
	for rows.Next() {
		var policyMessage PolicyMessage
		var resourceTypeString, typeString string
		if err := rows.Scan(
			&policyMessage.Workspace,
			&policyMessage.UpdatedAt,
			&resourceTypeString,
			&policyMessage.Resource,
			&policyMessage.InheritFromParent,
			&typeString,
			&policyMessage.Payload,
			&policyMessage.Enforce,
		); err != nil {
			return nil, err
		}
		resourceTypeValue, ok := storepb.Policy_Resource_value[resourceTypeString]
		if !ok {
			return nil, errors.Errorf("invalid policy resource type string: %s", resourceTypeString)
		}
		policyMessage.ResourceType = storepb.Policy_Resource(resourceTypeValue)
		value, ok := storepb.Policy_Type_value[typeString]
		if !ok {
			return nil, errors.Errorf("invalid policy type string: %s", typeString)
		}
		policyMessage.Type = storepb.Policy_Type(value)
		policyList = append(policyList, &policyMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, policy := range policyList {
		s.policyCache.Add(getPolicyCacheKey(policy.Workspace, policy.ResourceType, policy.Resource, policy.Type), policy)
	}

	return policyList, nil
}

// CreatePolicy creates a policy.
func (s *Store) CreatePolicy(ctx context.Context, create *PolicyMessage) (*PolicyMessage, error) {
	if create.Workspace == "" {
		return nil, errors.Errorf("workspace is required to create policy (resource_type=%s, resource=%s, type=%s)", create.ResourceType, create.Resource, create.Type)
	}
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	policy, err := upsertPolicyImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policy.Workspace, policy.ResourceType, policy.Resource, policy.Type), policy)
	if policy.Type == storepb.Policy_IAM {
		s.iamPolicyCache.Remove(getIamPolicyCacheKey(policy.Workspace, policy.ResourceType, policy.Resource))
	}

	return policy, nil
}

// UpdatePolicy updates the policy.
func (s *Store) UpdatePolicy(ctx context.Context, patch *UpdatePolicyMessage) (*PolicyMessage, error) {
	set := qb.Q()
	set.Comma("updated_at = ?", time.Now())
	if v := patch.InheritFromParent; v != nil {
		set.Comma("inherit_from_parent = ?", *v)
	}
	if v := patch.Payload; v != nil {
		set.Comma("payload = ?", *v)
	}
	if v := patch.Enforce; v != nil {
		set.Comma("enforce = ?", *v)
	}

	query, args, err := qb.Q().Space("UPDATE policy SET ? WHERE resource_type = ? AND resource = ? AND type = ? AND workspace = ? RETURNING payload, inherit_from_parent, enforce, updated_at", set, patch.ResourceType, patch.Resource, patch.Type.String(), patch.Workspace).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	policy := &PolicyMessage{
		Workspace:    patch.Workspace,
		Resource:     patch.Resource,
		ResourceType: patch.ResourceType,
		Type:         patch.Type,
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&policy.Payload,
		&policy.InheritFromParent,
		&policy.Enforce,
		&policy.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(patch.Workspace, patch.ResourceType, patch.Resource, patch.Type), policy)
	if patch.Type == storepb.Policy_IAM {
		s.iamPolicyCache.Remove(getIamPolicyCacheKey(patch.Workspace, patch.ResourceType, patch.Resource))
	}

	return policy, nil
}

// DeletePolicy deletes the policy.
func (s *Store) DeletePolicy(ctx context.Context, policy *PolicyMessage) error {
	q := qb.Q().Space("DELETE FROM policy WHERE resource_type = ? AND resource = ? AND type = ? AND workspace = ?",
		policy.ResourceType,
		policy.Resource,
		policy.Type.String(),
		policy.Workspace,
	)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}

	s.policyCache.Remove(getPolicyCacheKey(policy.Workspace, policy.ResourceType, policy.Resource, policy.Type))
	if policy.Type == storepb.Policy_IAM {
		s.iamPolicyCache.Remove(getIamPolicyCacheKey(policy.Workspace, policy.ResourceType, policy.Resource))
	}
	return nil
}

func upsertPolicyImpl(ctx context.Context, txn *sql.Tx, create *PolicyMessage) (*PolicyMessage, error) {
	create.UpdatedAt = time.Now()

	q := qb.Q().Space(`
		INSERT INTO policy (
			workspace,
			resource_type,
			resource,
			inherit_from_parent,
			type,
			payload,
			enforce,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace, resource_type, resource, type) DO UPDATE SET
			inherit_from_parent = EXCLUDED.inherit_from_parent,
			payload = EXCLUDED.payload,
			enforce = EXCLUDED.enforce,
			updated_at = EXCLUDED.updated_at
	`,
		create.Workspace,
		create.ResourceType.String(),
		create.Resource,
		create.InheritFromParent,
		create.Type.String(),
		create.Payload,
		create.Enforce,
		create.UpdatedAt,
	)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := txn.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}
	return create, nil
}
