package store

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

// FindRoleMessage is the message for finding roles.
type FindRoleMessage struct {
	ResourceID *string
}

type RoleUsedByResource struct {
	ResourceType storepb.Policy_Resource
	Resource     string
}

func (s *Store) GetResourcesUsedByRole(ctx context.Context, role string) ([]*RoleUsedByResource, error) {
	q := qb.Q().Space(`
		SELECT resource, resource_type FROM policy
		CROSS JOIN LATERAL jsonb_array_elements(payload->'bindings') AS binding
		WHERE
			type = ? AND
			COALESCE(jsonb_array_length(binding->'members'), 0) > 0 AND
			binding->>'role' = ?
		GROUP BY resource, resource_type
	`, storepb.Policy_IAM.String(), role)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var response []*RoleUsedByResource
	for rows.Next() {
		var usedByResource RoleUsedByResource
		if err := rows.Scan(
			&usedByResource.Resource,
			&usedByResource.ResourceType,
		); err != nil {
			return nil, err
		}
		response = append(response, &usedByResource)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return response, nil
}

// CreateRole creates a new role.
func (s *Store) CreateRole(ctx context.Context, create *RoleMessage) (*RoleMessage, error) {
	p := &storepb.RolePermissions{}
	for k := range create.Permissions {
		p.Permissions = append(p.Permissions, k)
	}
	permissionBytes, err := protojson.Marshal(p)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO
			role (resource_id, name, description, permissions)
		VALUES (?, ?, ?, ?)
	`, create.ResourceID, create.Name, create.Description, permissionBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}
	s.rolesCache.Add(create.ResourceID, create)
	return create, nil
}

// GetRole returns a role by ID.
func (s *Store) GetRole(ctx context.Context, find *FindRoleMessage) (*RoleMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.rolesCache.Get(*find.ResourceID); ok && s.enableCache {
			return v, nil
		}
	}

	roles, err := s.ListRoles(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	if len(roles) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d roles with filter %+v, expect 1", len(roles), find)}
	}
	role := roles[0]

	s.rolesCache.Add(role.ResourceID, role)
	return role, nil
}

// ListRoles returns a list of roles.
func (s *Store) ListRoles(ctx context.Context, find *FindRoleMessage) ([]*RoleMessage, error) {
	q := qb.Q().Space(`
		SELECT
			resource_id, name, description, permissions
		FROM role
		WHERE TRUE
	`)

	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
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
	set := qb.Q()
	if v := patch.Name; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Description; v != nil {
		set.Comma("description = ?", *v)
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
		set.Comma("permissions = ?", permissionBytes)
	}
	if set.Len() == 0 {
		return nil, errors.New("no fields to update")
	}

	q := qb.Q().Space(`
		UPDATE role
		SET ?
		WHERE resource_id = ?
		RETURNING name, description, permissions
	`, set, patch.ResourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	role := &RoleMessage{
		ResourceID:  patch.ResourceID,
		Permissions: map[string]bool{},
	}
	var permissionBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&role.Name, &role.Description, &permissionBytes); err != nil {
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
	q := qb.Q().Space(`
		DELETE FROM role
		WHERE resource_id = ?
	`, resourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	s.rolesCache.Remove(resourceID)
	return nil
}
