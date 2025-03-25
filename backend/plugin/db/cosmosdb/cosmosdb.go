// Package cosmosdb is the plugin for CosmosDB driver.
package cosmosdb

import (
	"context"
	"database/sql"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_COSMOSDB, newDriver)
}

// Driver is the CosmosDB driver.
type Driver struct {
	client       *azcosmos.Client
	connCfg      db.ConnectionConfig
	databaseName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a CosmosDB driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	endpoint := connCfg.DataSource.Host
	var credential azcore.TokenCredential
	if clientSecretCredential := connCfg.DataSource.GetClientSecretCredential(); clientSecretCredential != nil {
		c, err := azidentity.NewClientSecretCredential(clientSecretCredential.TenantId, clientSecretCredential.ClientId, clientSecretCredential.ClientSecret, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create client secret credential")
		}
		credential = c
	} else {
		c, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to found default Azure credential")
		}
		credential = c
	}
	client, err := azcosmos.NewClient(endpoint, credential, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create CosmosDB client")
	}
	d.client = client
	d.databaseName = connCfg.ConnectionContext.DatabaseName
	d.connCfg = connCfg
	return d, nil
}

// Close closes the CosmosDB driver.
func (*Driver) Close(_ context.Context) error {
	return nil
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	queryPager := d.client.NewQueryDatabasesPager("select 1", nil)
	for queryPager.More() {
		_, err := queryPager.NextPage(ctx)
		if err != nil {
			// TODO(zp): Deserialize the error into azcore.ResponseError
			return errors.Wrapf(err, "failed to ping CosmosDB")
		}
	}
	return nil
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) Execute(_ context.Context, _ string, _ db.ExecuteOptions) (int64, error) {
	return 0, status.Errorf(codes.Unimplemented, "method Execute unimplemented")
}

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Container == "" {
		return nil, status.Errorf(codes.InvalidArgument, "container argument is required for CosmosDB")
	}

	// Allow `SELECT * FROM {container} [alias]` only.
	_, _, err := base.ValidateSQLForEditor(storepb.Engine_COSMOSDB, statement)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "support simple SELECT statement for Cosmos DB engine only, err: %s", err.Error())
	}

	startTime := time.Now()
	container, err := d.client.NewContainer(d.databaseName, queryContext.Container)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrapf(err, "failed to create container").Error())
	}

	var queryOption *azcosmos.QueryOptions
	// TODO(zp): Rewrite limit while parser supported.
	if queryContext.Limit > 0 && queryContext.Limit < 1000 {
		// Set the page size to limit to avoid large page size.
		queryOption = &azcosmos.QueryOptions{
			PageSizeHint: int32(queryContext.Limit),
		}
	}

	pager := container.NewCrossPartitionQueryItemsPager(statement, queryOption)
	var items [][]byte
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, errors.Wrapf(err, "failed to read more items").Error())
		}
		var stop bool
		for _, item := range response.Items {
			items = append(items, item)
			if queryContext.Limit > 0 && len(items) >= queryContext.Limit {
				stop = true
				break
			}
		}
		if stop {
			break
		}
	}

	result := &v1pb.QueryResult{
		ColumnNames:     []string{"result"},
		ColumnTypeNames: []string{"TEXT"},
	}
	for _, item := range items {
		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: string(item)}},
			},
		})
	}

	result.Latency = durationpb.New(time.Since(startTime))

	return []*v1pb.QueryResult{result}, nil
}
