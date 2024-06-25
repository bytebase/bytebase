package databricks

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, nil
}

func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return nil
}

func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, nil
}

func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, nil
}

// TODO(tommy): maybe implement this when Permissions API is implemented.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, nil
}
