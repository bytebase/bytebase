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

// CountActivePrincipals counts non-deleted principals cross-workspace. Used for display purposes
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

// CountSeatOccupyingUsers counts the distinct end users occupying license seats
// in a workspace: direct workspace IAM members (users/{email}) plus members of
// IAM groups (groups/{email} or groups/{id}), deduplicated. Pending invites
// (emails without a principal yet) count; soft-deleted principals, service
// accounts, and workload identities do not. When the workspace IAM policy
// includes allUsers, every active principal occupies a seat.
//
// The query reads the shared metadata tables directly and bypasses every
// per-replica cache so replicas derive equivalent values from the same state.
func (s *Store) CountSeatOccupyingUsers(ctx context.Context, workspaceID string) (int, error) {
	const query = `
WITH seat_emails AS (
	SELECT DISTINCT lower(substring(member FROM char_length('users/') + 1)) AS email
	FROM policy p,
		jsonb_array_elements(p.payload->'bindings') AS binding,
		jsonb_array_elements_text(binding->'members') AS member
	WHERE p.workspace = $1
		AND p.resource_type = 'WORKSPACE'
		AND p.resource = 'workspaces/' || $1
		AND p.type = 'IAM'
		AND member LIKE 'users/%'
	UNION
	SELECT DISTINCT lower(substring(gm->>'member' FROM char_length('users/') + 1)) AS email
	FROM policy p,
		jsonb_array_elements(p.payload->'bindings') AS binding,
		jsonb_array_elements_text(binding->'members') AS member,
		user_group ug,
		jsonb_array_elements(ug.payload->'members') AS gm
	WHERE p.workspace = $1
		AND p.resource_type = 'WORKSPACE'
		AND p.resource = 'workspaces/' || $1
		AND p.type = 'IAM'
		AND member LIKE 'groups/%'
		AND ug.workspace = $1
		AND ('groups/' || ug.email = member OR 'groups/' || ug.id = member)
		AND gm->>'member' LIKE 'users/%'
)
SELECT
	CASE
		WHEN EXISTS (
			SELECT 1
			FROM policy p,
				jsonb_array_elements(p.payload->'bindings') AS binding,
				jsonb_array_elements_text(binding->'members') AS member
			WHERE p.workspace = $1
				AND p.resource_type = 'WORKSPACE'
				AND p.resource = 'workspaces/' || $1
				AND p.type = 'IAM'
				AND member = 'allUsers'
		)
		THEN (SELECT count(*) FROM principal WHERE deleted = FALSE)
		ELSE (
			SELECT count(*)
			FROM seat_emails s
			WHERE NOT EXISTS (
					SELECT 1 FROM principal p WHERE p.email = s.email AND p.deleted
				)
		)
	END`
	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, workspaceID).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "failed to count seat-occupying users for workspace %q", workspaceID)
	}
	return count, nil
}
