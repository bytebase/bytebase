package obo

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// Role
// CreateRole creates the role.
func (*Driver) CreateRole(context.Context, *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.New("not implemented")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(context.Context, string, *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.New("not implemented")
}

// FindRole finds the role by name.
func (*Driver) FindRole(context.Context, string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.New("not implemented")
}

// ListRole lists the role.
func (*Driver) ListRole(context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.New("not implemented")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(context.Context, string) error {
	return errors.New("not implemented")
}
