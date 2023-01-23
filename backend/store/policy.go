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

// policyRaw is the store model for an Policy.
// Fields have exactly the same meanings as Policy.
type policyRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ResourceType api.PolicyResourceType
	ResourceID   int

	// Domain specific fields
	InheritFromParent bool
	Type              api.PolicyType
	Payload           string
}

// toPolicy creates an instance of Policy based on the PolicyRaw.
// This is intended to be called when we need to compose an Policy relationship.
func (raw *policyRaw) toPolicy() *api.Policy {
	return &api.Policy{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ResourceType: raw.ResourceType,
		ResourceID:   raw.ResourceID,

		// Domain specific fields
		InheritFromParent: raw.InheritFromParent,
		Type:              raw.Type,
		Payload:           raw.Payload,
	}
}

// GetPolicy gets a policy.
func (s *Store) GetPolicy(ctx context.Context, find *api.PolicyFind) (*api.Policy, error) {
	policyRaw, err := s.getPolicyRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get policy with PolicyFind[%+v]", find)
	}
	policy, err := s.composePolicy(ctx, policyRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose policy with policyRaw[%+v]", policyRaw)
	}
	return policy, nil
}

// ListPolicy gets a list of policy by PolicyFind.
func (s *Store) ListPolicy(ctx context.Context, find *api.PolicyFind) ([]*api.Policy, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	policyRawList, err := findPolicyImpl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list policy with PolicyFind[%+v]", find)
	}

	policyList := []*api.Policy{}
	for _, raw := range policyRawList {
		policy, err := s.composePolicy(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose policy with policyRaw[%+v]", raw)
		}
		policyList = append(policyList, policy)
	}

	return policyList, nil
}

// GetBackupPlanPolicyByEnvID will get the backup plan policy for an environment.
func (s *Store) GetBackupPlanPolicyByEnvID(ctx context.Context, environmentID int) (*api.BackupPlanPolicy, error) {
	environmentResourceType := api.PolicyResourceTypeEnvironment
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		ResourceType: &environmentResourceType,
		ResourceID:   &environmentID,
		Type:         api.PolicyTypeBackupPlan,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalBackupPlanPolicy(policy.Payload)
}

// GetPipelineApprovalPolicy will get the pipeline approval policy for an environment.
func (s *Store) GetPipelineApprovalPolicy(ctx context.Context, environmentID int) (*api.PipelineApprovalPolicy, error) {
	environmentResourceType := api.PolicyResourceTypeEnvironment
	p, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		ResourceType: &environmentResourceType,
		ResourceID:   &environmentID,
		Type:         api.PolicyTypePipelineApproval,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalPipelineApprovalPolicy(p.Payload)
}

// GetNormalSQLReviewPolicy will get the normal SQL review policy for an environment.
func (s *Store) GetNormalSQLReviewPolicy(ctx context.Context, find *api.PolicyFind) (*advisor.SQLReviewPolicy, error) {
	if find.ID != nil && *find.ID == api.DefaultPolicyID {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy not found with ID %d", *find.ID)}
	}

	environmentResourceType := api.PolicyResourceTypeEnvironment
	find.ResourceType = &environmentResourceType
	find.Type = api.PolicyTypeSQLReview
	policy, err := s.getPolicyRaw(ctx, find)
	if err != nil {
		return nil, err
	}
	if policy.RowStatus == api.Archived {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy ID: %d for environment %d is archived", policy.ID, policy.ResourceID)}
	}
	if policy.ID == api.DefaultPolicyID {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("SQL review policy ID: %d for environment %d not found", policy.ID, policy.ResourceID)}
	}
	return api.UnmarshalSQLReviewPolicy(policy.Payload)
}

// GetSQLReviewPolicyIDByEnvID will get the SQL review policy ID for an environment.
func (s *Store) GetSQLReviewPolicyIDByEnvID(ctx context.Context, environmentID int) (int, error) {
	environmentResourceType := api.PolicyResourceTypeEnvironment
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		ResourceType: &environmentResourceType,
		ResourceID:   &environmentID,
		Type:         api.PolicyTypeSQLReview,
	})
	if err != nil {
		return 0, err
	}
	return policy.ID, nil
}

// GetSensitiveDataPolicy will get the sensitive data policy for database ID.
func (s *Store) GetSensitiveDataPolicy(ctx context.Context, databaseID int) (*api.SensitiveDataPolicy, error) {
	databaseResourceType := api.PolicyResourceTypeDatabase
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		ResourceType: &databaseResourceType,
		ResourceID:   &databaseID,
		Type:         api.PolicyTypeSensitiveData,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalSensitiveDataPolicy(policy.Payload)
}

// GetNormalAccessControlPolicy will get the normal access control polciy. Return nil if InheritFromParent is true.
func (s *Store) GetNormalAccessControlPolicy(ctx context.Context, resourceType api.PolicyResourceType, resourceID int) (*api.AccessControlPolicy, bool, error) {
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Type:         api.PolicyTypeAccessControl,
	})
	if err != nil {
		// For access constrol policy, the default value for InheritFromParent is true.
		return nil, true, err
	}
	if policy == nil || policy.RowStatus != api.Normal {
		// For access constrol policy, the default value for InheritFromParent is true.
		return nil, true, nil
	}

	accessControlPolicy, err := api.UnmarshalAccessControlPolicy(policy.Payload)
	if err != nil {
		// For access constrol policy, the default value for InheritFromParent is true.
		return nil, true, err
	}
	return accessControlPolicy, policy.InheritFromParent, nil
}

//
// private functions
//

func (s *Store) composePolicy(ctx context.Context, raw *policyRaw) (*api.Policy, error) {
	policy := raw.toPolicy()

	creator, err := s.GetPrincipalByID(ctx, policy.CreatorID)
	if err != nil {
		return nil, err
	}
	policy.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, policy.UpdaterID)
	if err != nil {
		return nil, err
	}
	policy.Updater = updater

	if policy.ResourceType == api.PolicyResourceTypeEnvironment {
		env, err := s.GetEnvironmentByID(ctx, policy.ResourceID)
		if err != nil {
			return nil, err
		}
		policy.Environment = env
	}

	return policy, nil
}

// getPolicyRaw finds the policy for an environment.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getPolicyRaw(ctx context.Context, find *api.PolicyFind) (*policyRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	policyRawList, err := findPolicyImpl(ctx, tx, find)
	var ret *policyRaw
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if len(policyRawList) == 0 {
		ret = &policyRaw{
			CreatorID: api.SystemBotID,
			UpdaterID: api.SystemBotID,
			Type:      find.Type,
		}
		if find.ResourceType != nil {
			ret.ResourceType = *find.ResourceType
		}
		if find.ResourceID != nil {
			ret.ResourceID = *find.ResourceID
		}
		if find.Type == api.PolicyTypeAccessControl {
			// For access constrol policy, the default value for InheritFromParent is true.
			ret.InheritFromParent = true
		}
	} else if len(policyRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d policy with filter %+v, expect 1. ", len(policyRawList), find)}
	} else {
		ret = policyRawList[0]
	}

	if ret.Payload == "" {
		// Return the default policy when there is no stored policy.
		payload, err := api.GetDefaultPolicy(find.Type)
		if err != nil {
			return nil, &common.Error{Code: common.Internal, Err: err}
		}
		ret.Payload = payload
		ret.ID = api.DefaultPolicyID
	}
	return ret, nil
}

func findPolicyImpl(ctx context.Context, tx *Tx, find *api.PolicyFind) ([]*policyRaw, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceType; v != nil {
		where, args = append(where, fmt.Sprintf("resource_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, find.Type)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			row_status,
			resource_type,
			resource_id,
			inherit_from_parent,
			type,
			payload
		FROM policy
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into policyRawList.
	var policyRawList []*policyRaw
	for rows.Next() {
		var policyRaw policyRaw
		if err := rows.Scan(
			&policyRaw.ID,
			&policyRaw.CreatorID,
			&policyRaw.CreatedTs,
			&policyRaw.UpdaterID,
			&policyRaw.UpdatedTs,
			&policyRaw.RowStatus,
			&policyRaw.ResourceType,
			&policyRaw.ResourceID,
			&policyRaw.InheritFromParent,
			&policyRaw.Type,
			&policyRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		policyRawList = append(policyRawList, &policyRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return policyRawList, nil
}

// PolicyMessage is the mssage for policy.
type PolicyMessage struct {
	ResourceUID       int
	ResourceType      api.PolicyResourceType
	Payload           string
	InheritFromParent bool
	Type              api.PolicyType

	// Output only.
	UID int
}

// FindPolicyMessage is the message for finding policies.
type FindPolicyMessage struct {
	ResourceType *api.PolicyResourceType
	ResourceUID  *int
	Type         *api.PolicyType
}

// UpdatePolicyMessage is the message for updating a policy.
type UpdatePolicyMessage struct {
	UpdaterID         int
	ResourceType      api.PolicyResourceType
	ResourceUID       int
	Type              api.PolicyType
	InheritFromParent *bool
	Payload           *string
}

// GetPolicyV2 gets a policy.
func (s *Store) GetPolicyV2(ctx context.Context, find *FindPolicyMessage) (*PolicyMessage, error) {
	if find.ResourceType != nil && find.ResourceUID != nil && find.Type != nil {
		if policy, ok := s.policyCache.Load(getPolicyCacheKey(*find.ResourceType, *find.ResourceUID, *find.Type)); ok {
			return policy.(*PolicyMessage), nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	policies, err := s.listPolicyImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}
	if len(policies) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d policies with filter %+v, expect 1", len(policies), find)}
	}
	policy := policies[0]

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.storePolicyIntoCache(policy)

	return policy, nil
}

// ListPoliciesV2 lists all policies.
func (s *Store) ListPoliciesV2(ctx context.Context, find *FindPolicyMessage) ([]*PolicyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	policies, err := s.listPolicyImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, policy := range policies {
		s.storePolicyIntoCache(policy)
	}

	return policies, nil
}

// CreatePolicyV2 creates a policy.
func (s *Store) CreatePolicyV2(ctx context.Context, create *PolicyMessage, creatorID int) (*PolicyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	policy, err := upsertPolicyV2Impl(ctx, tx, create, creatorID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.storePolicyIntoCache(policy)

	return policy, nil
}

// UpdatePolicyV2 updates the policy.
func (s *Store) UpdatePolicyV2(ctx context.Context, patch *UpdatePolicyMessage) (*PolicyMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.InheritFromParent; v != nil {
		set, args = append(set, fmt.Sprintf("inherit_from_parent = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ResourceType, patch.ResourceUID, patch.Type)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.storePolicyIntoCache(policy)

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
		return FormatError(err)
	}

	return tx.Commit()
}

func upsertPolicyV2Impl(ctx context.Context, tx *Tx, create *PolicyMessage, creatorID int) (*PolicyMessage, error) {
	var uid int
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO policy (
				creator_id,
				updater_id,
				resource_type,
				resource_id,
				inherit_from_parent,
				type,
				payload
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT(resource_type, resource_id, type) DO UPDATE SET
				payload = EXCLUDED.payload
			RETURNING id
		`,
		creatorID,
		creatorID,
		create.ResourceType,
		create.ResourceUID,
		create.InheritFromParent,
		create.Type,
		create.Payload,
	).Scan(
		&uid,
	); err != nil {
		return nil, FormatError(err)
	}
	create.UID = uid
	return create, nil
}

func (*Store) listPolicyImplV2(ctx context.Context, tx *Tx, find *FindPolicyMessage) ([]*PolicyMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.ResourceType; v != nil {
		where, args = append(where, fmt.Sprintf("resource_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceUID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_type,
			resource_id,
			inherit_from_parent,
			type,
			payload
		FROM policy
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var policyList []*PolicyMessage
	for rows.Next() {
		var policyMessage PolicyMessage
		if err := rows.Scan(
			&policyMessage.UID,
			&policyMessage.ResourceType,
			&policyMessage.ResourceUID,
			&policyMessage.InheritFromParent,
			&policyMessage.Type,
			&policyMessage.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		policyList = append(policyList, &policyMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return policyList, nil
}

func (s *Store) storePolicyIntoCache(policy *PolicyMessage) {
	if policy.Type != api.PolicyTypePipelineApproval {
		return
	}

	s.policyCache.Store(getPolicyCacheKey(policy.ResourceType, policy.ResourceUID, policy.Type), policy)
}
