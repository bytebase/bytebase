package elasticsearch

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// ListRole implements db.Driver.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

// FindRole implements db.Driver.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

// CreateRole implements db.Driver.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

// DeleteRole implements db.Driver.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	panic("unimplemented")
}

// UpdateRole implements db.Driver.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}
