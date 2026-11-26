package drivers

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	omnicosmosdb "github.com/bytebase/omni/cosmosdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"

	// Registers the CosmosDB query validator that QueryConn calls.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cosmosdb"
)

type fakeCosmosDBDriver struct {
	session *fakeCosmosDBSession
}

func (*fakeCosmosDBDriver) Ping(context.Context) error { return nil }
func (*fakeCosmosDBDriver) Close() error               { return nil }
func (d *fakeCosmosDBDriver) Session(context.Context) (omnidriver.Session, error) {
	return d.session, nil
}

type fakeCosmosDBSession struct {
	gotStatement string
	gotOpts      omnidriver.QueryOptions
	results      []*omnidriver.Result
	err          error
	closed       bool
}

func (s *fakeCosmosDBSession) Query(_ context.Context, statement string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotStatement = statement
	s.gotOpts = opts
	return s.results, s.err
}

func (s *fakeCosmosDBSession) Close() error {
	s.closed = true
	return nil
}

func TestCosmosDBConnectionConfigMapsTheCredential(t *testing.T) {
	secret, err := cosmosdbConfig(&storepb.DataSource{
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

	fallback, err := cosmosdbConfig(&storepb.DataSource{Host: "https://account.documents.azure.com:443/"}, "")
	require.NoError(t, err)
	require.Equal(t, omnicosmosdb.AuthDefault, fallback.Params[omnicosmosdb.ParamAuth])
	require.Empty(t, fallback.Password)

	_, err = cosmosdbConfig(nil, "")
	require.ErrorContains(t, err, "a data source is required")
}

func TestCosmosDBConnectionConfigEmulatorInDev(t *testing.T) {
	require.True(t, common.IsDev(), "test builds run in dev mode")

	for _, endpoint := range []string{
		"https://localhost:8081",
		"https://127.0.0.1:8081",
		"https://host.docker.internal:8081",
	} {
		cfg, err := cosmosdbConfig(&storepb.DataSource{Host: endpoint}, "testdb")
		require.NoError(t, err, endpoint)
		require.Equal(t, omnicosmosdb.AuthMasterKey, cfg.Params[omnicosmosdb.ParamAuth])
		require.Equal(t, "true", cfg.Params[omnicosmosdb.ParamEmulator])
		require.Equal(t, cosmosDBEmulatorKey, cfg.Password)
		require.NotNil(t, cfg.TLS)
		tlsCfg, err := cfg.TLS("localhost")
		require.NoError(t, err)
		require.True(t, tlsCfg.InsecureSkipVerify)
	}

	require.False(t, isLocalhostEndpoint("https://account.documents.azure.com:443/"))
	require.False(t, isLocalhostEndpoint("://bad-url"))
}

// TestCosmosDBAdapterBoundaryWithoutAnAccount covers what the adapter answers
// before it reaches an account — no pool, no Executor — and the query argument
// check the API depends on.
func TestCosmosDBAdapterBoundaryWithoutAnAccount(t *testing.T) {
	driver := newDriver(cosmosdbSpec)

	_, err := driver.Execute(context.Background(), "SELECT * FROM c", db.ExecuteOptions{})
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))

	_, err = driver.QueryConn(context.Background(), "SELECT * FROM c", db.QueryContext{})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a container is required")
}

func TestCosmosDBQueryOptionsCarriesTheRequest(t *testing.T) {
	options := cosmosdbQueryOptions(db.QueryContext{
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

func TestCosmosDBQueryConnSeam(t *testing.T) {
	ctx := context.Background()
	session := &fakeCosmosDBSession{
		results: []*omnidriver.Result{{
			Statement: "SELECT * FROM c",
			Columns:   []omnidriver.Column{{Name: "c", TypeName: "JSON"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.String, String: `{"id":"1"}`}}},
		}},
	}
	d := &Driver{spec: cosmosdbSpec, OmniAdapter: util.OmniAdapter{Driver: &fakeCosmosDBDriver{session: session}}}

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

	session.results = []*omnidriver.Result{{Statement: "SELECT * FROM c", Err: errors.New("gateway error")}}
	_, err = d.QueryConn(ctx, "SELECT * FROM c", db.QueryContext{Container: "orders"})
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.ErrorContains(t, err, "gateway error")
}

func TestCosmosDBQueryConnAdapterSeam(t *testing.T) {
	t.Run("rejects invalid SQL before opening a session", func(t *testing.T) {
		driver := newDriver(cosmosdbSpec)
		_, err := driver.QueryConn(context.Background(), "SELECT FROM WHERE", db.QueryContext{Container: "cities"})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("forwards options and converts result rows", func(t *testing.T) {
		sess := &fakeCosmosDBSession{
			results: []*omnidriver.Result{{
				Statement: `SELECT c.name FROM c`,
				Columns:   []omnidriver.Column{{Name: "result", TypeName: "TEXT"}},
				Rows: [][]omnidriver.Value{
					{{Kind: omnidriver.String, String: `{"name":"city-1"}`}},
				},
			}},
		}
		driver := &Driver{spec: cosmosdbSpec, OmniAdapter: util.OmniAdapter{Driver: &fakeCosmosDBDriver{session: sess}}}

		results, err := driver.QueryConn(context.Background(), `SELECT c.name FROM c`, db.QueryContext{
			Container:            "cities",
			Limit:                7,
			MaximumSQLResultSize: 4096,
		})
		require.NoError(t, err)
		require.True(t, sess.closed)
		require.Equal(t, `SELECT c.name FROM c`, sess.gotStatement)
		require.Equal(t, omnidriver.QueryOptions{
			Schema:         "cities",
			Limit:          7,
			MaxResultBytes: 4096,
		}, sess.gotOpts)
		require.Len(t, results, 1)
		require.Equal(t, `SELECT c.name FROM c`, results[0].Statement)
		require.Len(t, results[0].Rows, 1)
		require.Equal(t, `{"name":"city-1"}`, results[0].Rows[0].GetValues()[0].GetStringValue())
	})

	t.Run("surfaces statement error as internal error", func(t *testing.T) {
		sess := &fakeCosmosDBSession{
			results: []*omnidriver.Result{{
				Statement: `SELECT * FROM c`,
				Err:       errors.New("container not found"),
			}},
		}
		driver := &Driver{spec: cosmosdbSpec, OmniAdapter: util.OmniAdapter{Driver: &fakeCosmosDBDriver{session: sess}}}

		results, err := driver.QueryConn(context.Background(), `SELECT * FROM c`, db.QueryContext{Container: "missing"})
		require.Nil(t, results)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.ErrorContains(t, err, "container not found")
	})
}
