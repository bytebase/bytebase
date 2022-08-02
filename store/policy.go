package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
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
	EnvironmentID int

	// Domain specific fields
	Type    api.PolicyType
	Payload string
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
		EnvironmentID: raw.EnvironmentID,

		// Domain specific fields
		Type:    raw.Type,
		Payload: raw.Payload,
	}
}

// UpsertPolicy upserts an instance of Policy.
func (s *Store) UpsertPolicy(ctx context.Context, upsert *api.PolicyUpsert) (*api.Policy, error) {
	policyRaw, err := s.upsertPolicyRaw(ctx, upsert)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert policy with PolicyUpsert[%+v], error: %w", upsert, err)
	}
	policy, err := s.composePolicy(ctx, policyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose policy with policyRaw[%+v], error: %w", policyRaw, err)
	}
	return policy, nil
}

// GetPolicy gets a policy.
func (s *Store) GetPolicy(ctx context.Context, find *api.PolicyFind) (*api.Policy, error) {
	policyRaw, err := s.getPolicyRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy with PolicyFind[%+v], error: %w", find, err)
	}
	policy, err := s.composePolicy(ctx, policyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose policy with policyRaw[%+v], error: %w", policyRaw, err)
	}
	return policy, nil
}

// DeletePolicy deletes an existing ARCHIVED policy by PolicyDelete.
func (s *Store) DeletePolicy(ctx context.Context, delete *api.PolicyDelete) error {
	// Validate policy.
	// Currently we only support PolicyTypeSQLReview type policy to delete by id
	if delete.Type != api.PolicyTypeSQLReview {
		return &common.Error{Code: common.Invalid, Err: fmt.Errorf("invalid policy type")}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	find := &api.PolicyFind{
		EnvironmentID: &delete.EnvironmentID,
		Type:          &delete.Type,
	}
	policyRawList, err := findPolicyImpl(ctx, tx.PTx, find)
	if err != nil {
		return fmt.Errorf("failed to list policy with PolicyFind[%+v], error: %w", find, err)
	}
	if len(policyRawList) != 1 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("failed to found policy with filter %+v, expect 1. ", find)}
	}
	policyRaw := policyRawList[0]
	if policyRaw.RowStatus != api.Archived {
		return &common.Error{Code: common.Invalid, Err: fmt.Errorf("failed to delete policy with PolicyDelete[%+v], expect 'ARCHIVED' row_status", delete)}
	}

	if err := s.deletePolicyImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// ListPolicy gets a list of policy by PolicyFind.
func (s *Store) ListPolicy(ctx context.Context, find *api.PolicyFind) ([]*api.Policy, error) {
	// Validate policy type existence.
	if find.Type != nil && *find.Type != "" {
		if err := api.ValidatePolicy(*find.Type, ""); err != nil {
			return nil, &common.Error{Code: common.Invalid, Err: err}
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	policyRawList, err := findPolicyImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list policy with PolicyFind[%+v], error: %w", find, err)
	}

	policyList := []*api.Policy{}
	for _, raw := range policyRawList {
		policy, err := s.composePolicy(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose policy with policyRaw[%+v], error: %w", raw, err)
		}
		policyList = append(policyList, policy)
	}

	return policyList, nil
}

// GetBackupPlanPolicyByEnvID will get the backup plan policy for an environment.
func (s *Store) GetBackupPlanPolicyByEnvID(ctx context.Context, environmentID int) (*api.BackupPlanPolicy, error) {
	pType := api.PolicyTypeBackupPlan
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		EnvironmentID: &environmentID,
		Type:          &pType,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalBackupPlanPolicy(policy.Payload)
}

// GetPipelineApprovalPolicy will get the pipeline approval policy for an environment.
func (s *Store) GetPipelineApprovalPolicy(ctx context.Context, environmentID int) (*api.PipelineApprovalPolicy, error) {
	pType := api.PolicyTypePipelineApproval
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		EnvironmentID: &environmentID,
		Type:          &pType,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalPipelineApprovalPolicy(policy.Payload)
}

// GetNormalSQLReviewPolicy will get the normal SQL review policy for an environment.
func (s *Store) GetNormalSQLReviewPolicy(ctx context.Context, find *api.PolicyFind) (*advisor.SQLReviewPolicy, error) {
	if find.ID != nil && *find.ID == api.DefaultPolicyID {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("SQL review policy not found with ID %d", *find.ID)}
	}

	pType := api.PolicyTypeSQLReview
	find.Type = &pType
	policy, err := s.getPolicyRaw(ctx, find)
	if err != nil {
		return nil, err
	}
	if policy.RowStatus == api.Archived {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("SQL review policy ID: %d for environment %d is archived", policy.ID, policy.EnvironmentID)}
	}
	if policy.ID == api.DefaultPolicyID {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("SQL review policy ID: %d for environment %d not found", policy.ID, policy.EnvironmentID)}
	}
	return api.UnmarshalSQLReviewPolicy(policy.Payload)
}

// GetSQLReviewPolicyIDByEnvID will get the SQL review policy ID for an environment.
func (s *Store) GetSQLReviewPolicyIDByEnvID(ctx context.Context, environmentID int) (int, error) {
	pType := api.PolicyTypeSQLReview
	policy, err := s.getPolicyRaw(ctx, &api.PolicyFind{
		EnvironmentID: &environmentID,
		Type:          &pType,
	})
	if err != nil {
		return 0, err
	}
	return policy.ID, nil
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

	env, err := s.GetEnvironmentByID(ctx, policy.EnvironmentID)
	if err != nil {
		return nil, err
	}
	policy.Environment = env

	return policy, nil
}

// getPolicyRaw finds the policy for an environment.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getPolicyRaw(ctx context.Context, find *api.PolicyFind) (*policyRaw, error) {
	// Validate policy type existence.
	if find.Type != nil && *find.Type != "" {
		if err := api.ValidatePolicy(*find.Type, ""); err != nil {
			return nil, &common.Error{Code: common.Invalid, Err: err}
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	policyRawList, err := findPolicyImpl(ctx, tx.PTx, find)
	var ret *policyRaw
	if err != nil {
		return nil, err
	}

	if len(policyRawList) == 0 {
		ret = &policyRaw{
			CreatorID: api.SystemBotID,
			UpdaterID: api.SystemBotID,
			Type:      *find.Type,
		}
		if find.EnvironmentID != nil {
			ret.EnvironmentID = *find.EnvironmentID
		}
	} else if len(policyRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d policy with filter %+v, expect 1. ", len(policyRawList), find)}
	} else {
		ret = policyRawList[0]
	}

	if ret.Payload == "" {
		// Return the default policy when there is no stored policy.
		payload, err := api.GetDefaultPolicy(*find.Type)
		if err != nil {
			return nil, &common.Error{Code: common.Internal, Err: err}
		}
		ret.Payload = payload
		ret.ID = api.DefaultPolicyID
	}
	return ret, nil
}

func findPolicyImpl(ctx context.Context, tx *sql.Tx, find *api.PolicyFind) ([]*policyRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			row_status,
			environment_id,
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
			&policyRaw.EnvironmentID,
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

// upsertPolicyRaw sets a policy for an environment.
func (s *Store) upsertPolicyRaw(ctx context.Context, upsert *api.PolicyUpsert) (*policyRaw, error) {
	// Validate policy.
	if upsert.Type != "" && upsert.Payload != nil {
		if err := api.ValidatePolicy(upsert.Type, *upsert.Payload); err != nil {
			return nil, &common.Error{Code: common.Invalid, Err: err}
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	policy, err := upsertPolicyImpl(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return policy, nil
}

// upsertPolicyImpl updates an existing policy by environment id and type.
func upsertPolicyImpl(ctx context.Context, tx *sql.Tx, upsert *api.PolicyUpsert) (*policyRaw, error) {
	// Upsert row into policy.
	var set []string
	if v := upsert.Payload; v != nil {
		set = append(set, "payload = EXCLUDED.payload")
	}
	if v := upsert.RowStatus; v != nil {
		set = append(set, "row_status = EXCLUDED.row_status")
	}

	if len(set) == 0 {
		return nil, &common.Error{Code: common.Invalid, Err: fmt.Errorf("invalid policy upsert %+v", upsert)}
	}

	if upsert.Payload == nil || *upsert.Payload == "" {
		emptyPayload := "{}"
		upsert.Payload = &emptyPayload
	}
	if upsert.RowStatus == nil {
		rowStatus := string(api.Normal)
		upsert.RowStatus = &rowStatus
	}

	query := fmt.Sprintf(`
		INSERT INTO policy (
			creator_id,
			updater_id,
			environment_id,
			type,
			payload,
			row_status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(environment_id, type) DO UPDATE SET
			%s
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, row_status, environment_id, type, payload
	`, strings.Join(set, ","))
	var policyRaw policyRaw
	if err := tx.QueryRowContext(ctx, query,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.EnvironmentID,
		upsert.Type,
		upsert.Payload,
		upsert.RowStatus,
	).Scan(
		&policyRaw.ID,
		&policyRaw.CreatorID,
		&policyRaw.CreatedTs,
		&policyRaw.UpdaterID,
		&policyRaw.UpdatedTs,
		&policyRaw.RowStatus,
		&policyRaw.EnvironmentID,
		&policyRaw.Type,
		&policyRaw.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &policyRaw, nil
}

// deletePolicyImpl deletes an existing ARCHIVED policy by id and type.
func (*Store) deletePolicyImpl(ctx context.Context, tx *sql.Tx, delete *api.PolicyDelete) error {
	// Remove row from policy.
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM policy
			WHERE environment_id = $1 AND type = $2 AND row_status = $3
		`,
		delete.EnvironmentID,
		delete.Type,
		api.Archived,
	); err != nil {
		return FormatError(err)
	}
	return nil
}
