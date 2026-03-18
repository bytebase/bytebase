package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

// WorkspaceMessage is the message for a workspace.
type WorkspaceMessage struct {
	ResourceID string
	Name       string
}

// getWorkspace returns the workspace. Store-internal only.
func (s *Store) getWorkspace(ctx context.Context) (*WorkspaceMessage, error) {
	var workspace WorkspaceMessage
	if err := s.GetDB().QueryRowContext(ctx,
		`SELECT resource_id, name FROM workspace WHERE deleted = FALSE LIMIT 1`,
	).Scan(&workspace.ResourceID, &workspace.Name); err != nil {
		return nil, errors.Wrap(err, "failed to get workspace")
	}
	return &workspace, nil
}

// GetWorkspaceID returns the workspace resource ID.
// Only used for OAuth2 token issuance and server startup.
func (s *Store) GetWorkspaceID(ctx context.Context) (string, error) {
	ws, err := s.getWorkspace(ctx)
	if err != nil {
		return "", err
	}
	return ws.ResourceID, nil
}

func (s *Store) FindWorkspaceIDByMemberEmail(ctx context.Context, memberName string) (string, error) {
	workspaces, err := s.FindWorkspacesByMemberEmail(ctx, memberName)
	if err != nil {
		return "", errors.Wrap(err, "failed to find workspaces for user")
	}
	if len(workspaces) == 0 {
		return "", errors.Errorf("%q is not a member of any workspace", memberName)
	}
	// TODO(ed): In SaaS mode with multiple workspaces, return a workspace picker
	// instead of auto-selecting the first one. For now, use the first workspace.
	return workspaces[0].ResourceID, nil
}

// FindWorkspacesByMemberEmail finds all workspaces where the given email is a member
// in the workspace IAM policy bindings. The memberName should be in the format
// "users/{email}", "serviceAccounts/{email}", etc.
// Returns workspaces sorted by name.
func (s *Store) FindWorkspacesByMemberEmail(ctx context.Context, memberName string) ([]*WorkspaceMessage, error) {
	q := qb.Q().Space(`
		SELECT DISTINCT w.resource_id, w.name
		FROM workspace w
		JOIN policy p ON p.workspace = w.resource_id
		WHERE p.resource_type = 'WORKSPACE'
		  AND p.type = 'IAM'
		  AND w.deleted = FALSE
		  AND EXISTS (
			SELECT 1
			FROM jsonb_array_elements(p.payload->'bindings') AS binding,
			     jsonb_array_elements_text(binding->'members') AS member
			WHERE member = ?
		  )
		ORDER BY w.name
	`, memberName)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find workspaces by member")
	}
	defer rows.Close()

	var workspaces []*WorkspaceMessage
	for rows.Next() {
		var ws WorkspaceMessage
		if err := rows.Scan(&ws.ResourceID, &ws.Name); err != nil {
			return nil, errors.Wrap(err, "failed to scan workspace")
		}
		workspaces = append(workspaces, &ws)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate workspaces")
	}
	return workspaces, nil
}
