package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.PolicyService = (*PolicyService)(nil)
)

// PolicyService represents a service for managing environment based policies.
type PolicyService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewPolicyService returns a new instance of PolicyService.
func NewPolicyService(logger *zap.Logger, db *DB, cache api.CacheService) *PolicyService {
	return &PolicyService{l: logger, db: db, cache: cache}
}

// FindPolicy finds the policy for an environment.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *PolicyService) FindPolicy(ctx context.Context, find *api.PolicyFind) (*api.Policy, error) {
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
	defer tx.Rollback()

	list, err := s.findPolicy(ctx, tx, find)
	var ret *api.Policy
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		ret = &api.Policy{
			CreatorID:     api.SystemBotID,
			UpdaterID:     api.SystemBotID,
			EnvironmentID: *find.EnvironmentID,
			Type:          *find.Type,
		}
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d policy with filter %+v, expect 1. ", len(list), find)}
	} else {
		ret = list[0]
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

func (s *PolicyService) findPolicy(ctx context.Context, tx *Tx, find *api.PolicyFind) (_ []*api.Policy, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, "environment_id = ?"), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, "type = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Policy, 0)
	for rows.Next() {
		var policy api.Policy
		if err := rows.Scan(
			&policy.ID,
			&policy.CreatorID,
			&policy.CreatedTs,
			&policy.UpdaterID,
			&policy.UpdatedTs,
			&policy.EnvironmentID,
			&policy.Type,
			&policy.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &policy)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// UpsertPolicy sets a policy for an environment.
func (s *PolicyService) UpsertPolicy(ctx context.Context, upsert *api.PolicyUpsert) (*api.Policy, error) {
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
	defer tx.Rollback()

	policy, err := s.upsertPolicy(ctx, tx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return policy, nil
}

// upsertPolicy updates an existing policy.
func (s *PolicyService) upsertPolicy(ctx context.Context, tx *Tx, upsert *api.PolicyUpsert) (*api.Policy, error) {
	// Upsert row into policy.
	// TODO(spinningbot): fix the query.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO policy (
			creator_id,
			updater_id,
			environment_id,
			type,
			payload
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(environment_id, type) DO UPDATE SET
				payload = excluded.payload
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, environment_id, type, payload
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
	var policy api.Policy
	if err := row.Scan(
		&policy.ID,
		&policy.CreatorID,
		&policy.CreatedTs,
		&policy.UpdaterID,
		&policy.UpdatedTs,
		&policy.EnvironmentID,
		&policy.Type,
		&policy.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &policy, nil
}

// GetBackupPlanPolicy will get the backup plan policy for an environment.
func (s *PolicyService) GetBackupPlanPolicy(ctx context.Context, environmentID int) (*api.BackupPlanPolicy, error) {
	pType := api.PolicyTypeBackupPlan
	policy, err := s.FindPolicy(ctx, &api.PolicyFind{
		EnvironmentID: &environmentID,
		Type:          &pType,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalBackupPlanPolicy(policy.Payload)
}

// GetPipelineApprovalPolicy will get the pipeline approval policy for an environment.
func (s *PolicyService) GetPipelineApprovalPolicy(ctx context.Context, environmentID int) (*api.PipelineApprovalPolicy, error) {
	pType := api.PolicyTypePipelineApproval
	policy, err := s.FindPolicy(ctx, &api.PolicyFind{
		EnvironmentID: &environmentID,
		Type:          &pType,
	})
	if err != nil {
		return nil, err
	}
	return api.UnmarshalPipelineApprovalPolicy(policy.Payload)
}
