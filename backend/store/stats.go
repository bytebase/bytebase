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
	if v, ok := s.userCountCache.Get(workspaceID); ok {
		return v, nil
	}

	// Count distinct active users who are members of the workspace, either
	// directly (users/email) or via group expansion (groups/email -> group members).
	q := qb.Q().Space(`
		WITH iam_members AS (
			SELECT DISTINCT jsonb_array_elements_text(binding->'members') AS member
			FROM policy,
			     jsonb_array_elements(policy.payload->'bindings') AS binding
			WHERE policy.workspace = ?
			  AND policy.resource_type = 'WORKSPACE'
			  AND policy.type = 'IAM'
		),
		direct_users AS (
			SELECT SUBSTRING(member FROM 7) AS email
			FROM iam_members
			WHERE member LIKE 'users/%'
		),
		group_users AS (
			SELECT SUBSTRING(gm.value->>'member' FROM 7) AS email
			FROM iam_members im
			JOIN user_group ug ON ug.workspace = ? AND ug.email = SUBSTRING(im.member FROM 8)
			, jsonb_array_elements(ug.payload->'members') AS gm
			WHERE im.member LIKE 'groups/%'
			  AND gm.value->>'member' LIKE 'users/%'
		),
		all_users AS (
			SELECT email FROM direct_users
			UNION
			SELECT email FROM group_users
		)
		SELECT count(*)
		FROM principal
		WHERE principal.deleted = FALSE
		  AND principal.email IN (SELECT email FROM all_users)
	`, workspaceID, workspaceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	s.userCountCache.Add(workspaceID, count)
	return count, nil
}
