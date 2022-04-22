package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
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

// UpsertPolicy upserts an instance of Policy
func (s *Store) UpsertPolicy(ctx context.Context, upsert *api.PolicyUpsert) (*api.Policy, error) {
	policyRaw, err := s.upsertPolicyRaw(ctx, upsert)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert policy with PolicyUpsert[%+v], error[%w]", upsert, err)
	}
	policy, err := s.composePolicy(ctx, policyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose policy with policyRaw[%+v], error[%w]", policyRaw, err)
	}
	return policy, nil
}

// GetPolicy gets a policy
func (s *Store) GetPolicy(ctx context.Context, find *api.PolicyFind) (*api.Policy, error) {
	policyRaw, err := s.getPolicyRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy with PolicyFind[%+v], error[%w]", find, err)
	}
	policy, err := s.composePolicy(ctx, policyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose policy with policyRaw[%+v], error[%w]", policyRaw, err)
	}
	return policy, nil
}

// PatchPolicy patchs an instance of Policy by PolicyPatch.
func (s *Store) PatchPolicy(ctx context.Context, patch *api.PolicyPatch) (*api.Policy, error) {
	policyRaw, err := s.patchPolicyRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch policy with PolicyPatch[%+v], error[%w]", patch, err)
	}
	policy, err := s.composePolicy(ctx, policyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose policy with policyRaw[%+v], error[%w]", policyRaw, err)
	}
	return policy, nil
}

// DeletePolicy deletes an existing ARCHIVED policy by PolicyDelete.
func (s *Store) DeletePolicy(ctx context.Context, delete *api.PolicyDelete) error {
	// Validate policy.
	// Currently we only support PolicyTypeSchemaReview type policy to delete by id
	if delete.Type != api.PolicyTypeSchemaReview {
		return &common.Error{Code: common.Invalid, Err: fmt.Errorf("Invalid policy type")}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	find := &api.PolicyFind{
		ID:   &delete.ID,
		Type: &delete.Type,
	}
	policyRawList, err := findPolicyImpl(ctx, tx.PTx, find)
	if err != nil {
		return fmt.Errorf("Failed to list policy with PolicyFind[%+v], error[%w]", find, err)
	}
	if len(policyRawList) != 1 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("Failed to found policy with filter %+v, expect 1. ", find)}
	}
	policyRaw := policyRawList[0]
	if policyRaw.RowStatus != "ARCHIVED" {
		return &common.Error{Code: common.Invalid, Err: fmt.Errorf("Failed to delete policy with PolicyDelete[%+v], expect 'ARCHIVED' row_status", delete)}
	}

	if err := deletePolicyImpl(ctx, tx.PTx, delete); err != nil {
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
		return nil, fmt.Errorf("Failed to list policy with PolicyFind[%+v], error[%w]", find, err)
	}

	policyList := []*api.Policy{}
	for _, raw := range policyRawList {
		policy, err := s.composePolicy(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose policy with policyRaw[%+v], error[%w]", raw, err)
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
			CreatorID:     api.SystemBotID,
			UpdaterID:     api.SystemBotID,
			EnvironmentID: *find.EnvironmentID,
			Type:          *find.Type,
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
	if upsert.Type != "" {
		if err := api.ValidatePolicy(upsert.Type, upsert.Payload); err != nil {
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

// patchPolicyRaw sets a policy by id.
func (s *Store) patchPolicyRaw(ctx context.Context, patch *api.PolicyPatch) (*policyRaw, error) {
	// Validate policy.
	// Currently we only support PolicyTypeSchemaReview type policy to patch by id
	if patch.Type != api.PolicyTypeSchemaReview {
		return nil, &common.Error{Code: common.Invalid, Err: fmt.Errorf("Invalid policy type")}
	}
	if patch.Payload != nil {
		if *patch.Payload == "" {
			return nil, &common.Error{Code: common.Invalid, Err: fmt.Errorf("Invalid policy payload")}
		}
		if err := api.ValidatePolicy(patch.Type, *patch.Payload); err != nil {
			return nil, &common.Error{Code: common.Invalid, Err: err}
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	policy, err := patchPolicyImpl(ctx, tx.PTx, patch)
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
	if upsert.Payload == "" {
		upsert.Payload = "{}"
	}
	row, err := tx.QueryContext(ctx, `
		INSERT INTO policy (
			creator_id,
			updater_id,
			environment_id,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(environment_id, type) DO UPDATE SET
			payload = excluded.payload
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, row_status, environment_id, type, payload
		`,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.EnvironmentID,
		upsert.Type,
		upsert.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var policyRaw policyRaw
	if err := row.Scan(
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

	return &policyRaw, nil
}

// patchPolicyImpl updates an existing policy by id and type.
func patchPolicyImpl(ctx context.Context, tx *sql.Tx, patch *api.PolicyPatch) (*policyRaw, error) {
	set, args := []string{}, []interface{}{}

	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EnvironmentID; v != nil {
		set, args = append(set, fmt.Sprintf("environment_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}

	where := []string{}
	where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, patch.ID)
	where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, patch.Type)

	// Update the policy.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE policy
		SET %s
		WHERE %s
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, row_status, environment_id, type, payload
	`, strings.Join(set, ", "), strings.Join(where, " AND ")),
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var policyRaw policyRaw
	if err := row.Scan(
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

	return &policyRaw, nil
}

// deletePolicyImpl deletes an existing ARCHIVED policy by id and type.
func deletePolicyImpl(ctx context.Context, tx *sql.Tx, delete *api.PolicyDelete) error {
	// Remove row from policy.
	if _, err := tx.ExecContext(ctx, `DELETE FROM policy WHERE id = $1 AND type = $2 AND row_status = $3`, delete.ID, delete.Type, "ARCHIVED"); err != nil {
		return FormatError(err)
	}
	return nil
}
