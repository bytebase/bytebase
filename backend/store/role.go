package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// RoleMessage is the message for roles.
type RoleMessage struct {
	ResourceID  string
	Name        string
	Description string

	// Output only
	CreatorID int
}

// UpdateRoleMessage is the message for updating roles.
type UpdateRoleMessage struct {
	UpdaterID  int
	ResourceID string

	Name        *string
	Description *string
}

// CreateRole creates a new role.
func (s *Store) CreateRole(ctx context.Context, create *RoleMessage, creatorID int) (*RoleMessage, error) {
	query := `
		INSERT INTO
			role (creator_id, updater_id, resource_id, name, description)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := s.db.db.ExecContext(ctx, query, creatorID, creatorID, create.ResourceID, create.Name, create.Description); err != nil {
		return nil, err
	}
	return create, nil
}

// GetRole returns a role by ID.
func (s *Store) GetRole(ctx context.Context, resourceID string) (*RoleMessage, error) {
	query := `
		SELECT
			creator_id, name, description
		FROM role
		WHERE resource_id = $1
	`
	var role RoleMessage
	if err := s.db.db.QueryRowContext(ctx, query, resourceID).Scan(&role.CreatorID, &role.Name, &role.Description); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	role.ResourceID = resourceID
	return &role, nil
}

// ListRoles returns a list of roles.
func (s *Store) ListRoles(ctx context.Context) ([]*RoleMessage, error) {
	query := `
		SELECT
			creator_id, resource_id, name, description
		FROM role
	`
	rows, err := s.db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*RoleMessage
	roles = append(roles,
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.Owner.String(),
			Name:        "Project owner",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.Developer.String(),
			Name:        "Project developer",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.Releaser.String(),
			Name:        "Project releaser",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.DatabaseViewer.String(),
			Name:        "Project database viewer",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.Exporter.String(),
			Name:        "Project exporter",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.Querier.String(),
			Name:        "Project querier",
			Description: "",
		},
	)

	for rows.Next() {
		var role RoleMessage
		if err := rows.Scan(&role.CreatorID, &role.ResourceID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, &role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// UpdateRole updates an existing role.
func (s *Store) UpdateRole(ctx context.Context, patch *UpdateRoleMessage) (*RoleMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ResourceID)

	query := fmt.Sprintf(`
		UPDATE role
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d
		RETURNING creator_id, name, description
	`, len(args))

	role := RoleMessage{
		ResourceID: patch.ResourceID,
	}
	if err := s.db.db.QueryRowContext(ctx, query, args...).Scan(&role.CreatorID, &role.Name, &role.Description); err != nil {
		return nil, err
	}

	return &role, nil
}

// DeleteRole deletes an existing role.
func (s *Store) DeleteRole(ctx context.Context, resourceID string) error {
	query := `
		DELETE FROM role
		WHERE resource_id = $1
	`
	if _, err := s.db.db.ExecContext(ctx, query, resourceID); err != nil {
		return err
	}
	return nil
}
