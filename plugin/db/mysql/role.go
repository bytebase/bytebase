package mysql

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *storepb.DatabaseRoleUpsert) (*storepb.DatabaseRole, error) {
	return nil, errors.Errorf("create role for MySQL is not implemented yet")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *storepb.DatabaseRoleUpsert) (*storepb.DatabaseRole, error) {
	return nil, errors.Errorf("update role for MySQL is not implemented yet")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*storepb.DatabaseRole, error) {
	return nil, errors.Errorf("find role for MySQL is not implemented yet")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("delete role for MySQL is not implemented yet")
}
