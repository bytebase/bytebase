package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// RoleMessage is the message for roles.
type RoleMessage struct {
	ResourceID  string
	Name        string
	Description string
	Permissions *storepb.RolePermissions

	// Output only
	CreatorID int
}

// UpdateRoleMessage is the message for updating roles.
type UpdateRoleMessage struct {
	UpdaterID  int
	ResourceID string

	Name        *string
	Description *string
	Permissions *storepb.RolePermissions
}

// CreateRole creates a new role.
func (s *Store) CreateRole(ctx context.Context, create *RoleMessage, creatorID int) (*RoleMessage, error) {
	query := `
		INSERT INTO
			role (creator_id, updater_id, resource_id, name, description, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	permissionBytes, err := protojson.Marshal(create.Permissions)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.db.ExecContext(ctx, query, creatorID, creatorID, create.ResourceID, create.Name, create.Description, permissionBytes); err != nil {
		return nil, err
	}
	return create, nil
}

// GetRole returns a role by ID.
func (s *Store) GetRole(ctx context.Context, resourceID string) (*RoleMessage, error) {
	query := `
		SELECT
			creator_id, name, description, permissions
		FROM role
		WHERE resource_id = $1
	`
	var role RoleMessage
	var permissions []byte
	if err := s.db.db.QueryRowContext(ctx, query, resourceID).Scan(&role.CreatorID, &role.Name, &role.Description, &permissions); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var rolePermissions storepb.RolePermissions
	if err := protojson.Unmarshal(permissions, &rolePermissions); err != nil {
		return nil, err
	}
	role.Permissions = &rolePermissions
	role.ResourceID = resourceID
	return &role, nil
}

// ListRoles returns a list of roles.
func (s *Store) ListRoles(ctx context.Context) ([]*RoleMessage, error) {
	query := `
		SELECT
			creator_id, resource_id, name, description, permissions
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
			ResourceID:  api.WorkspaceAdmin.String(),
			Name:        "Workspace admin",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.WorkspaceDBA.String(),
			Name:        "Workspace DBA",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.WorkspaceMember.String(),
			Name:        "Workspace member",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.ProjectOwner.String(),
			Name:        "Project owner",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.ProjectDeveloper.String(),
			Name:        "Project developer",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.ProjectReleaser.String(),
			Name:        "Project releaser",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.ProjectQuerier.String(),
			Name:        "Project querier",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.ProjectExporter.String(),
			Name:        "Project exporter",
			Description: "",
		},
		&RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  api.ProjectViewer.String(),
			Name:        "Project viewer",
			Description: "",
		},
	)

	for rows.Next() {
		var role RoleMessage
		var permissions []byte
		if err := rows.Scan(&role.CreatorID, &role.ResourceID, &role.Name, &role.Description, &permissions); err != nil {
			return nil, err
		}
		var rolePermissions storepb.RolePermissions
		if err := protojson.Unmarshal(permissions, &rolePermissions); err != nil {
			return nil, err
		}
		role.Permissions = &rolePermissions
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
	if v := patch.Permissions; v != nil {
		permissionBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("permissions = $%d", len(args)+1)), append(args, permissionBytes)
	}
	args = append(args, patch.ResourceID)

	query := fmt.Sprintf(`
		UPDATE role
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d
		RETURNING creator_id, name, description, permissions
	`, len(args))

	role := RoleMessage{
		ResourceID: patch.ResourceID,
	}
	var permissions []byte
	if err := s.db.db.QueryRowContext(ctx, query, args...).Scan(&role.CreatorID, &role.Name, &role.Description, &permissions); err != nil {
		return nil, err
	}
	var rolePermissions storepb.RolePermissions
	if err := protojson.Unmarshal(permissions, &rolePermissions); err != nil {
		return nil, err
	}
	role.Permissions = &rolePermissions
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
