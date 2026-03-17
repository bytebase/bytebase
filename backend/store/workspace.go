package store

import (
	"context"

	"github.com/pkg/errors"
)

// WorkspaceMessage is the message for a workspace.
type WorkspaceMessage struct {
	ResourceID string
	Name       string
}

// GetWorkspace returns the workspace.
func (s *Store) GetWorkspace(ctx context.Context) (*WorkspaceMessage, error) {
	var workspace WorkspaceMessage
	if err := s.GetDB().QueryRowContext(ctx,
		`SELECT resource_id, name FROM workspace WHERE deleted = FALSE LIMIT 1`,
	).Scan(&workspace.ResourceID, &workspace.Name); err != nil {
		return nil, errors.Wrap(err, "failed to get workspace")
	}
	return &workspace, nil
}
