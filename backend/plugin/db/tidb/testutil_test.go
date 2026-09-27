package tidb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func openTestDriver(ctx context.Context, t *testing.T, container *testcontainer.Container) *Driver {
	t.Helper()

	driver := &Driver{}
	d, err := driver.Open(ctx, storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     container.GetHost(),
			Port:     container.GetPort(),
		},
		ConnectionContext: db.ConnectionContext{},
	})
	require.NoError(t, err)

	tidbDriver, ok := d.(*Driver)
	require.True(t, ok)
	return tidbDriver
}
