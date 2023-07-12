package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

// GetBackupPlanPolicyByEnvID will get the backup plan policy for an environment.
func (s *Store) GetBackupPlanPolicyByEnvID(ctx context.Context, environmentID int) (*api.BackupPlanPolicy, error) {
	resourceType := api.PolicyResourceTypeEnvironment
	pType := api.PolicyTypeBackupPlan
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &environmentID,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return &api.BackupPlanPolicy{
			Schedule: api.BackupPlanPolicyScheduleUnset,
		}, nil
	}
	return api.UnmarshalBackupPlanPolicy(policy.Payload)
}

// GetPipelineApprovalPolicy will get the pipeline approval policy for an environment.
func (s *Store) GetPipelineApprovalPolicy(ctx context.Context, environmentID int) (*api.PipelineApprovalPolicy, error) {
	resourceType := api.PolicyResourceTypeEnvironment
	pType := api.PolicyTypePipelineApproval
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &environmentID,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return &api.PipelineApprovalPolicy{
			Value: api.PipelineApprovalValueManualAlways,
		}, nil
	}

	return api.UnmarshalPipelineApprovalPolicy(policy.Payload)
}

// GetSQLReviewPolicy will get the SQL review policy for an environment.
func (s *Store) GetSQLReviewPolicy(ctx context.Context, environmentID int) (*advisor.SQLReviewPolicy, error) {
	resourceType := api.PolicyResourceTypeEnvironment
	pType := api.PolicyTypeSQLReview
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &environmentID,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy for environment %d not found", environmentID)}
	}
	if !policy.Enforce {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy is not enforced for environment %d", environmentID)}
	}
	return api.UnmarshalSQLReviewPolicy(policy.Payload)
}

// GetSensitiveDataPolicy will get the sensitive data policy for database ID.
func (s *Store) GetSensitiveDataPolicy(ctx context.Context, databaseID int) (*api.SensitiveDataPolicy, error) {
	resourceType := api.PolicyResourceTypeDatabase
	pType := api.PolicyTypeSensitiveData
	policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &databaseID,
		Type:         &pType,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return &api.SensitiveDataPolicy{}, nil
	}

	return api.UnmarshalSensitiveDataPolicy(policy.Payload)
}

// GetSlowQueryPolicy will get the slow query policy for instance ID.
func (s *Store) GetSlowQueryPolicy(ctx context.Context, resourceType api.PolicyResourceType, resourceID int) (*api.SlowQueryPolicy, error) {
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
		return &api.SlowQueryPolicy{Active: false}, nil
	}

	return api.UnmarshalSlowQueryPolicy(policy.Payload)
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
	UID int
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
	Delete            *bool
}

// GetPolicyV2 gets a policy.
func (s *Store) GetPolicyV2(ctx context.Context, find *FindPolicyMessage) (*PolicyMessage, error) {
	if find.ResourceType != nil && find.ResourceUID != nil && find.Type != nil {
		if policy, ok := s.policyCache.Load(getPolicyCacheKey(*find.ResourceType, *find.ResourceUID, *find.Type)); ok {
			if policy == nil {
				return nil, nil
			}
			return policy.(*PolicyMessage), nil
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
			s.policyCache.Store(getPolicyCacheKey(*find.ResourceType, *find.ResourceUID, *find.Type), nil)
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

	s.policyCache.Store(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)

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
		s.policyCache.Store(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)
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

	s.policyCache.Store(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)

	return policy, nil
}

// UpdatePolicyV2 updates the policy.
func (s *Store) UpdatePolicyV2(ctx context.Context, patch *UpdatePolicyMessage) (*PolicyMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", patch.UpdaterID)}
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
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE policy
			SET `+strings.Join(set, ", ")+`
			WHERE resource_type = $%d AND resource_id = $%d AND type =$%d
			RETURNING
				payload,
				inherit_from_parent,
				row_status
		`, len(args)-2, len(args)-1, len(args)),
		args...,
	).Scan(
		&policy.Payload,
		&policy.InheritFromParent,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if rowStatus == string(api.Normal) {
		policy.Enforce = true
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Store(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)

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

	s.policyCache.Delete(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type))
	return nil
}

func upsertPolicyV2Impl(ctx context.Context, tx *Tx, create *PolicyMessage, creatorID int) (*PolicyMessage, error) {
	var uid int
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
				row_status = EXCLUDED.row_status
			RETURNING id
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
	); err != nil {
		return nil, err
	}
	create.UID = uid
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
		if err := rows.Scan(
			&policyMessage.UID,
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
		policyList = append(policyList, &policyMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return policyList, nil
}
