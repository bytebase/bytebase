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
	_ api.EnvironmentService = (*EnvironmentService)(nil)
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
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *PolicyService) FindPolicy(ctx context.Context, find *api.PolicyFind) (*api.Policy, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findPolicy(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &common.Error{Code: common.ENOTFOUND, Message: fmt.Sprintf("policy not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.ECONFLICT, Message: fmt.Sprintf("found %d policy with filter %+v, expect 1. ", len(list), find)}
	}

	return list[0], nil
}

func (s *PolicyService) findPolicy(ctx context.Context, tx *Tx, find *api.PolicyFind) (_ []*api.Policy, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.EnvironmentId; v != nil {
		where, args = append(where, "environment_id = ?"), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			environment_id,
			name,
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
			&policy.CreatorId,
			&policy.CreatedTs,
			&policy.UpdaterId,
			&policy.UpdatedTs,
			&policy.Environment,
			&policy.Name,
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
			name,
			payload
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(environment_id, name) DO UPDATE SET
				payload = excluded.payload
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, environment_id, name, payload
		`,
		upsert.UpdaterId,
		upsert.UpdaterId,
		upsert.EnvironmentId,
		upsert.Name,
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
		&policy.CreatorId,
		&policy.CreatedTs,
		&policy.UpdaterId,
		&policy.UpdatedTs,
		&policy.EnvironmentId,
		&policy.Name,
		&policy.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &policy, nil
}
