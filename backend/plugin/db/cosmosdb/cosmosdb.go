// Package cosmosdb is Bytebase's Cosmos DB driver: a thin adapter over
// github.com/bytebase/omni/cosmosdb/driver. It maps a db.ConnectionConfig onto
// the contract's Config and back; the client, the sync queries, the emulator's
// REST listings and the cross-partition query path belong to the omni driver.
//
// Constraints the adapter owns:
//
//   - A development build pointed at a local endpoint (localhost, 127.0.0.1 or
//     host.docker.internal) is the emulator: it authenticates with the
//     well-known account key and accepts the certificate it signs itself, and
//     the adapter says so through Params["emulator"].
//   - Otherwise an app registration becomes a client-secret credential, and a
//     data source without one falls back to azidentity's default chain. Config
//     carries no account key, application name, read-only flag or engine
//     version, and there is no SSH tunnel: Cosmos DB has no such session knob.
//   - Execute is connect.CodeUnimplemented. The query language has no statement
//     that writes, and the omni driver has no Executor for this engine.
//   - Dump writes nothing: the synced metadata does not carry the partition
//     key, indexing policy, unique keys or default TTL that recreate a
//     container.
//   - QueryOptions.MaxResultBytes enforces MaximumSQLResultSize: reading stops
//     at the first row that passes the cap and the result reports Truncated.
//   - A statement's failure is the call's connect.CodeInternal error rather than
//     a result, because the API reads one result per query here.
//   - QueryConn runs one statement scoped to QueryContext.Container, which is the
//     query's scope, not a schema, and the query language cannot open another.
//   - An explain request never reaches QueryConn: db.SupportedExplainFormats(COSMOSDB)
//     is empty, so the API refuses it first. The flag is carried to the driver
//     for the contract's sake, since the driver answers an explain with the plan
//     the query engine would run.
package cosmosdb

import (
	"context"
	"crypto/tls"
	"net/url"

	"connectrpc.com/connect"
	omnicosmosdb "github.com/bytebase/omni/cosmosdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
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
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the contract's Config. It does not reach
// the account: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	driverConfig, err := connectionConfig(config.DataSource, config.ConnectionContext.DatabaseName)
	if err != nil {
		return nil, err
	}
	inner, err := omnicosmosdb.Open(ctx, driverConfig)
	if err != nil {
		return nil, err
	}
	return &Driver{OmniAdapter: util.OmniAdapter{Driver: inner}}, nil
}

// connectionConfig maps a data source onto the contract's Config: the emulator
// path in a development build, an app registration as a client secret, or
// azidentity's default chain.
func connectionConfig(dataSource *storepb.DataSource, database string) (omnidriver.Config, error) {
	if dataSource == nil {
		return omnidriver.Config{}, errors.New("cosmosdb: a data source is required")
	}
	driverConfig := omnidriver.Config{
		Host:     dataSource.Host,
		Database: database,
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
	return driverConfig, nil
}

func isLocalhostEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "host.docker.internal"
}

// Execute is unimplemented: the query language has no statement that changes
// data, and the omni driver has no Executor for this engine.
func (*Driver) Execute(context.Context, string, db.ExecuteOptions) (int64, error) {
	return 0, connect.NewError(connect.CodeUnimplemented, errors.New("method Execute unimplemented"))
}

// QueryConn runs statement in the container QueryContext.Container names.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Container == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("container argument is required for CosmosDB"))
	}
	// Allow `SELECT * FROM {container} [alias]` only.
	if _, _, err := base.ValidateSQLForEditor(storepb.Engine_COSMOSDB, statement); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("support simple SELECT statement for Cosmos DB engine only, err: %s", err.Error()))
	}

	results, err := d.QueryEphemeralSession(ctx, statement, queryOptions(queryContext))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			// Known difference: a failed statement is the call's error, not a
			// result, which is the shape the previous driver answered with.
			return nil, connect.NewError(connect.CodeInternal, result.Err)
		}
		converted = append(converted, util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted, nil
}

// queryOptions maps the product's query context onto the driver's. The
// container is the query's scope, not a schema: a statement reads the container
// the caller already opened, and the query language has no form that opens
// another one.
func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return omnidriver.QueryOptions{
		Schema:         queryContext.Container,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}
}
