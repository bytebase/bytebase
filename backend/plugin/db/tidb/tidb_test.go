package tidb

import (
	"context"
	"errors"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type stubTiDBDriver struct {
	sessionOpens int
	sessions     []*stubTiDBSession

	execScript string
	execOpts   omnidriver.ExecuteOptions
	execFn     func(context.Context, string, omnidriver.ExecuteOptions) (int64, error)
}

func (*stubTiDBDriver) Ping(context.Context) error { return nil }
func (*stubTiDBDriver) Close() error               { return nil }

func (d *stubTiDBDriver) Session(context.Context) (omnidriver.Session, error) {
	d.sessionOpens++
	if len(d.sessions) > 0 {
		s := d.sessions[0]
		d.sessions = d.sessions[1:]
		return s, nil
	}
	return &stubTiDBSession{}, nil
}

func (d *stubTiDBDriver) Execute(ctx context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	d.execScript = script
	d.execOpts = opts
	if d.execFn != nil {
		return d.execFn(ctx, script, opts)
	}
	return 0, nil
}

type stubTiDBSession struct {
	queryFn func(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error)
	closed  bool
}

func (s *stubTiDBSession) Query(ctx context.Context, statement string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	if s.queryFn != nil {
		return s.queryFn(ctx, statement, opts)
	}
	return []*omnidriver.Result{{Statement: statement}}, nil
}

func (s *stubTiDBSession) Close() error {
	s.closed = true
	return nil
}

// TestBuildConfigMapsExtraConnectionParameters pins the connection parameters a
// data source carries reaching the contract's Params, where the omni driver
// appends them to the DSN beside multiStatements and maxAllowedPacket.
func TestBuildConfigMapsExtraConnectionParameters(t *testing.T) {
	config, err := buildConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
			ExtraConnectionParameters: map[string]string{
				"readTimeout":  "30s",
				"writeTimeout": "45s",
			},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName:  "test",
			EngineVersion: "8.0.11-TiDB-v8.5.5",
		},
	}, &util.OmniAdapter{})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"readTimeout": "30s", "writeTimeout": "45s"}, config.Params)
	require.Equal(t, "test", config.Database)
	require.Equal(t, "secret", config.Password)
	require.Equal(t, "8.0.11-TiDB-v8.5.5", config.EngineVersion)
	require.Equal(t, "bytebase", config.ApplicationName)
}

// TestOpenRejectsAllowAllFiles keeps the refusal the old driver made before it
// built a DSN.
func TestOpenRejectsAllowAllFiles(t *testing.T) {
	driver := newDriver()
	_, err := driver.Open(context.Background(), storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
			ExtraConnectionParameters: map[string]string{
				"allowAllFiles": "true",
			},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "test",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowAllFiles")
}

// TestOpenRequiresAUser pins that Open refuses a connection with no login.
func TestOpenRequiresAUser(t *testing.T) {
	driver := newDriver()
	_, err := driver.Open(context.Background(), storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host: "127.0.0.1",
			Port: "4000",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "test"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "user must be set")
}

// TestExecuteHonoursTheTransactionDirective pins that Execute strips Bytebase's
// transaction directive, translates it to omnidriver.TransactionMode, and
// forwards execution callbacks into task run logs.
func TestExecuteHonoursTheTransactionDirective(t *testing.T) {
	for _, tc := range []struct {
		name         string
		script       string
		wantMode     omnidriver.TransactionMode
		wantCleaned  string
		wantLogTypes []storepb.TaskRunLog_Type
	}{
		{
			name:        "default transactional",
			script:      "INSERT INTO t VALUES (1);",
			wantMode:    omnidriver.Transactional,
			wantCleaned: "INSERT INTO t VALUES (1);",
			wantLogTypes: []storepb.TaskRunLog_Type{
				storepb.TaskRunLog_TRANSACTION_CONTROL,
				storepb.TaskRunLog_COMMAND_EXECUTE,
				storepb.TaskRunLog_COMMAND_RESPONSE,
				storepb.TaskRunLog_TRANSACTION_CONTROL,
			},
		},
		{
			name:        "txn-mode off switches to auto-commit",
			script:      "-- txn-mode = off\nINSERT INTO t VALUES (1);",
			wantMode:    omnidriver.AutoCommit,
			wantCleaned: "INSERT INTO t VALUES (1);",
			wantLogTypes: []storepb.TaskRunLog_Type{
				storepb.TaskRunLog_COMMAND_EXECUTE,
				storepb.TaskRunLog_COMMAND_RESPONSE,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubTiDBDriver{
				execFn: func(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
					if opts.Transaction == omnidriver.Transactional {
						opts.Log.TransactionControl(omnidriver.Begin, nil)
					}
					opts.Log.CommandExecute(omnidriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
					opts.Log.CommandResponse(1, []int64{1}, nil)
					if opts.Transaction == omnidriver.Transactional {
						opts.Log.TransactionControl(omnidriver.Commit, nil)
					}
					return 1, nil
				},
			}
			driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: stub}}

			var logs []*storepb.TaskRunLog
			affected, err := driver.Execute(context.Background(), tc.script, db.ExecuteOptions{
				MaximumRetries: 3,
				CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
					logs = append(logs, entry)
					return nil
				},
			})
			require.NoError(t, err)
			require.Equal(t, int64(1), affected)
			require.Equal(t, tc.wantMode, stub.execOpts.Transaction)
			require.Equal(t, 3, stub.execOpts.MaxRetries)
			require.Equal(t, tc.wantCleaned, stub.execScript)

			gotTypes := make([]storepb.TaskRunLog_Type, len(logs))
			for i, entry := range logs {
				gotTypes[i] = entry.Type
			}
			require.Equal(t, tc.wantLogTypes, gotTypes)
		})
	}
}

// TestQueryConnRefusesAZeroResultCap states the adapter's guard: the product
// reads MaximumSQLResultSize 0 as "no rows" and the contract reads it as "no
// cap", so the adapter refuses rather than read an unbounded result.
func TestQueryConnRefusesAZeroResultCap(t *testing.T) {
	driver := &Driver{}
	_, err := driver.QueryConn(context.Background(), "SELECT 1", db.QueryContext{})
	require.EqualError(t, err, "maximum SQL result size must be positive")
}

// TestQueryConnAdapterSeam pins QueryConn's held session lifecycle, option
// forwarding, explain plan conversion, and error passthrough.
func TestQueryConnAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("reuses one session and discards it on canceled context", func(t *testing.T) {
		var seenOpts []omnidriver.QueryOptions
		sess1 := &stubTiDBSession{
			queryFn: func(ctx context.Context, statement string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				seenOpts = append(seenOpts, opts)
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return []*omnidriver.Result{{
					Statement: statement,
					Columns:   []omnidriver.Column{{Name: "v", TypeName: "BIGINT"}},
					Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int64, Int64: 42}}},
				}}, nil
			},
		}
		stub := &stubTiDBDriver{sessions: []*stubTiDBSession{sess1}}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: stub}}
		qctx := db.QueryContext{Limit: 100, MaximumSQLResultSize: 1 << 20}

		_, err := driver.QueryConn(ctx, "SET @p = 42", qctx)
		require.NoError(t, err)
		res, err := driver.QueryConn(ctx, "SELECT @p", qctx)
		require.NoError(t, err)
		require.Equal(t, 1, stub.sessionOpens)
		require.Same(t, sess1, driver.Session())
		require.Equal(t, int64(42), res[0].GetRows()[0].GetValues()[0].GetInt64Value())

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err = driver.QueryConn(canceledCtx, "SELECT 1", qctx)
		require.ErrorIs(t, err, context.Canceled)
		require.True(t, sess1.closed)
		require.Nil(t, driver.Session())
	})

	t.Run("maps explain format and suppresses typed plan on explain requests", func(t *testing.T) {
		var gotOpts omnidriver.QueryOptions
		sess := &stubTiDBSession{
			queryFn: func(_ context.Context, statement string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				gotOpts = opts
				return []*omnidriver.Result{{
					Statement: statement,
					Plan:      &omnidriver.Plan{Format: "JSON", Executed: false},
				}}, nil
			},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubTiDBDriver{sessions: []*stubTiDBSession{sess}}}}

		explainRes, err := driver.QueryConn(ctx, "SELECT 1", db.QueryContext{
			Explain:              true,
			MaximumSQLResultSize: 1 << 20,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Equal(t, "JSON", gotOpts.ExplainFormat)
		require.True(t, gotOpts.Explain)
		require.Nil(t, explainRes[0].GetQueryPlan(), "explain requests omit typed QueryPlan")

		ranRes, err := driver.QueryConn(ctx, "EXPLAIN FORMAT=tidb_json SELECT 1", db.QueryContext{
			Explain:              false,
			MaximumSQLResultSize: 1 << 20,
		})
		require.NoError(t, err)
		require.Equal(t, "TEXT", gotOpts.ExplainFormat)
		require.Equal(t, v1pb.QueryOption_JSON, ranRes[0].GetQueryPlan().GetFormat())
		require.False(t, ranRes[0].GetQueryPlan().GetExecuted())
	})

	t.Run("passes parser error through unchanged", func(t *testing.T) {
		parseErr := errors.New(`syntax error at or near "FROM" (line 1, column 8)`)
		sess := &stubTiDBSession{
			queryFn: func(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				return nil, parseErr
			},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubTiDBDriver{sessions: []*stubTiDBSession{sess}}}}

		_, err := driver.QueryConn(ctx, "SELECT FROM WHERE", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.ErrorIs(t, err, parseErr)
	})
}
