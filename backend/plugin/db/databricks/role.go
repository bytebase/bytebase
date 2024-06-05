package databricks

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (d *DatabricksDriver) CreateRole(ctx context.Context, upsert *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

func (d *DatabricksDriver) DeleteRole(ctx context.Context, roleName string) error {
	panic("unimplemented")
}

func (d *DatabricksDriver) FindRole(ctx context.Context, roleName string) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

func (d *DatabricksDriver) UpdateRole(ctx context.Context, roleName string, upsert *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}

func (d *DatabricksDriver) ListRole(ctx context.Context) ([]*db.DatabaseRoleMessage, error) {
	panic("unimplemented")
}
