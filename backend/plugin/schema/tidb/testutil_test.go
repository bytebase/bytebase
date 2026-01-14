package tidb

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
)

// createTiDBDriver creates and opens a TiDB driver connection
func createTiDBDriver(ctx context.Context, host, port, database string) (db.Driver, error) {
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
	return driver.Open(ctx, storepb.Engine_TIDB, config)
}

// executeStatements executes multiple SQL statements
// TiDB/MySQL driver supports multi-statement execution natively
func executeStatements(db *sql.DB, statements string) error {
	if _, err := db.Exec(statements); err != nil {
		return errors.Wrapf(err, "failed to execute statements")
	}
	return nil
}
