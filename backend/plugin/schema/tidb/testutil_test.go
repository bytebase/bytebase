package tidb

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
)

// createTiDBDriver creates and opens a TiDB driver connection
func createTiDBDriver(ctx context.Context, host, port, database string) (*tidbdb.Driver, error) {
	driver := &tidbdb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.5.0",
			DatabaseName:  database,
		},
	}
	d, err := driver.Open(ctx, storepb.Engine_TIDB, config)
	if err != nil {
		return nil, err
	}
	tidbDriver, ok := d.(*tidbdb.Driver)
	if !ok {
		return nil, errors.Errorf("failed to cast to TiDB driver")
	}
	return tidbDriver, nil
}
