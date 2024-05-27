// Package dynamodb is the plugin for DynamoDB driver.
package dynamodb

import (
	"context"
	"database/sql"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_DYNAMODB, newDriver)
}

// Driver is the BigQuery driver.
type Driver struct {
	config    db.ConnectionConfig
	connCtx   db.ConnectionContext
	client    *dynamodb.Client
	awsConfig aws.Config
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a BigQuery driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, conf db.ConnectionConfig) (db.Driver, error) {
	d.config = conf
	d.connCtx = conf.ConnectionContext

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load AWS config")
	}
	d.awsConfig = cfg
	client := dynamodb.NewFromConfig(cfg)
	d.client = client
	return d, nil
}

// Close closes the driver.
func (*Driver) Close(_ context.Context) error {
	return nil
}

// Ping pings the instance.
func (d *Driver) Ping(ctx context.Context) error {
	// DynamoDB does not support ping method, we list tables instead. To avoid network overhead,
	// we set the limit to 1.
	var limit int32 = 1
	_, err := d.client.ListTables(ctx, &dynamodb.ListTablesInput{
		Limit: &limit,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list dynamodb tables")
	}
	return nil
}

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_DYNAMODB
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement.
func (*Driver) Execute(_ context.Context, _ string, _ db.ExecuteOptions) (int64, error) {
	panic("implement me")
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	panic("implement me")
}

// RunStatement executes a SQL statement.
func (*Driver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	panic("implement me")
}
