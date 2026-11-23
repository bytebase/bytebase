package cosmosdb

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	omnicosmosdb "github.com/bytebase/omni/cosmosdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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

	_, err := driver.Execute(context.Background(), "SELECT * FROM c", db.ExecuteOptions{})
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))

	_, err = driver.QueryConn(context.Background(), "SELECT * FROM c", db.QueryContext{})
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

type fakeCosmosDriver struct {
	session omnidriver.Session
}

func (*fakeCosmosDriver) Ping(context.Context) error { return nil }
func (*fakeCosmosDriver) Close() error               { return nil }
func (f *fakeCosmosDriver) Session(context.Context) (omnidriver.Session, error) {
	return f.session, nil
}

type fakeCosmosSession struct {
	gotOpts omnidriver.QueryOptions
	results []*omnidriver.Result
	err     error
}

func (s *fakeCosmosSession) Query(_ context.Context, _ string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotOpts = opts
	return s.results, s.err
}

func (*fakeCosmosSession) Close() error { return nil }

func TestQueryConnSeam(t *testing.T) {
	ctx := context.Background()
	session := &fakeCosmosSession{
		results: []*omnidriver.Result{{
			Statement: "SELECT * FROM c",
			Columns:   []omnidriver.Column{{Name: "c", TypeName: "JSON"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.String, String: `{"id":"1"}`}}},
		}},
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeCosmosDriver{session: session}}}

	results, err := d.QueryConn(ctx, "SELECT * FROM c", db.QueryContext{
		Container:            "orders",
		Limit:                10,
		MaximumSQLResultSize: 1024,
	})
	require.NoError(t, err)
	require.Equal(t, "orders", session.gotOpts.Schema)
	require.Equal(t, 10, session.gotOpts.Limit)
	require.Equal(t, int64(1024), session.gotOpts.MaxResultBytes)
	require.Len(t, results, 1)
	require.Equal(t, `{"id":"1"}`, results[0].Rows[0].Values[0].GetStringValue())

	// A per-statement result error surfaces as connect.CodeInternal.
	session.results = []*omnidriver.Result{{Statement: "SELECT * FROM c", Err: errors.New("gateway error")}}
	_, err = d.QueryConn(ctx, "SELECT * FROM c", db.QueryContext{Container: "orders"})
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.ErrorContains(t, err, "gateway error")
}
