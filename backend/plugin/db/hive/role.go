package hive

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// Role
// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// ListRole lists the role.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("Not implemeted")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("Not implemeted")
}
