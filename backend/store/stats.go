package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

// CountActiveInstances counts the number of instances.
func (s *Store) CountActiveInstances(ctx context.Context, workspaceID string) (int, error) {
	q := qb.Q().Space("SELECT count(1) FROM instance WHERE instance.workspace = ?", workspaceID).And("instance.deleted = ?", false)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountAllActivePrincipals counts all active end users globally (cross-workspace).
func (s *Store) CountAllActivePrincipals(ctx context.Context) (int, error) {
	var count int
	if err := s.GetDB().QueryRowContext(ctx,
		`SELECT count(*) FROM principal WHERE deleted = FALSE`,
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}


// CountActivePrincipals counts non-deleted principals. Used for display purposes
// (e.g. actuator info) — not for seat limit enforcement.
func (s *Store) CountActivePrincipals(ctx context.Context) (int, error) {
	q := qb.Q().Space(`SELECT count(*) FROM principal WHERE deleted = FALSE`)
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}
	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
