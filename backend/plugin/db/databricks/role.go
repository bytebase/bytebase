package databricks

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

func (*Driver) DeleteRole(_ context.Context, _ string) error {
	panic("unimplemented")
}

func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}
