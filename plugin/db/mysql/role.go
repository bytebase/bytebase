package mysql

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/db"
)

// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.RoleUpsert) (*db.Role, error) {
	return nil, errors.Errorf("create role for MySQL is not implemented yet")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.RoleUpsert) (*db.Role, error) {
	return nil, errors.Errorf("update role for MySQL is not implemented yet")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.Role, error) {
	return nil, errors.Errorf("find role for MySQL is not implemented yet")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("delete role for MySQL is not implemented yet")
}
