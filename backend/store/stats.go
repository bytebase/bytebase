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

// CountActiveEndUsersPerWorkspace counts active end users who are members of the given workspace
// by joining principal with workspace IAM policy bindings.
func (s *Store) CountActiveEndUsersPerWorkspace(ctx context.Context, workspaceID string) (int, error) {
	q := qb.Q().Space(`
		SELECT count(DISTINCT principal.id)
		FROM principal
		JOIN policy ON policy.workspace = ? AND policy.resource_type = 'WORKSPACE' AND policy.type = 'IAM'
		WHERE principal.deleted = FALSE
		  AND EXISTS (
			SELECT 1
			FROM jsonb_array_elements(policy.payload->'bindings') AS binding,
			     jsonb_array_elements_text(binding->'members') AS member
			WHERE member = 'users/' || principal.email
		  )
	`, workspaceID)

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
