package mysql

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	mariadbd "github.com/bytebase/omni/mariadb/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	mysqld "github.com/bytebase/omni/mysql/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type fakeMySQLDriver struct {
	pinged       bool
	sessionOpens int
	sessions     []*fakeMySQLSession

	execScript string
	execOpts   odriver.ExecuteOptions
	execFn     func(context.Context, string, odriver.ExecuteOptions) (int64, error)

	instance    *odriver.Instance
	schema      *metadatapb.DatabaseSchemaMetadata
	dumpDDL     string
	estimate    int64
	estimateSQL string
	explainJSON string
	explainSQL  string
}

func (f *fakeMySQLDriver) Ping(context.Context) error {
	f.pinged = true
	return nil
}

func (*fakeMySQLDriver) Close() error { return nil }

func (f *fakeMySQLDriver) Session(context.Context) (odriver.Session, error) {
	f.sessionOpens++
	if len(f.sessions) > 0 {
		s := f.sessions[0]
		f.sessions = f.sessions[1:]
		return s, nil
	}
	return &fakeMySQLSession{}, nil
}

func (f *fakeMySQLDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeMySQLDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (f *fakeMySQLDriver) Dump(_ context.Context, out io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(out, f.dumpDDL)
	return err
}

func (f *fakeMySQLDriver) Execute(ctx context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	if f.execFn != nil {
		return f.execFn(ctx, script, opts)
	}
	return 1, nil
}

func (f *fakeMySQLDriver) CountAffectedRows(_ context.Context, statement string) (int64, error) {
	f.estimateSQL = statement
	return f.estimate, nil
}

func (f *fakeMySQLDriver) ExplainJSON(_ context.Context, statement string) (string, error) {
	f.explainSQL = statement
	return f.explainJSON, nil
}

type fakeMySQLSession struct {
	statements []string
	lastOpts   odriver.QueryOptions
	queryFn    func(context.Context, string, odriver.QueryOptions) ([]*odriver.Result, error)
	closed     bool
}

func (s *fakeMySQLSession) Query(ctx context.Context, statement string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.statements = append(s.statements, statement)
	s.lastOpts = opts
	if s.queryFn != nil {
		return s.queryFn(ctx, statement, opts)
	}
	return []*odriver.Result{{Statement: statement}}, nil
}

func (s *fakeMySQLSession) Close() error {
	s.closed = true
	return nil
}

func TestAdapterSyncInstanceAndDumpSeam(t *testing.T) {
	ctx := context.Background()
	schema := &metadatapb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{Name: "t"}},
		}},
	}
	fake := &fakeMySQLDriver{
		instance: &odriver.Instance{
			Version:             "8.0.46",
			LowerCaseTableNames: 1,
			Databases:           []*metadatapb.DatabaseSchemaMetadata{{Name: "appdb"}},
		},
		schema:  schema,
		dumpDDL: "CREATE TABLE `t` (`id` INT PRIMARY KEY);",
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake, WithInstanceMetadata: true}}

	require.NoError(t, d.Ping(ctx))
	require.True(t, fake.pinged)

	instance, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "8.0.46", instance.Version)
	require.NotNil(t, instance.Metadata)
	require.Equal(t, int32(1), instance.Metadata.GetMysqlLowerCaseTableNames())
	require.Len(t, instance.Databases, 1)

	metadata, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, metadata.Schemas, 1)
	require.Equal(t, "t", metadata.Schemas[0].Tables[0].Name)

	var dump bytes.Buffer
	require.NoError(t, d.Dump(ctx, &dump, metadata))
	require.Contains(t, dump.String(), "CREATE TABLE `t`")
}

func TestAdapterQueryConnSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("holds one session across calls and converts rows and masking counts", func(t *testing.T) {
		zero := int64(0)
		sess := &fakeMySQLSession{
			queryFn: func(_ context.Context, statement string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
				if opts.MaskingEnabled {
					return []*odriver.Result{{Statement: statement, AffectedRows: &zero}}, nil
				}
				return []*odriver.Result{{
					Statement: statement,
					Columns:   []odriver.Column{{Name: "id", TypeName: "INT"}, {Name: "note", TypeName: "VARCHAR"}},
					Rows: [][]odriver.Value{{
						{Kind: odriver.Int64, Int64: 1},
						{Kind: odriver.String, String: "one"},
					}},
				}}, nil
			},
		}
		fake := &fakeMySQLDriver{sessions: []*fakeMySQLSession{sess}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		results, err := d.QueryConn(ctx, "SELECT id, note FROM qitems ORDER BY id", db.QueryContext{
			Limit:                10,
			MaximumSQLResultSize: 1 << 20,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Equal(t, 1, fake.sessionOpens)
		require.Same(t, sess, d.Session())
		require.Equal(t, 10, sess.lastOpts.Limit)
		require.Equal(t, "JSON", sess.lastOpts.ExplainFormat)
		require.Equal(t, int64(1<<20), sess.lastOpts.MaxResultBytes)
		require.Len(t, results, 1)
		require.Equal(t, []string{"id", "note"}, results[0].ColumnNames)
		require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
		require.Equal(t, "one", results[0].Rows[0].Values[1].GetStringValue())

		// A zero size cap maps to 1 byte so the contract bounds the result, and
		// MaskingEnabled forwards to the session.
		masked, err := d.QueryConn(ctx, "CALL p()", db.QueryContext{MaskingEnabled: true})
		require.NoError(t, err)
		require.Equal(t, 1, fake.sessionOpens)
		require.True(t, sess.lastOpts.MaskingEnabled)
		require.Equal(t, int64(1), sess.lastOpts.MaxResultBytes)
		require.Len(t, masked, 1)
		require.Equal(t, []string{"Affected Rows"}, masked[0].ColumnNames)
		require.Equal(t, int64(0), masked[0].Rows[0].Values[0].GetInt64Value())
	})

	t.Run("discards the held session after a canceled query", func(t *testing.T) {
		sess1 := &fakeMySQLSession{
			queryFn: func(ctx context.Context, statement string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return []*odriver.Result{{Statement: statement}}, nil
			},
		}
		sess2 := &fakeMySQLSession{
			queryFn: func(_ context.Context, statement string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				return []*odriver.Result{{
					Statement: statement,
					Columns:   []odriver.Column{{Name: "1", TypeName: "BIGINT"}},
					Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
				}}, nil
			},
		}
		fake := &fakeMySQLDriver{sessions: []*fakeMySQLSession{sess1, sess2}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
		qctx := db.QueryContext{MaximumSQLResultSize: 1 << 20}

		_, err := d.QueryConn(ctx, "SET @session_marker = 41", qctx)
		require.NoError(t, err)

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err = d.QueryConn(canceledCtx, "SELECT SLEEP(10)", qctx)
		require.ErrorIs(t, err, context.Canceled)
		require.True(t, sess1.closed)
		require.Nil(t, d.Session(), "a canceled statement's session is not one to keep")

		results, err := d.QueryConn(ctx, "SELECT 1", qctx)
		require.NoError(t, err)
		require.Equal(t, 2, fake.sessionOpens)
		require.Same(t, sess2, d.Session())
		require.Len(t, results, 1)
		require.Len(t, results[0].Rows, 1)
	})
}

// TestAdapterExecuteMapsTxnDirectives pins how Execute maps transaction mode
// and isolation directives while leaving CreateDatabase and MaximumRetries
// unset.
func TestAdapterExecuteMapsTxnDirectives(t *testing.T) {
	fake := &fakeMySQLDriver{
		execFn: func(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
			if opts.Transaction == odriver.Transactional {
				opts.Log.TransactionControl(odriver.Begin, nil)
			}
			opts.Log.CommandExecute(odriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(1, []int64{1}, nil)
			if opts.Transaction == odriver.Transactional {
				opts.Log.TransactionControl(odriver.Commit, nil)
			}
			return 1, nil
		},
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	var logTypes []storepb.TaskRunLog_Type
	rows, err := d.Execute(context.Background(), "-- txn-mode = on\n-- txn-isolation = READ COMMITTED\nINSERT INTO t VALUES (1);", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 3,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logTypes = append(logTypes, entry.Type)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	require.Equal(t, "INSERT INTO t VALUES (1);", fake.execScript)
	require.Equal(t, odriver.Transactional, fake.execOpts.Transaction)
	require.Equal(t, odriver.ReadCommitted, fake.execOpts.Isolation)
	require.False(t, fake.execOpts.CreateDatabase)
	require.Zero(t, fake.execOpts.MaxRetries)
	require.Equal(t, []storepb.TaskRunLog_Type{
		storepb.TaskRunLog_TRANSACTION_CONTROL,
		storepb.TaskRunLog_COMMAND_EXECUTE,
		storepb.TaskRunLog_COMMAND_RESPONSE,
		storepb.TaskRunLog_TRANSACTION_CONTROL,
	}, logTypes)

	_, err = d.Execute(context.Background(), "-- txn-mode = off\nUPDATE t SET v = 1;", db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, odriver.AutoCommit, fake.execOpts.Transaction)
}

func TestAdapterProbeAndRowEstimatorSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("transactional probe runs pre-executions and rolls back", func(t *testing.T) {
		probeSess := &fakeMySQLSession{
			queryFn: func(_ context.Context, statement string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				if statement == "EXPLAIN SELECT @probe_marker" {
					return []*odriver.Result{
						{Statement: "ignored"},
						{
							Statement: statement,
							Columns:   []odriver.Column{{Name: "id", TypeName: "INT"}},
							Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
						},
					}, nil
				}
				return []*odriver.Result{{Statement: statement}}, nil
			},
		}
		fake := &fakeMySQLDriver{
			sessions:    []*fakeMySQLSession{probeSess},
			estimate:    3,
			explainJSON: `{"query_block":{}}`,
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		res, err := d.Probe(ctx, db.ProbeContext{PreExecutions: []string{"", "SET @probe_marker = 1"}}, "EXPLAIN SELECT @probe_marker")
		require.NoError(t, err)
		require.Equal(t, []string{
			"START TRANSACTION",
			"SET @probe_marker = 1",
			"EXPLAIN SELECT @probe_marker",
			"ROLLBACK",
		}, probeSess.statements)
		require.True(t, probeSess.closed)
		require.Equal(t, []string{"id"}, res.Columns)
		require.Equal(t, [][]any{{int64(1)}}, res.Rows)

		count, err := d.CountAffectedRows(ctx, "DELETE FROM td WHERE d = 0;")
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
		require.Equal(t, "DELETE FROM td WHERE d = 0;", fake.estimateSQL)

		plan, err := d.ExplainJSON(ctx, "SELECT 1")
		require.NoError(t, err)
		require.Equal(t, `{"query_block":{}}`, plan)
		require.Equal(t, "SELECT 1", fake.explainSQL)
	})

	t.Run("autocommit probe skips transaction control", func(t *testing.T) {
		probeSess := &fakeMySQLSession{}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeMySQLDriver{sessions: []*fakeMySQLSession{probeSess}}}}

		_, err := d.Probe(ctx, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET @a = 1"}}, "EXPLAIN SELECT @a")
		require.NoError(t, err)
		require.Equal(t, []string{"SET @a = 1", "EXPLAIN SELECT @a"}, probeSess.statements)
		require.True(t, probeSess.closed)
	})
}

// TestAdapterOpenDispatchesMySQLAndMariaDB pins that Open routes MYSQL to
// omni/mysql/driver, MARIADB to omni/mariadb/driver, and guards engine-specific
// estimator methods accordingly.
func TestAdapterOpenDispatchesMySQLAndMariaDB(t *testing.T) {
	ctx := context.Background()
	cfg := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "3306",
			Username: "root",
		},
		Password:          "secret",
		ConnectionContext: db.ConnectionContext{DatabaseName: "appdb"},
	}

	mysqlDrv, err := newDriver().Open(ctx, storepb.Engine_MYSQL, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, mysqlDrv.Close(ctx)) })
	md, ok := mysqlDrv.(*Driver)
	require.True(t, ok)
	require.IsType(t, (*mysqld.Driver)(nil), md.Driver)
	require.True(t, md.WithInstanceMetadata)
	_, err = md.EstimateAffectedRowsMariaDB(ctx, nil, "", "")
	require.ErrorContains(t, err, "needs the MariaDB driver")

	mariaDrv, err := newDriver().Open(ctx, storepb.Engine_MARIADB, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, mariaDrv.Close(ctx)) })
	mrd, ok := mariaDrv.(*Driver)
	require.True(t, ok)
	require.IsType(t, (*mariadbd.Driver)(nil), mrd.Driver)
	require.True(t, mrd.WithInstanceMetadata)
	_, err = mrd.EstimateAffectedRows(ctx, nil, "", "")
	require.ErrorContains(t, err, "needs the MySQL driver")

	_, err = newDriver().Open(ctx, storepb.Engine_POSTGRES, cfg)
	require.ErrorContains(t, err, "unsupported engine POSTGRES")

	_, err = newDriver().Open(ctx, storepb.Engine_MYSQL, db.ConnectionConfig{})
	require.ErrorContains(t, err, "a data source is required")
}
