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

	driver := newDriver()
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

// openTestDriverOn opens a driver on one database of the container.
func openTestDriverOn(ctx context.Context, t *testing.T, container *testcontainer.Container, database string) *Driver {
	t.Helper()

	driver := newDriver()
	d, err := driver.Open(ctx, storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     container.GetHost(),
			Port:     container.GetPort(),
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	})
	require.NoError(t, err)

	tidbDriver, ok := d.(*Driver)
	require.True(t, ok)
	return tidbDriver
}
