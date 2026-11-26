package drivers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"

	// The review check parses with Bytebase's TiDB parser and runs the engine's
	// registered advisors, neither of which this package otherwise imports.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
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

// TestTiDBConfigMapsExtraConnectionParameters pins the connection parameters a
// data source carries reaching the contract's Params, where the omni driver
// appends them to the DSN beside multiStatements and maxAllowedPacket.
func TestTiDBConfigMapsExtraConnectionParameters(t *testing.T) {
	config, err := tidbConfig(newDriver(tidbSpec), db.ConnectionConfig{
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
	})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"readTimeout": "30s", "writeTimeout": "45s"}, config.Params)
	require.Equal(t, "test", config.Database)
	require.Equal(t, "secret", config.Password)
	require.Equal(t, "8.0.11-TiDB-v8.5.5", config.EngineVersion)
	require.Equal(t, "bytebase", config.ApplicationName)
}

// TestTiDBOpenRejectsAllowAllFiles keeps the refusal the old driver made before it
// built a DSN.
func TestTiDBOpenRejectsAllowAllFiles(t *testing.T) {
	driver := newDriver(tidbSpec)
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

// TestTiDBOpenRequiresAUser pins that Open refuses a connection with no login.
func TestTiDBOpenRequiresAUser(t *testing.T) {
	driver := newDriver(tidbSpec)
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

// TestTiDBExecuteHonoursTheTransactionDirective pins that Execute strips Bytebase's
// transaction directive, translates it to omnidriver.TransactionMode, and
// forwards execution callbacks into task run logs.
func TestTiDBExecuteHonoursTheTransactionDirective(t *testing.T) {
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
			driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: stub}}

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

// TestTiDBQueryConnRefusesAZeroResultCap states the adapter's guard: the product
// reads MaximumSQLResultSize 0 as "no rows" and the contract reads it as "no
// cap", so the adapter refuses rather than read an unbounded result.
func TestTiDBQueryConnRefusesAZeroResultCap(t *testing.T) {
	driver := newDriver(tidbSpec)
	_, err := driver.QueryConn(context.Background(), "SELECT 1", db.QueryContext{})
	require.EqualError(t, err, "maximum SQL result size must be positive")
}

// TestTiDBQueryConnAdapterSeam pins QueryConn's held session lifecycle, option
// forwarding, explain plan conversion, and error passthrough.
func TestTiDBQueryConnAdapterSeam(t *testing.T) {
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
		driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: stub}}
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
		driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: &stubTiDBDriver{sessions: []*stubTiDBSession{sess}}}}

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
		driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: &stubTiDBDriver{sessions: []*stubTiDBSession{sess}}}}

		_, err := driver.QueryConn(ctx, "SELECT FROM WHERE", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.ErrorIs(t, err, parseErr)
	})
}

// overlapTiDBSession records whether two statements ran on it at the same time,
// which is what one server session must never do.
type overlapTiDBSession struct {
	inFlight   atomic.Int32
	overlapped atomic.Bool
}

func (s *overlapTiDBSession) Query(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	if s.inFlight.Add(1) > 1 {
		s.overlapped.Store(true)
	}
	time.Sleep(5 * time.Millisecond)
	s.inFlight.Add(-1)
	return nil, nil
}

func (*overlapTiDBSession) Close() error { return nil }

// TestTiDBQueryConnSerializesStatementsOnTheSession pins that the lock covers the
// statement, not just the session's acquisition: two callers sharing a driver
// share one server session, and their statements must not interleave on it.
func TestTiDBQueryConnSerializesStatementsOnTheSession(t *testing.T) {
	session := &overlapTiDBSession{}
	d := newDriver(tidbSpec)
	d.SetSession(session)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_, _ = d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
		})
	}
	wg.Wait()

	require.False(t, session.overlapped.Load(), "two statements ran on the one session at once")
}

// TestTiDBDumpRendersTheSyncedMetadata proves the adapter hands the schema it synced
// to the omni driver's Dumper and propagates dump errors.
func TestTiDBDumpRendersTheSyncedMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantMetadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "shop",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:    "orders",
				Comment: "orders",
				Indexes: []*metadatapb.IndexMetadata{{Name: "orders_note"}},
			}},
		}},
	}
	fake := &fakeTiDBSyncDriver{
		schema:  wantMetadata,
		dumpSQL: "CREATE TABLE `orders` (`id` bigint, `note` varchar(64), KEY `orders_note` (`note`));\n",
	}
	driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: fake}}

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &out, metadata))
	require.Equal(t, fake.dumpSQL, out.String())
	require.Empty(t, cmp.Diff(wantMetadata, fake.dumpedSchema, protocmp.Transform()))

	fake.dumpErr = errors.New("render failed")
	require.ErrorIs(t, driver.Dump(ctx, &out, metadata), fake.dumpErr)
}

type fakeTiDBProbeDriver struct {
	sessions []*recordingTiDBProbeSession
	handle   func(stmt string) ([]*omnidriver.Result, error)
}

func (*fakeTiDBProbeDriver) Ping(context.Context) error { return nil }
func (*fakeTiDBProbeDriver) Close() error               { return nil }

func (f *fakeTiDBProbeDriver) Session(context.Context) (omnidriver.Session, error) {
	sess := &recordingTiDBProbeSession{handle: f.handle}
	f.sessions = append(f.sessions, sess)
	return sess, nil
}

type recordingTiDBProbeSession struct {
	statements  []string
	rollbackCtx context.Context
	closed      bool
	handle      func(stmt string) ([]*omnidriver.Result, error)
}

func (s *recordingTiDBProbeSession) Query(ctx context.Context, script string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	trimmed := strings.TrimSpace(script)
	s.statements = append(s.statements, trimmed)
	if trimmed == "ROLLBACK" {
		s.rollbackCtx = ctx
		return []*omnidriver.Result{{}}, nil
	}
	if trimmed == "START TRANSACTION" {
		return []*omnidriver.Result{{}}, nil
	}
	if s.handle != nil {
		return s.handle(trimmed)
	}
	return []*omnidriver.Result{{}}, nil
}

func (s *recordingTiDBProbeSession) Close() error {
	s.closed = true
	return nil
}

// TestTiDBDryRunRuleProbesThroughTheAdapter pins the advisor-to-adapter seam:
// STATEMENT_DML_DRY_RUN probes through Driver.Probe in a rolled-back
// transaction for regular DML and in autocommit mode for BATCH statements.
func TestTiDBDryRunRuleProbesThroughTheAdapter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for _, tc := range []struct {
		name           string
		statement      string
		probeErr       error
		wantAdvice     bool
		wantAutocommit bool
	}{
		{
			name:      "a dry run the server accepts gives no advice",
			statement: "DELETE FROM app.t WHERE id = 1;",
		},
		{
			name:       "a dry run the server refuses gives advice",
			statement:  "DELETE FROM app.t WHERE missing_column = 1;",
			probeErr:   errors.New("Unknown column 'missing_column' in 'where clause'"),
			wantAdvice: true,
		},
		{
			name:           "a batch dry run outside a transaction gives no advice",
			statement:      "BATCH ON id LIMIT 2 DELETE FROM app.t WHERE id > 0;",
			wantAutocommit: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeTiDBProbeDriver{
				handle: func(_ string) ([]*omnidriver.Result, error) {
					if tc.probeErr != nil {
						//nolint:nilerr // a refused statement's error rides on Result.Err, as the contract specifies.
						return []*omnidriver.Result{{Err: tc.probeErr}}, nil
					}
					return []*omnidriver.Result{{
						Columns: []omnidriver.Column{{Name: "id", TypeName: "VARCHAR"}},
						Rows:    [][]omnidriver.Value{{{Kind: omnidriver.String, String: "Delete_1"}}},
					}}, nil
				},
			}
			driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: fake}}

			adviceList, err := advisor.SQLReviewCheck(ctx, sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: storepb.Engine_TIDB,
			}}, advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Prober:          db.ProberOf(driver),
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			if tc.wantAutocommit {
				// BATCH dry-run probes the BATCH ... DRY RUN statement in autocommit
				// mode and then EXPLAINs the inner DML inside a rolled-back transaction.
				require.Len(t, fake.sessions, 2)
				require.True(t, fake.sessions[0].closed)
				require.Len(t, fake.sessions[0].statements, 1)
				require.Contains(t, fake.sessions[0].statements[0], "DRY RUN")

				require.True(t, fake.sessions[1].closed)
				require.Len(t, fake.sessions[1].statements, 3)
				require.Equal(t, "START TRANSACTION", fake.sessions[1].statements[0])
				require.True(t, strings.HasPrefix(fake.sessions[1].statements[1], "EXPLAIN "))
				require.Equal(t, "ROLLBACK", fake.sessions[1].statements[2])
			} else {
				require.Len(t, fake.sessions, 1)
				require.True(t, fake.sessions[0].closed)
				require.Len(t, fake.sessions[0].statements, 3)
				require.Equal(t, "START TRANSACTION", fake.sessions[0].statements[0])
				require.True(t, strings.HasPrefix(fake.sessions[0].statements[1], "EXPLAIN "))
				require.Equal(t, "ROLLBACK", fake.sessions[0].statements[2])
			}

			if !tc.wantAdvice {
				require.Empty(t, adviceList)
				return
			}
			require.Len(t, adviceList, 1)
			require.Contains(t, adviceList[0].GetContent(), "dry runs failed")
		})
	}
}

// TestTiDBProbeHonoursAutocommit pins transactional pre-executions, rollback on a
// canceled context, autocommit validation, and tidbProbeCell timestamp formatting.
func TestTiDBProbeHonoursAutocommit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ts := time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.UTC)
	tsNoFrac := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	fake := &fakeTiDBProbeDriver{
		handle: func(stmt string) ([]*omnidriver.Result, error) {
			if strings.HasPrefix(stmt, "INSERT ") {
				return []*omnidriver.Result{{}}, nil
			}
			return []*omnidriver.Result{{
				Columns: []omnidriver.Column{
					{Name: "cnt", TypeName: "BIGINT"},
					{Name: "ts3", TypeName: "DATETIME"},
					{Name: "ts0", TypeName: "TIMESTAMP"},
					{Name: "nul", TypeName: "VARCHAR"},
				},
				Rows: [][]omnidriver.Value{{
					{Kind: omnidriver.Int64, Int64: 1},
					{Kind: omnidriver.Timestamp, Time: ts, Precision: 3},
					{Kind: omnidriver.TimestampTZ, Time: tsNoFrac, Precision: 0},
					{Kind: omnidriver.Null},
				}},
			}}, nil
		},
	}
	driver := &Driver{spec: tidbSpec, OmniAdapter: util.OmniAdapter{Driver: fake}}
	prober := db.ProberOf(driver)

	// Transactional probe skips empty pre-executions, runs non-empty ones,
	// formats temporal cells via tidbProbeCell, and rolls back with a detached ctx.
	probeCtx, cancel := context.WithCancel(ctx)
	cancel()
	probe, err := prober.Probe(probeCtx, db.ProbeContext{
		PreExecutions: []string{"", "INSERT INTO app.t VALUES (42)"},
	}, "SELECT COUNT(*) FROM t")
	require.NoError(t, err)
	require.Equal(t, []string{"cnt", "ts3", "ts0", "nul"}, probe.Columns)
	require.Equal(t, [][]any{{int64(1), "2026-03-04 05:06:07.123", "2026-03-04 05:06:07", nil}}, probe.Rows)
	require.Len(t, fake.sessions, 1)
	require.Equal(t, []string{
		"START TRANSACTION",
		"INSERT INTO app.t VALUES (42)",
		"SELECT COUNT(*) FROM t",
		"ROLLBACK",
	}, fake.sessions[0].statements)
	require.NoError(t, fake.sessions[0].rollbackCtx.Err(), "rollback must ignore caller cancellation")

	// Autocommit runs the statement directly without opening a transaction.
	_, err = prober.Probe(ctx, db.ProbeContext{Autocommit: true}, "BATCH ON id LIMIT 2 DRY RUN DELETE FROM app.t WHERE id > 0")
	require.NoError(t, err)
	require.Len(t, fake.sessions, 2)
	require.Equal(t, []string{"BATCH ON id LIMIT 2 DRY RUN DELETE FROM app.t WHERE id > 0"}, fake.sessions[1].statements)

	// Pre-executions cannot run outside a transaction.
	_, err = prober.Probe(ctx, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET @x = 1"}}, "SELECT 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "pre-executions cannot run outside a transaction")
	require.Len(t, fake.sessions, 2, "invalid probe must fail before opening a session")
}

// TestTiDBProbeReachesTheRowEstimator pins that a row-limit rule holding the
// probe finds the omni TiDB driver's estimate through it.
func TestTiDBProbeReachesTheRowEstimator(t *testing.T) {
	ctx := context.Background()
	opened, err := newDriver(tidbSpec).Open(ctx, storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "test"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, opened.Close(ctx)) })

	_, ok := db.As[db.RowEstimator](db.ProberOf(opened))
	require.True(t, ok)
}

type fakeTiDBSyncDriver struct {
	instance     *omnidriver.Instance
	schema       *metadatapb.DatabaseSchemaMetadata
	syncDBErr    error
	dumpSQL      string
	dumpErr      error
	dumpedSchema *metadatapb.DatabaseSchemaMetadata
}

func (*fakeTiDBSyncDriver) Ping(context.Context) error { return nil }
func (*fakeTiDBSyncDriver) Close() error               { return nil }

func (f *fakeTiDBSyncDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeTiDBSyncDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	if f.syncDBErr != nil {
		return nil, f.syncDBErr
	}
	return f.schema, nil
}

func (f *fakeTiDBSyncDriver) Dump(_ context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	f.dumpedSchema = metadata
	if f.dumpErr != nil {
		return f.dumpErr
	}
	_, err := io.WriteString(out, f.dumpSQL)
	return err
}

// TestTiDBSyncDBSchemaPassesTheDriversMetadataThrough proves the adapter returns
// omni's synced database schema unchanged.
func TestTiDBSyncDBSchemaPassesTheDriversMetadataThrough(t *testing.T) {
	t.Parallel()

	want := &metadatapb.DatabaseSchemaMetadata{
		Name: "shop",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name: "orders",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int"},
					{Name: "note", Type: "varchar(64)"},
				},
			}},
		}},
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeTiDBSyncDriver{schema: want}}}

	got, err := driver.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, got, protocmp.Transform()))
}

// TestTiDBSyncDBSchemaReportsAMissingDatabase pins the adapter's translation of
// omni's ErrNotFound into Bytebase's common.NotFound error.
func TestTiDBSyncDBSchemaReportsAMissingDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := &fakeTiDBSyncDriver{syncDBErr: omnidriver.ErrNotFound}
	driver := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:           fake,
			NotFoundResource: "database",
			DatabaseName:     "shop_missing",
		},
	}

	_, err := driver.SyncDBSchema(ctx)
	require.Error(t, err)
	require.Equal(t, common.NotFound, common.ErrorCode(err))
	require.Contains(t, err.Error(), `database "shop_missing" not found`)

	otherErr := errors.New("connection refused")
	fake.syncDBErr = otherErr
	_, err = driver.SyncDBSchema(ctx)
	require.ErrorIs(t, err, otherErr)
}

// TestTiDBSyncInstanceMapsTheDriversInstance pins how the adapter maps omni's
// instance version, databases, roles, and lower_case_table_names onto
// Bytebase's InstanceMetadata.
func TestTiDBSyncInstanceMapsTheDriversInstance(t *testing.T) {
	t.Parallel()

	grantAttr := "GRANT SELECT ON `shop`.* TO 'bb_sync_probe'@'%'"
	contract := &omnidriver.Instance{
		Version:             "8.0.11-TiDB-v8.5.5",
		LowerCaseTableNames: 2,
		Databases: []*metadatapb.DatabaseSchemaMetadata{
			{Name: "shop", CharacterSet: "utf8mb4", Collation: "utf8mb4_bin"},
		},
		Roles: []*omnidriver.Role{
			{Name: "'bb_sync_probe'@'%'", Attribute: &grantAttr},
		},
	}
	driver := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:               &fakeTiDBSyncDriver{instance: contract},
			WithInstanceMetadata: true,
		},
	}

	instance, err := driver.SyncInstance(context.Background())
	require.NoError(t, err)
	require.Equal(t, contract.Version, instance.Version)
	require.NotNil(t, instance.Metadata)
	require.Equal(t, int32(2), instance.Metadata.GetMysqlLowerCaseTableNames())
	require.Empty(t, cmp.Diff(contract.Databases, instance.Databases, protocmp.Transform()))
	require.Empty(t, cmp.Diff([]*storepb.InstanceRole{{
		Name:      "'bb_sync_probe'@'%'",
		Attribute: &grantAttr,
	}}, instance.Metadata.GetRoles(), protocmp.Transform()))
}
