// Copyright (c) Bytebase (Hong Kong) Limited.
package cosmosdb

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

type stubCosmosDriver struct {
	session *stubCosmosSession
}

func (*stubCosmosDriver) Ping(context.Context) error { return nil }
func (*stubCosmosDriver) Close() error               { return nil }
func (d *stubCosmosDriver) Session(context.Context) (omnidriver.Session, error) {
	return d.session, nil
}

type stubCosmosSession struct {
	gotStatement string
	gotOpts      omnidriver.QueryOptions
	results      []*omnidriver.Result
	err          error
	closed       bool
}

func (s *stubCosmosSession) Query(_ context.Context, statement string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotStatement = statement
	s.gotOpts = opts
	return s.results, s.err
}

func (s *stubCosmosSession) Close() error {
	s.closed = true
	return nil
}

func TestConnectionConfigEmulatorInDev(t *testing.T) {
	require.True(t, common.IsDev(), "test builds run in dev mode")

	for _, endpoint := range []string{
		"https://localhost:8081",
		"https://127.0.0.1:8081",
		"https://host.docker.internal:8081",
	} {
		cfg, err := connectionConfig(&storepb.DataSource{Host: endpoint}, "testdb")
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

func TestQueryConnAdapterSeam(t *testing.T) {
	t.Run("rejects invalid SQL before opening a session", func(t *testing.T) {
		driver := &Driver{}
		_, err := driver.QueryConn(context.Background(), "SELECT FROM WHERE", db.QueryContext{Container: "cities"})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("forwards options and converts result rows", func(t *testing.T) {
		sess := &stubCosmosSession{
			results: []*omnidriver.Result{{
				Statement: `SELECT c.name FROM c`,
				Columns:   []omnidriver.Column{{Name: "result", TypeName: "TEXT"}},
				Rows: [][]omnidriver.Value{
					{{Kind: omnidriver.String, String: `{"name":"city-1"}`}},
				},
			}},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubCosmosDriver{session: sess}}}

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
		sess := &stubCosmosSession{
			results: []*omnidriver.Result{{
				Statement: `SELECT * FROM c`,
				Err:       errors.New("container not found"),
			}},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubCosmosDriver{session: sess}}}

		results, err := driver.QueryConn(context.Background(), `SELECT * FROM c`, db.QueryContext{Container: "missing"})
		require.Nil(t, results)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.ErrorContains(t, err, "container not found")
	})
}
