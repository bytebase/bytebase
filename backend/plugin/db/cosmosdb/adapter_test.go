package cosmosdb

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	omnicosmosdb "github.com/bytebase/omni/cosmosdb/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestConnectionConfigMapsTheCredential pins the adapter's half of Open: an app
// registration becomes a client-secret credential, and a data source without
// one falls back to azidentity's default chain.
func TestConnectionConfigMapsTheCredential(t *testing.T) {
	secret, err := connectionConfig(&storepb.DataSource{
		Host: "https://account.documents.azure.com:443/",
		IamExtension: &storepb.DataSource_AzureCredential_{
			AzureCredential: &storepb.DataSource_AzureCredential{
				TenantId:     "tenant",
				ClientId:     "client",
				ClientSecret: "secret",
			},
		},
	}, "sales")
	require.NoError(t, err)
	require.Equal(t, "sales", secret.Database)
	require.Equal(t, omnicosmosdb.AuthClientSecret, secret.Params[omnicosmosdb.ParamAuth])
	require.Equal(t, "tenant", secret.Params[omnicosmosdb.ParamTenantID])
	require.Equal(t, "client", secret.User)
	require.Equal(t, "secret", secret.Password)

	fallback, err := connectionConfig(&storepb.DataSource{Host: "https://account.documents.azure.com:443/"}, "")
	require.NoError(t, err)
	require.Equal(t, omnicosmosdb.AuthDefault, fallback.Params[omnicosmosdb.ParamAuth])
	require.Empty(t, fallback.Password)

	_, err = connectionConfig(nil, "")
	require.ErrorContains(t, err, "a data source is required")
}

// TestAdapterBoundaryWithoutAnAccount covers what the adapter answers before it
// reaches an account — no pool, no Executor — and the query argument check the
// API depends on.
func TestAdapterBoundaryWithoutAnAccount(t *testing.T) {
	driver := &Driver{}
	require.Nil(t, driver.GetDB(), "Cosmos DB has no database/sql handle")

	_, err := driver.Execute(context.Background(), "SELECT * FROM c", db.ExecuteOptions{})
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))

	_, err = driver.QueryConn(context.Background(), nil, "SELECT * FROM c", db.QueryContext{})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a container is required")
}

// TestQueryOptionsCarriesTheRequest pins the mapping QueryConn hands the driver:
// the container is the statement's scope, and the row limit and the result-size
// cap the query policy sets bind what the driver reads back.
func TestQueryOptionsCarriesTheRequest(t *testing.T) {
	options := queryOptions(db.QueryContext{
		Container:            "orders",
		Limit:                25,
		Explain:              true,
		MaximumSQLResultSize: 2048,
	})
	require.Equal(t, "orders", options.Schema)
	require.Equal(t, 25, options.Limit)
	require.True(t, options.Explain)
	require.Equal(t, int64(2048), options.MaxResultBytes)
}
