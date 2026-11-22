package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func openTestDriver(ctx context.Context, t *testing.T, engine storepb.Engine, container *testcontainer.Container, username, password, database string) *Driver {
	t.Helper()

	d, err := newDriver().Open(ctx, engine, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: username,
			Host:     container.GetHost(),
			Port:     container.GetPort(),
		},
		Password:          password,
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close(ctx)) })

	driver, ok := d.(*Driver)
	require.True(t, ok)
	return driver
}
