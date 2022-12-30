package spanner

import (
	"context"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// CreateRole creates a role.
func (*Driver) CreateRole(_ context.Context, _ *v1pb.DatabaseRoleUpsert) (*v1pb.DatabaseRole, error) {
	panic("not implemented")
}

// UpdateRole updates a role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *v1pb.DatabaseRoleUpsert) (*v1pb.DatabaseRole, error) {
	panic("not implemented")
}

// FindRole finds the role.
func (*Driver) FindRole(_ context.Context, _ string) (*v1pb.DatabaseRole, error) {
	panic("not implemented")
}

// ListRole lists the roles.
func (*Driver) ListRole(_ context.Context) ([]*v1pb.DatabaseRole, error) {
	panic("not implemented")
}

// DeleteRole deletes the role.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	panic("not implemented")
}
