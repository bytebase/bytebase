// Package obo is for OceanBase Oracle mode
package obo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Register OceanBase Oracle mode driver.
	_ "github.com/mattn/go-oci8"
)

func init() {
	db.Register(storepb.Engine_OCEANBASE_ORACLE, newDriver)
}

type Driver struct {
	db *sql.DB
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	dsn := fmt.Sprintf("%s/%s@%s:%s", config.Username, config.Password, config.Host, config.Port)

	db, err := sql.Open("oci8", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection")
	}

	driver.db = db
	return driver, nil
}

func (*Driver) Close(context.Context) error {
	return errors.New("not implemented")
}

func (*Driver) Ping(context.Context) error {
	return errors.New("not implemented")
}

func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_OCEANBASE_ORACLE
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) Execute(context.Context, string, bool, db.ExecuteOptions) (int64, error) {
	return 0, errors.New("not implemented")
}

func (*Driver) QueryConn(context.Context, *sql.Conn, string, *db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) RunStatement(context.Context, *sql.Conn, string) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}
