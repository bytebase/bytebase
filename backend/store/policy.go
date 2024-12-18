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
	Member     string
	Roles      []string
	UpdaterUID int
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
	}, patch.UpdaterUID); err != nil {
		return nil, err
	}

	return s.GetWorkspaceIamPolicy(ctx)
}

func (s *Store) GetProjectIamPolicy(ctx context.Context, projectUID int) (*IamPolicyMessage, error) {
	resourceType := api.PolicyResourceTypeProject
	return s.getIamPolicy(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &projectUID,
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
		Etag:   generateEtag(policy.UpdatedTime),
	}, nil
}

func (s *Store) GetRolloutPolicy(ctx context.Context, environmentID int) (*storepb.RolloutPolicy, error) {
	resourceType := api.PolicyResourceTypeEnvironment
	pType := api.PolicyTypeRollout
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &environmentID,
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
	pType := api.PolicyTypeExportData
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
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
	resources := []DatabaseReviewConfig{
		&databaseReviewConfigResource{},
		&databaseProjectReviewConfigResource{},
		&databaseEnvironmentReviewConfigResource{},
	}

	for _, resource := range resources {
		resourceType := resource.GetResourceType()
		resourceUID, err := resource.GetResourceUID(ctx, s, database)
		if err != nil {
			slog.Debug("failed to resource id", slog.String("resource_type", string(resourceType)), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}

		reviewConfig, err := s.getReviewConfigByResource(ctx, resourceType, resourceUID)
		if err != nil {
			slog.Debug("failed to get review config", slog.String("resource_type", string(resourceType)), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		if reviewConfig == nil {
			slog.Debug("review config is empty", slog.String("resource_type", string(resourceType)), slog.String("database", database.DatabaseName), log.BBError(err))
			continue
		}
		return reviewConfig, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy for database %s not found", database.DatabaseName)}
}

func (s *Store) getReviewConfigByResource(ctx context.Context, resourceType api.PolicyResourceType, resourceUID int) (*storepb.ReviewConfigPayload, error) {
	pType := api.PolicyTypeTag

	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &resourceUID,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("tag policy for resource %v/%d not found", resourceType, resourceUID)}
	}
	if !policy.Enforce {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("tag policy is not enforced for resource %v/%d", resourceType, resourceUID)}
	}

	payload := &storepb.TagPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal tag policy payload")
	}

	reviewConfigName, ok := payload.Tags[string(api.ReservedTagReviewConfig)]
	if !ok {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config tag for resource %v/%d not found", resourceType, resourceUID)}
	}
	reviewConfigID, err := common.GetReviewConfigID(reviewConfigName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract review config uid from %s", reviewConfigName)
	}

	reviewConfig, err := s.GetReviewConfig(ctx, reviewConfigID)
	if err != nil {
		return nil, err
	}
	if reviewConfig == nil {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config for resource %v/%d not found", resourceType, resourceUID)}
	}
	if !reviewConfig.Enforce {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("review config is not enforced for resource %v/%d", resourceType, resourceUID)}
	}

	return reviewConfig.Payload, nil
}

// GetSlowQueryPolicy will get the slow query policy for instance ID.
func (s *Store) GetSlowQueryPolicy(ctx context.Context, resourceType api.PolicyResourceType, resourceID int) (*storepb.SlowQueryPolicy, error) {
	pType := api.PolicyTypeSlowQuery
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &resourceID,
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

// GetMaskingExceptionPolicyByProjectUID gets the masking exception policy for a project.
func (s *Store) GetMaskingExceptionPolicyByProjectUID(ctx context.Context, projectUID int) (*storepb.MaskingExceptionPolicy, error) {
	resourceType := api.PolicyResourceTypeProject
	pType := api.PolicyTypeMaskingException
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &projectUID,
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
	ResourceUID       int
	ResourceType      api.PolicyResourceType
	Payload           string
	InheritFromParent bool
	Type              api.PolicyType
	Enforce           bool

	// Output only.
	UID         int
	UpdatedTime time.Time
}

// FindPolicyMessage is the message for finding policies.
type FindPolicyMessage struct {
	ResourceType *api.PolicyResourceType
	ResourceUID  *int
	Type         *api.PolicyType
	ShowDeleted  bool
}

// UpdatePolicyMessage is the message for updating a policy.
type UpdatePolicyMessage struct {
	UpdaterID         int
	ResourceType      api.PolicyResourceType
	ResourceUID       int
	Type              api.PolicyType
	InheritFromParent *bool
	Payload           *string
	Enforce           *bool
}

// GetPolicyV2 gets a policy.
func (s *Store) GetPolicyV2(ctx context.Context, find *FindPolicyMessage) (*PolicyMessage, error) {
	if find.ResourceType != nil && find.ResourceUID != nil && find.Type != nil {
		if v, ok := s.policyCache.Get(getPolicyCacheKey(*find.ResourceType, *find.ResourceUID, *find.Type)); ok {
			return v, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true
	policies, err := s.listPolicyImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		// Cache the policy for not found as well to reduce the look up latency.
		if find.ResourceType != nil && find.ResourceUID != nil && find.Type != nil {
			s.policyCache.Add(getPolicyCacheKey(*find.ResourceType, *find.ResourceUID, *find.Type), nil)
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

	s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)

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
		s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)
	}

	return policies, nil
}

// CreatePolicyV2 creates a policy.
func (s *Store) CreatePolicyV2(ctx context.Context, create *PolicyMessage, creatorID int) (*PolicyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	policy, err := upsertPolicyV2Impl(ctx, tx, create, creatorID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)

	return policy, nil
}

// UpdatePolicyV2 updates the policy.
func (s *Store) UpdatePolicyV2(ctx context.Context, patch *UpdatePolicyMessage) (*PolicyMessage, error) {
	set, args := []string{"updater_id = $1", "updated_ts = $2"}, []any{patch.UpdaterID, time.Now().Unix()}
	if v := patch.InheritFromParent; v != nil {
		set, args = append(set, fmt.Sprintf("inherit_from_parent = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Enforce; v != nil {
		rowStatus := api.Normal
		if !*patch.Enforce {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	args = append(args, patch.ResourceType, patch.ResourceUID, patch.Type)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	policy := &PolicyMessage{
		ResourceUID:  patch.ResourceUID,
		ResourceType: patch.ResourceType,
		Type:         patch.Type,
	}
	var rowStatus string
	var updatedTs int64

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE policy
			SET `+strings.Join(set, ", ")+`
			WHERE resource_type = $%d AND resource_id = $%d AND type =$%d
			RETURNING
				payload,
				inherit_from_parent,
				row_status,
				updated_ts
		`, len(args)-2, len(args)-1, len(args)),
		args...,
	).Scan(
		&policy.Payload,
		&policy.InheritFromParent,
		&rowStatus,
		&updatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if rowStatus == string(api.Normal) {
		policy.Enforce = true
	}
	policy.UpdatedTime = time.Unix(updatedTs, 0)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)

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
		`DELETE FROM policy WHERE resource_type = $1 AND resource_id = $2 AND type = $3`,
		policy.ResourceType,
		policy.ResourceUID,
		policy.Type,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.policyCache.Remove(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type))
	return nil
}

func upsertPolicyV2Impl(ctx context.Context, tx *Tx, create *PolicyMessage, creatorID int) (*PolicyMessage, error) {
	var uid int
	var updatedTs int64

	rowStatus := api.Normal
	if !create.Enforce {
		rowStatus = api.Archived
	}
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO policy (
				creator_id,
				updater_id,
				resource_type,
				resource_id,
				inherit_from_parent,
				type,
				payload,
				row_status
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT(resource_type, resource_id, type) DO UPDATE SET
				inherit_from_parent = EXCLUDED.inherit_from_parent,
				payload = EXCLUDED.payload,
				updated_ts = extract(epoch from now()),
				row_status = EXCLUDED.row_status,
				updater_id = EXCLUDED.updater_id
			RETURNING
				id,
				updated_ts
		`,
		creatorID,
		creatorID,
		create.ResourceType,
		create.ResourceUID,
		create.InheritFromParent,
		create.Type,
		create.Payload,
		rowStatus,
	).Scan(
		&uid,
		&updatedTs,
	); err != nil {
		return nil, err
	}
	create.UID = uid
	create.UpdatedTime = time.Unix(updatedTs, 0)
	return create, nil
}

func (*Store) listPolicyImplV2(ctx context.Context, tx *Tx, find *FindPolicyMessage) ([]*PolicyMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceType; v != nil {
		where, args = append(where, fmt.Sprintf("resource_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceUID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			updated_ts,
			resource_type,
			resource_id,
			inherit_from_parent,
			type,
			payload,
			row_status
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
		var rowStatus api.RowStatus
		var updatedTs int64

		if err := rows.Scan(
			&policyMessage.UID,
			&updatedTs,
			&policyMessage.ResourceType,
			&policyMessage.ResourceUID,
			&policyMessage.InheritFromParent,
			&policyMessage.Type,
			&policyMessage.Payload,
			&rowStatus,
		); err != nil {
			return nil, err
		}
		if rowStatus == api.Normal {
			policyMessage.Enforce = true
		}
		policyMessage.UpdatedTime = time.Unix(updatedTs, 0)
		policyList = append(policyList, &policyMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return policyList, nil
}
