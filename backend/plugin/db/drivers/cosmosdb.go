package drivers

import (
	"context"
	"crypto/tls"
	"net/url"

	"connectrpc.com/connect"
	omnicosmosdb "github.com/bytebase/omni/cosmosdb/driver"
	odriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// cosmosDBEmulatorKey is the well-known master key for the Cosmos DB emulator.
// https://learn.microsoft.com/en-us/azure/cosmos-db/emulator
const cosmosDBEmulatorKey = "C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw=="

// cosmosdbSpec has no execute: the query language has no statement that
// changes data, and the omni driver has no Executor.
var cosmosdbSpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := cosmosdbConfig(config.DataSource, config.ConnectionContext.DatabaseName)
		if err != nil {
			return nil, err
		}
		return omnicosmosdb.Open(ctx, cfg)
	},
	query: cosmosdbQuery,
}

// cosmosdbConfig picks the emulator path in a development build, an app
// registration as a client secret, or azidentity's default chain.
func cosmosdbConfig(dataSource *storepb.DataSource, database string) (odriver.Config, error) {
	if dataSource == nil {
		return odriver.Config{}, errors.New("cosmosdb: a data source is required")
	}
	cfg := odriver.Config{
		Host:     dataSource.Host,
		Database: database,
		Params:   map[string]string{},
	}
	switch credential := dataSource.GetAzureCredential(); {
	case common.IsDev() && isLocalhostEndpoint(dataSource.Host):
		// The emulator path wins over an app registration, and the emulator's
		// metadata listings go over its REST API.
		cfg.Params[omnicosmosdb.ParamAuth] = omnicosmosdb.AuthMasterKey
		cfg.Params[omnicosmosdb.ParamEmulator] = "true"
		cfg.Password = cosmosDBEmulatorKey
		cfg.TLS = func(string) (*tls.Config, error) {
			return &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, //nolint:gosec // the emulator signs its certificate itself
			}, nil
		}
	case credential != nil:
		cfg.Params[omnicosmosdb.ParamAuth] = omnicosmosdb.AuthClientSecret
		cfg.Params[omnicosmosdb.ParamTenantID] = credential.TenantId
		cfg.User = credential.ClientId
		cfg.Password = credential.ClientSecret
	default:
		cfg.Params[omnicosmosdb.ParamAuth] = omnicosmosdb.AuthDefault
	}
	return cfg, nil
}

func isLocalhostEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "host.docker.internal"
}

// cosmosdbQuery runs `SELECT * FROM {container} [alias]` in the container the
// query context names. A failed statement is the call's error, not a result.
func cosmosdbQuery(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Container == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("container argument is required for CosmosDB"))
	}
	if _, _, err := base.ValidateSQLForEditor(storepb.Engine_COSMOSDB, statement); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("support simple SELECT statement for Cosmos DB engine only, err: %s", err.Error()))
	}
	results, err := d.QueryEphemeralSession(ctx, statement, cosmosdbQueryOptions(queryContext))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			return nil, connect.NewError(connect.CodeInternal, result.Err)
		}
		converted = append(converted, util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return converted, nil
}

// cosmosdbQueryOptions passes the container as the query's scope: a statement
// reads the container the caller opened, and the language has no form that
// opens another.
func cosmosdbQueryOptions(queryContext db.QueryContext) odriver.QueryOptions {
	return odriver.QueryOptions{
		Schema:         queryContext.Container,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}
}
