package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// RoleMessage is the message for roles.
type RoleMessage struct {
	ResourceID  string
	Name        string
	Description string
	Permissions map[string]bool
}

// UpdateRoleMessage is the message for updating roles.
type UpdateRoleMessage struct {
	ResourceID string

	Name        *string
	Description *string
	Permissions *map[string]bool
}

func (s *Store) CheckRoleInUse(ctx context.Context, role string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM policy
			CROSS JOIN LATERAL jsonb_array_elements(payload->'bindings') AS binding
			WHERE type = 'bb.policy.iam' AND binding->>'role' = $1
		);
	`
	var exist bool
	if err := s.db.QueryRowContext(ctx, query, role).Scan(&exist); err != nil {
		return false, err
	}
	return exist, nil
}

// CreateRole creates a new role.
func (s *Store) CreateRole(ctx context.Context, create *RoleMessage) (*RoleMessage, error) {
	query := `
		INSERT INTO
			role (resource_id, name, description, permissions)
		VALUES ($1, $2, $3, $4)
	`
	p := &storepb.RolePermissions{}
	for k := range create.Permissions {
		p.Permissions = append(p.Permissions, k)
	}
	permissionBytes, err := protojson.Marshal(p)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx, query, create.ResourceID, create.Name, create.Description, permissionBytes); err != nil {
		return nil, err
	}
	s.rolesCache.Add(create.ResourceID, create)
	return create, nil
}

// GetRole returns a role by ID.
func (s *Store) GetRole(ctx context.Context, resourceID string) (*RoleMessage, error) {
	if v, ok := s.rolesCache.Get(resourceID); ok {
		return v, nil
	}
	query := `
		SELECT
			name, description, permissions
		FROM role
		WHERE resource_id = $1
	`
	role := &RoleMessage{
		Permissions: map[string]bool{},
	}
	var permissions []byte
	if err := s.db.QueryRowContext(ctx, query, resourceID).Scan(&role.Name, &role.Description, &permissions); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var rolePermissions storepb.RolePermissions
	if err := common.ProtojsonUnmarshaler.Unmarshal(permissions, &rolePermissions); err != nil {
		return nil, err
	}
	for _, v := range rolePermissions.Permissions {
		role.Permissions[v] = true
	}
	role.ResourceID = resourceID
	s.rolesCache.Add(resourceID, role)
	return role, nil
}

// ListRoles returns a list of roles.
func (s *Store) ListRoles(ctx context.Context) ([]*RoleMessage, error) {
	query := `
		SELECT
			resource_id, name, description, permissions
		FROM role
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*RoleMessage
	for rows.Next() {
		role := &RoleMessage{
			Permissions: map[string]bool{},
		}
		var permissionBytes []byte
		if err := rows.Scan(&role.ResourceID, &role.Name, &role.Description, &permissionBytes); err != nil {
			return nil, err
		}
		var rolePermissions storepb.RolePermissions
		if err := common.ProtojsonUnmarshaler.Unmarshal(permissionBytes, &rolePermissions); err != nil {
			return nil, err
		}
		for _, v := range rolePermissions.Permissions {
			role.Permissions[v] = true
		}
		s.rolesCache.Add(role.ResourceID, role)
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// UpdateRole updates an existing role.
func (s *Store) UpdateRole(ctx context.Context, patch *UpdateRoleMessage) (*RoleMessage, error) {
	set, args := []string{}, []any{}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Permissions; v != nil {
		p := &storepb.RolePermissions{}
		for k := range *v {
			p.Permissions = append(p.Permissions, k)
		}
		permissionBytes, err := protojson.Marshal(p)
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
		RETURNING name, description, permissions
	`, len(args))

	role := &RoleMessage{
		ResourceID:  patch.ResourceID,
		Permissions: map[string]bool{},
	}
	var permissionBytes []byte
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&role.Name, &role.Description, &permissionBytes); err != nil {
		return nil, err
	}
	s.rolesCache.Remove(patch.ResourceID)
	var rolePermissions storepb.RolePermissions
	if err := common.ProtojsonUnmarshaler.Unmarshal(permissionBytes, &rolePermissions); err != nil {
		return nil, err
	}
	for _, v := range rolePermissions.Permissions {
		role.Permissions[v] = true
	}

	s.rolesCache.Add(role.ResourceID, role)
	return role, nil
}

// DeleteRole deletes an existing role.
func (s *Store) DeleteRole(ctx context.Context, resourceID string) error {
	query := `
		DELETE FROM role
		WHERE resource_id = $1
	`
	if _, err := s.db.ExecContext(ctx, query, resourceID); err != nil {
		return err
	}
	s.rolesCache.Remove(resourceID)
	return nil
}
