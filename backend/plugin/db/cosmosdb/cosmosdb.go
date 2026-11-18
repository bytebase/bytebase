// Package cosmosdb is Bytebase's Cosmos DB driver: a thin adapter over
// github.com/bytebase/omni/cosmosdb/driver. It maps a db.ConnectionConfig onto
// the contract's Config and back; the client, the sync queries, the emulator's
// REST listings and the cross-partition query path belong to the omni driver.
//
// The adapter keeps the credential and network path Bytebase owns:
//
//   - An emulator endpoint in a development build (localhost,
//     127.0.0.1 or host.docker.internal) authenticates with the emulator's
//     well-known account key and accepts the certificate it signs itself, as
//     the old driver did. The old driver chose that path from the build flag
//     and the endpoint; the adapter tells the omni driver the endpoint is the
//     emulator (Params["emulator"]) and supplies the key and TLS settings.
//   - Otherwise a data source with an Azure app registration becomes a
//     client-secret credential, and a data source without one falls back to
//     azidentity's default chain. The account key, application name, read-only
//     flag and engine version are not part of either mapping, and neither is a
//     SQL connection parameter: the adapter leaves Config.Dial, Config.ReadOnly
//     and Config.ApplicationName unset, because the old driver supported no SSH
//     tunnel, no read-only session and no application name.
//
// Every db.Driver method:
//
//   - Open builds the omni driver; it does not reach the account. Ping does.
//   - Close releases the client's connections. The old Close was a no-op.
//   - Ping reads the database list, which is what the old driver did.
//   - GetDB returns nil: Cosmos DB has no database/sql handle, and the API
//     pools no *sql.Conn for it.
//   - Execute reports connect.CodeUnimplemented with "method Execute
//     unimplemented": container items are written through the container API,
//     not the query language, and the omni driver has no Executor either.
//   - QueryConn runs one statement in QueryContext.Container and returns one
//     result, the shape the API expects for this engine.
//   - SyncInstance reads the account's databases, and SyncDBSchema the
//     database's containers as tables with no columns, since items are
//     schemaless.
//   - Dump writes nothing: the synced metadata does not carry the partition
//     key, indexing policy, unique keys or default TTL that recreate a
//     container, and the old driver dumped nothing either.
//
// QueryContext.Explain never reaches QueryConn for this engine: the API refuses
// an explain request before opening a driver, because
// db.SupportedExplainFormats(COSMOSDB) is empty (validateExplainFormat in
// backend/api/v1/sql_service.go). The old driver ignored the flag, and the
// adapter does not map it, so an explain that did arrive would run the
// statement as an ordinary query, exactly as before.
//
// Two defects are kept rather than fixed, so a reader can tell a cutover from a
// behaviour change:
//
//   - MaximumSQLResultSize is not enforced. The old driver returned every row a
//     statement delivered however the policy capped the result, and the adapter
//     leaves QueryOptions.MaxResultBytes unset for the same reason.
//   - A query failed as the call's error, not as a result. The omni driver
//     reports a failed statement in its Result; the adapter turns a Result with
//     an error back into the call's connect.CodeInternal answer, which is what
//     the old driver returned.
package cosmosdb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"io"
	"net/url"

	"connectrpc.com/connect"
	omnicosmosdb "github.com/bytebase/omni/cosmosdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_COSMOSDB, newDriver)
}

// cosmosDBEmulatorKey is the well-known master key for the CosmosDB emulator.
// https://learn.microsoft.com/en-us/azure/cosmos-db/emulator
const cosmosDBEmulatorKey = "C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw=="

// Driver is Bytebase's Cosmos DB driver.
type Driver struct {
	inner *omnicosmosdb.Driver
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the contract's Config. It does not reach
// the account: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource
	if dataSource == nil {
		return nil, errors.New("cosmosdb: a data source is required")
	}
	driverConfig := omnidriver.Config{
		Host:     dataSource.Host,
		Database: config.ConnectionContext.DatabaseName,
		Params:   map[string]string{},
	}
	switch credential := dataSource.GetAzureCredential(); {
	case common.IsDev() && isLocalhostEndpoint(dataSource.Host):
		// The old driver took this path ahead of the app registration, and the
		// emulator's metadata listings go over its REST API rather than the
		// client's query path.
		driverConfig.Params[omnicosmosdb.ParamAuth] = omnicosmosdb.AuthMasterKey
		driverConfig.Params[omnicosmosdb.ParamEmulator] = "true"
		driverConfig.Password = cosmosDBEmulatorKey
		driverConfig.TLS = func(string) (*tls.Config, error) {
			return &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, //nolint:gosec // the emulator signs its certificate itself
			}, nil
		}
	case credential != nil:
		driverConfig.Params[omnicosmosdb.ParamAuth] = omnicosmosdb.AuthClientSecret
		driverConfig.Params[omnicosmosdb.ParamTenantID] = credential.TenantId
		driverConfig.User = credential.ClientId
		driverConfig.Password = credential.ClientSecret
	default:
		driverConfig.Params[omnicosmosdb.ParamAuth] = omnicosmosdb.AuthDefault
	}
	inner, err := omnicosmosdb.Open(ctx, driverConfig)
	if err != nil {
		return nil, err
	}
	return &Driver{inner: inner}, nil
}

func isLocalhostEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "host.docker.internal"
}

func (d *Driver) Close(context.Context) error {
	if d.inner == nil {
		return nil
	}
	return d.inner.Close()
}

// Ping reads the account's database list as a connectivity check.
func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// GetDB returns nil: Cosmos DB has no database/sql handle.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute is unimplemented: the query language has no statement that changes
// data, and the omni driver has no Executor for this engine.
func (*Driver) Execute(context.Context, string, db.ExecuteOptions) (int64, error) {
	return 0, connect.NewError(connect.CodeUnimplemented, errors.New("method Execute unimplemented"))
}

// Dump writes nothing and returns nil; the package doc says what a synced
// container lacks that a restore would need.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

// QueryConn runs statement in the container QueryContext.Container names.
// conn is unused: the API pools no *sql.Conn for an engine whose GetDB is nil,
// and the contract's session is not one.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Container == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("container argument is required for CosmosDB"))
	}
	// Allow `SELECT * FROM {container} [alias]` only.
	if _, _, err := base.ValidateSQLForEditor(storepb.Engine_COSMOSDB, statement); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("support simple SELECT statement for Cosmos DB engine only, err: %s", err.Error()))
	}

	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		// The container is the query's scope, not a schema: a statement reads
		// the container the caller already opened, and the query language has
		// no form that opens another one.
		Schema: queryContext.Container,
		Limit:  queryContext.Limit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			return nil, connect.NewError(connect.CodeInternal, result.Err)
		}
		converted = append(converted, util.OmniResultToQueryResult(result, 0))
	}
	return converted, nil
}

// SyncInstance reads the account's databases, without their containers.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema reads the connected database's containers as tables.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.inner.SyncDatabase(ctx)
}
