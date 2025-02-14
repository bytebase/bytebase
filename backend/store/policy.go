package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type IamPolicyMessage struct {
	Policy *storepb.IamPolicy
	Etag   string
}

// generateEtag generates etag for the given body.
func generateEtag(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixMilli())
}

func (s *Store) GetWorkspaceIamPolicy(ctx context.Context) (*IamPolicyMessage, error) {
	resourceType := api.PolicyResourceTypeWorkspace
	return s.getIamPolicy(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
	})
}

type PatchIamPolicyMessage struct {
	Member string
	Roles  []string
}

// PatchWorkspaceIamPolicy will set or remove the member for the workspace role.
func (s *Store) PatchWorkspaceIamPolicy(ctx context.Context, patch *PatchIamPolicyMessage) (*IamPolicyMessage, error) {
	workspaceIamPolicy, err := s.GetWorkspaceIamPolicy(ctx)
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

	if _, err := s.CreatePolicyV2(ctx, &PolicyMessage{
		ResourceType:      api.PolicyResourceTypeWorkspace,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeIAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}); err != nil {
		return nil, err
	}

	return s.GetWorkspaceIamPolicy(ctx)
}

func (s *Store) GetProjectIamPolicy(ctx context.Context, projectID string) (*IamPolicyMessage, error) {
	resourceType := api.PolicyResourceTypeProject
	resource := common.FormatProject(projectID)
	return s.getIamPolicy(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &resource,
	})
}

func (s *Store) getIamPolicy(ctx context.Context, find *FindPolicyMessage) (*IamPolicyMessage, error) {
	pType := api.PolicyTypeIAM
	find.Type = &pType
	policy, err := s.GetPolicyV2(ctx, find)
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
		return nil, errors.Wrapf(err, "failed to unmarshal iam policy")
	}

	return &IamPolicyMessage{
		Policy: p,
		Etag:   generateEtag(policy.UpdatedAt),
	}, nil
}

func (s *Store) GetRolloutPolicy(ctx context.Context, environment string) (*storepb.RolloutPolicy, error) {
	resource := common.FormatEnvironment(environment)
	resourceType := api.PolicyResourceTypeEnvironment
	pType := api.PolicyTypeRollout
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         &pType,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy")
	}
	if policy == nil {
		return &storepb.RolloutPolicy{
			Automatic: true,
		}, nil
	}

	p := &storepb.RolloutPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal rollout policy")
	}

	return p, nil
}

func (s *Store) GetDataExportPolicy(ctx context.Context) (*storepb.ExportDataPolicy, error) {
	resourceType := api.PolicyResourceTypeWorkspace
	resource := ""
	pType := api.PolicyTypeExportData
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         &pType,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy")
	}
	if policy == nil {
		return &storepb.ExportDataPolicy{
			Disable: false,
		}, nil
	}

	p := &storepb.ExportDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal data export policy")
	}

	return p, nil
}

// GetReviewConfigForDatabase will get the review config for a database.
func (s *Store) GetReviewConfigForDatabase(ctx context.Context, database *DatabaseMessage) (*storepb.ReviewConfigPayload, error) {
	for _, v := range []struct {
		resourceType api.PolicyResourceType
		resource     string
	}{
		{
			resourceType: api.PolicyResourceTypeEnvironment,
			resource:     common.FormatEnvironment(database.EffectiveEnvironmentID),
		},
		{
			resourceType: api.PolicyResourceTypeProject,
			resource:     common.FormatProject(database.ProjectID),
		},
	} {
		reviewConfig, err := s.getReviewConfigByResource(ctx, v.resourceType, v.resource)
		if err != nil {
			slog.Debug("failed to get review config", slog.String("resource_type", string(v.resourceType)), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		if reviewConfig == nil {
			slog.Debug("review config is empty", slog.String("resource_type", string(v.resourceType)), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		return reviewConfig, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy for database %s not found", database.DatabaseName)}
}

func (s *Store) getReviewConfigByResource(ctx context.Context, resourceType api.PolicyResourceType, resource string) (*storepb.ReviewConfigPayload, error) {
	pType := api.PolicyTypeTag

	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         &pType,
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

	reviewConfigName, ok := payload.Tags[string(api.ReservedTagReviewConfig)]
	if !ok {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config tag for resource %v/%s not found", resourceType, resource)}
	}
	reviewConfigID, err := common.GetReviewConfigID(reviewConfigName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract review config %s", reviewConfigName)
	}

	reviewConfig, err := s.GetReviewConfig(ctx, reviewConfigID)
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

// GetSlowQueryPolicy will get the slow query policy for instance ID.
func (s *Store) GetSlowQueryPolicy(ctx context.Context, instanceID string) (*storepb.SlowQueryPolicy, error) {
	resourceType := api.PolicyResourceTypeInstance
	resource := common.FormatInstance(instanceID)
	pType := api.PolicyTypeSlowQuery
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}

	if policy == nil {
		return &storepb.SlowQueryPolicy{Active: false}, nil
	}

	payload := &storepb.SlowQueryPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal slow query policy payload")
	}

	return payload, nil
}

// GetMaskingRulePolicy will get the masking rule policy.
func (s *Store) GetMaskingRulePolicy(ctx context.Context) (*storepb.MaskingRulePolicy, error) {
	pType := api.PolicyTypeMaskingRule
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		Type: &pType,
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

// GetMaskingExceptionPolicyByProject gets the masking exception policy for a project.
func (s *Store) GetMaskingExceptionPolicyByProject(ctx context.Context, projectID string) (*storepb.MaskingExceptionPolicy, error) {
	resourceType := api.PolicyResourceTypeProject
	resource := common.FormatProject(projectID)
	pType := api.PolicyTypeMaskingException
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     &resource,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}

	if policy == nil {
		return &storepb.MaskingExceptionPolicy{}, nil
	}

	p := new(storepb.MaskingExceptionPolicy)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
		return nil, err
	}

	return p, nil
}

// PolicyMessage is the mssage for policy.
type PolicyMessage struct {
	Resource          string
	ResourceType      api.PolicyResourceType
	Payload           string
	InheritFromParent bool
	Type              api.PolicyType
	Enforce           bool

	UpdatedAt time.Time
}

// FindPolicyMessage is the message for finding policies.
type FindPolicyMessage struct {
	ResourceType *api.PolicyResourceType
	Resource     *string
	Type         *api.PolicyType
	// ShowAll will show all policies regardless of the enforce status.
	ShowAll bool
}

// UpdatePolicyMessage is the message for updating a policy.
type UpdatePolicyMessage struct {
	ResourceType      api.PolicyResourceType
	Resource          string
	Type              api.PolicyType
	InheritFromParent *bool
	Payload           *string
	Enforce           *bool
}

// GetPolicyV2 gets a policy.
func (s *Store) GetPolicyV2(ctx context.Context, find *FindPolicyMessage) (*PolicyMessage, error) {
	if find.ResourceType != nil && find.Resource != nil && find.Type != nil {
		if v, ok := s.policyCache.Get(getPolicyCacheKey(*find.ResourceType, *find.Resource, *find.Type)); ok {
			return v, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// We will always return the resource regardless of its deleted state.
	find.ShowAll = true
	policies, err := s.listPolicyImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		// Cache the policy for not found as well to reduce the look up latency.
		if find.ResourceType != nil && find.Resource != nil && find.Type != nil {
			s.policyCache.Add(getPolicyCacheKey(*find.ResourceType, *find.Resource, *find.Type), nil)
		}
		return nil, nil
	}
	if len(policies) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d policies with filter %+v, expect 1", len(policies), find)}
	}
	policy := policies[0]

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.Resource, policy.Type), policy)

	return policy, nil
}

// ListPoliciesV2 lists all policies.
func (s *Store) ListPoliciesV2(ctx context.Context, find *FindPolicyMessage) ([]*PolicyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	policies, err := s.listPolicyImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, policy := range policies {
		s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.Resource, policy.Type), policy)
	}

	return policies, nil
}

// CreatePolicyV2 creates a policy.
func (s *Store) CreatePolicyV2(ctx context.Context, create *PolicyMessage) (*PolicyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	policy, err := upsertPolicyV2Impl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.Resource, policy.Type), policy)

	return policy, nil
}

// UpdatePolicyV2 updates the policy.
func (s *Store) UpdatePolicyV2(ctx context.Context, patch *UpdatePolicyMessage) (*PolicyMessage, error) {
	set, args := []string{"updated_at = $1"}, []any{time.Now()}
	if v := patch.InheritFromParent; v != nil {
		set, args = append(set, fmt.Sprintf("inherit_from_parent = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Enforce; v != nil {
		set, args = append(set, fmt.Sprintf(`enforce = $%d`, len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ResourceType, patch.Resource, patch.Type)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	policy := &PolicyMessage{
		Resource:     patch.Resource,
		ResourceType: patch.ResourceType,
		Type:         patch.Type,
	}

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE policy
			SET `+strings.Join(set, ", ")+`
			WHERE resource_type = $%d AND resource = $%d AND type =$%d
			RETURNING
				payload,
				inherit_from_parent,
				enforce,
				updated_at
		`, len(args)-2, len(args)-1, len(args)),
		args...,
	).Scan(
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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.Resource, policy.Type), policy)

	return policy, nil
}

// DeletePolicyV2 deletes the policy.
func (s *Store) DeletePolicyV2(ctx context.Context, policy *PolicyMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM policy WHERE resource_type = $1 AND resource = $2 AND type = $3`,
		policy.ResourceType,
		policy.Resource,
		policy.Type,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.policyCache.Remove(getPolicyCacheKey(policy.ResourceType, policy.Resource, policy.Type))
	return nil
}

func upsertPolicyV2Impl(ctx context.Context, tx *Tx, create *PolicyMessage) (*PolicyMessage, error) {
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO policy (
				resource_type,
				resource,
				inherit_from_parent,
				type,
				payload,
				enforce
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT(resource_type, resource, type) DO UPDATE SET
				inherit_from_parent = EXCLUDED.inherit_from_parent,
				payload = EXCLUDED.payload,
				enforce = EXCLUDED.enforce
			RETURNING updated_at
		`,
		create.ResourceType,
		create.Resource,
		create.InheritFromParent,
		create.Type,
		create.Payload,
		create.Enforce,
	).Scan(
		&create.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return create, nil
}

func (*Store) listPolicyImplV2(ctx context.Context, tx *Tx, find *FindPolicyMessage) ([]*PolicyMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceType; v != nil {
		where, args = append(where, fmt.Sprintf("resource_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Resource; v != nil {
		where, args = append(where, fmt.Sprintf("resource = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowAll {
		where, args = append(where, fmt.Sprintf("enforce = $%d", len(args)+1)), append(args, true)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			updated_at,
			resource_type,
			resource,
			inherit_from_parent,
			type,
			payload,
			enforce
		FROM policy
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policyList []*PolicyMessage
	for rows.Next() {
		var policyMessage PolicyMessage
		if err := rows.Scan(
			&policyMessage.UpdatedAt,
			&policyMessage.ResourceType,
			&policyMessage.Resource,
			&policyMessage.InheritFromParent,
			&policyMessage.Type,
			&policyMessage.Payload,
			&policyMessage.Enforce,
		); err != nil {
			return nil, err
		}
		policyList = append(policyList, &policyMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return policyList, nil
}
