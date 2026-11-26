package drivers

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func openTrino(t *testing.T, password string, useSSL bool) error {
	t.Helper()
	driver, err := newDriver(trinoSpec).Open(context.Background(), storepb.Engine_TRINO, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "trino.example.com",
			Port:     "8080",
			Username: "bytebase",
			UseSsl:   useSSL,
		},
		Password: password,
	})
	if driver != nil {
		// Releases the custom HTTP client the SSL path registers globally.
		t.Cleanup(func() { _ = driver.Close(context.Background()) })
	}
	return err
}

// trino-go-client refuses a password on a plaintext connection, so such a data
// source cannot connect whatever we do. The guard exists for the message: it
// names the setting to change instead of reporting a DSN parse failure.
func TestTrinoOpenRefusesAPasswordWithoutSSL(t *testing.T) {
	err := openTrino(t, "secret", false)
	require.ErrorContains(t, err, "requires SSL to authenticate with a password")
	require.NotContains(t, err.Error(), "secret", "the error must not carry the password")
}

func TestTrinoOpenAcceptsTheConfigurationsThatCanConnect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		useSSL   bool
	}{
		{"password over SSL", "secret", true},
		{"no password without SSL", "", false},
		{"no password over SSL", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := openTrino(t, tc.password, tc.useSSL)
			require.NoError(t, err)
		})
	}
}

type captureTrinoSession struct {
	gotOpts omnidriver.QueryOptions
	results []*omnidriver.Result
}

func (s *captureTrinoSession) Query(_ context.Context, _ string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotOpts = opts
	return s.results, nil
}

func (*captureTrinoSession) Close() error { return nil }

func TestTrinoQueryConnFillsZeroAffectedRowsAndMapsExplainFormat(t *testing.T) {
	session := &captureTrinoSession{
		results: []*omnidriver.Result{
			{Statement: "CREATE TABLE memory.default.t (id BIGINT)"},
		},
	}
	d := newDriver(trinoSpec)
	d.SetSession(session)

	results, err := d.QueryConn(context.Background(), "CREATE TABLE memory.default.t (id BIGINT)", db.QueryContext{
		Schema:               "default",
		Limit:                100,
		Explain:              true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		MaximumSQLResultSize: 1024,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.Equal(t, "default", session.gotOpts.Schema)
	require.Equal(t, 100, session.gotOpts.Limit)
	require.True(t, session.gotOpts.Explain)
	require.Equal(t, "JSON", session.gotOpts.ExplainFormat)
	require.Equal(t, int64(1024), session.gotOpts.MaxResultBytes)
	require.True(t, session.gotOpts.MaskingEnabled)
	require.Len(t, results, 1)
	require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
	require.Equal(t, int64(0), results[0].Rows[0].Values[0].GetInt64Value())
}

type fakeTrinoExecutorDriver struct {
	gotScript string
	gotOpts   omnidriver.ExecuteOptions
	commands  []omnidriver.Command
	affected  []int64
}

func (*fakeTrinoExecutorDriver) Ping(context.Context) error { return nil }
func (*fakeTrinoExecutorDriver) Close() error               { return nil }

func (f *fakeTrinoExecutorDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.gotScript = script
	f.gotOpts = opts
	var total int64
	for i, cmd := range f.commands {
		if opts.Log != nil {
			opts.Log.CommandExecute(cmd)
		}
		rows := f.affected[i]
		total += rows
		if opts.Log != nil {
			opts.Log.CommandResponse(rows, []int64{rows}, nil)
		}
	}
	return total, nil
}

// TestTrinoExecuteLogsTheCommandRangeWithItsSemicolon pins the adapter's log bridge:
// omni ends a command at its last token, while the task-run log's range covers
// the terminating semicolon when present.
func TestTrinoExecuteLogsTheCommandRangeWithItsSemicolon(t *testing.T) {
	t.Parallel()

	const (
		first  = "CREATE TABLE memory.default.t (id BIGINT)"
		second = "INSERT INTO memory.default.t VALUES (1)"
		third  = "SELECT 1"
	)
	script := first + ";\n" + second + ";\n" + third
	secondStart := len(first) + 2
	thirdStart := secondStart + len(second) + 2

	fake := &fakeTrinoExecutorDriver{
		commands: []omnidriver.Command{
			{Text: first, ByteStart: 0, ByteEnd: len(first)},
			{Text: second, ByteStart: secondStart, ByteEnd: secondStart + len(second)},
			{Text: third, ByteStart: thirdStart, ByteEnd: thirdStart + len(third)},
		},
		affected: []int64{0, 1, 0},
	}
	driver := &Driver{spec: trinoSpec, OmniAdapter: util.OmniAdapter{Driver: fake}}

	var logs []*storepb.TaskRunLog
	affected, err := driver.Execute(context.Background(), script, db.ExecuteOptions{
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
	require.Equal(t, script, fake.gotScript)
	require.Empty(t, fake.gotOpts.Transaction)
	require.Empty(t, fake.gotOpts.Isolation)
	require.Equal(t, 2, fake.gotOpts.MaxRetries)

	var ranges []string
	for _, log := range logs {
		if execute := log.GetCommandExecute(); execute != nil {
			require.NotNil(t, execute.Range)
			ranges = append(ranges, script[execute.Range.Start:execute.Range.End])
		}
	}
	require.Equal(t, []string{
		first + ";",
		second + ";",
		third,
	}, ranges)
}

type stubTrinoQuerySession struct {
	gotScript string
	gotOpts   omnidriver.QueryOptions
	results   []*omnidriver.Result
}

func (s *stubTrinoQuerySession) Query(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

func (*stubTrinoQuerySession) Close() error { return nil }

func trinoStringValue(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

// TestTrinoQueryResultRendersTypes pins how the adapter maps omni driver
// values, query options, and columnless zero-affected-row results into v1pb.
func TestTrinoQueryResultRendersTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ts := time.Date(2026, 1, 2, 12, 34, 56, 0, time.UTC)
	tsFrac := time.Date(2026, 1, 2, 12, 34, 56, 789000000, time.UTC)
	tsTZ := time.Date(2026, 1, 2, 12, 34, 56, 0, time.FixedZone("", 7200))

	tests := []struct {
		name    string
		colType string
		value   omnidriver.Value
		want    *v1pb.RowValue
	}{
		{"date", "DATE", omnidriver.Value{Kind: omnidriver.String, String: "2026-01-02"}, trinoStringValue("2026-01-02")},
		{"time", "TIME", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56"}, trinoStringValue("12:34:56")},
		{"time with precision", "TIME", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56.789"}, trinoStringValue("12:34:56.789")},
		{"zoned time", "TIME WITH TIME ZONE", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56+02:00"}, trinoStringValue("12:34:56+02:00")},
		{"zoned time with precision", "TIME WITH TIME ZONE", omnidriver.Value{Kind: omnidriver.String, String: "12:34:56.789+02:00"}, trinoStringValue("12:34:56.789+02:00")},
		{"timestamp", "TIMESTAMP", omnidriver.Value{Kind: omnidriver.Timestamp, Time: ts}, &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(ts),
			}},
		}},
		{"timestamp with a fraction", "TIMESTAMP", omnidriver.Value{Kind: omnidriver.Timestamp, Time: tsFrac, Precision: 3}, &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(tsFrac),
				Accuracy:        3,
			}},
		}},
		{"zoned timestamp", "TIMESTAMP WITH TIME ZONE", omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: tsTZ}, &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 1, 2, 10, 34, 56, 0, time.UTC)),
				Offset:          7200,
			}},
		}},
		{"varchar", "VARCHAR", omnidriver.Value{Kind: omnidriver.String, String: "hi"}, trinoStringValue("hi")},
		{"bigint", "BIGINT", omnidriver.Value{Kind: omnidriver.Int64, Int64: 42}, &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}}},
		{"boolean", "BOOLEAN", omnidriver.Value{Kind: omnidriver.Bool, Bool: true}, &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}}},
		{"decimal", "DECIMAL", omnidriver.Value{Kind: omnidriver.String, String: "1.25"}, trinoStringValue("1.25")},
		{"null date", "DATE", omnidriver.Value{Kind: omnidriver.Null}, util.NullRowValue},
		{"null varchar", "VARCHAR", omnidriver.Value{Kind: omnidriver.Null}, util.NullRowValue},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sess := &stubTrinoQuerySession{
				results: []*omnidriver.Result{{
					Columns: []omnidriver.Column{{Name: "v", TypeName: tc.colType}},
					Rows:    [][]omnidriver.Value{{tc.value}},
				}},
			}
			driver := newDriver(trinoSpec)
			driver.SetSession(sess)

			results, err := driver.QueryConn(ctx, "SELECT v", db.QueryContext{
				Schema:               "analytics",
				Limit:                50,
				Explain:              true,
				Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
				MaximumSQLResultSize: 1 << 20,
				MaskingEnabled:       true,
			})
			require.NoError(t, err)
			require.Equal(t, omnidriver.QueryOptions{
				Schema:         "analytics",
				Limit:          50,
				Explain:        true,
				ExplainFormat:  "JSON",
				MaxResultBytes: 1 << 20,
				MaskingEnabled: true,
			}, sess.gotOpts)
			require.Len(t, results, 1)
			require.Empty(t, results[0].Error)
			require.Equal(t, []string{tc.colType}, results[0].ColumnTypeNames)
			require.Len(t, results[0].Rows, 1)
			require.Empty(t, cmp.Diff(tc.want, results[0].Rows[0].Values[0], protocmp.Transform()))
		})
	}

	t.Run("fills zero affected rows only for columnless error-free results", func(t *testing.T) {
		t.Parallel()
		affected := int64(3)
		sess := &stubTrinoQuerySession{
			results: []*omnidriver.Result{
				{Statement: "SET SESSION query_max_run_time = '1m'"},
				{Statement: "INSERT INTO t VALUES (1), (2), (3)", AffectedRows: &affected},
				{Statement: "SELECT * FROM missing", Err: errors.New("table not found")},
			},
		}
		driver := newDriver(trinoSpec)
		driver.SetSession(sess)

		results, err := driver.QueryConn(ctx, "script", db.QueryContext{MaximumSQLResultSize: 1024})
		require.NoError(t, err)
		require.Len(t, results, 3)

		require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
		require.Equal(t, int64(1), results[0].RowsCount)
		require.Equal(t, int64(0), results[0].Rows[0].Values[0].GetInt64Value())

		require.Equal(t, []string{"Affected Rows"}, results[1].ColumnNames)
		require.Equal(t, int64(3), results[1].Rows[0].Values[0].GetInt64Value())

		require.Equal(t, "table not found", results[2].Error)
		require.Empty(t, results[2].Rows)
	})
}

type countingTrinoQuerierDriver struct {
	sessionOpens int
	driverClosed bool
	session      *recordingTrinoSession
}

func (*countingTrinoQuerierDriver) Ping(context.Context) error { return nil }
func (d *countingTrinoQuerierDriver) Close() error {
	d.driverClosed = true
	return nil
}

func (d *countingTrinoQuerierDriver) Session(context.Context) (omnidriver.Session, error) {
	d.sessionOpens++
	return d.session, nil
}

type recordingTrinoSession struct {
	scripts []string
	closed  bool
}

func (s *recordingTrinoSession) Query(_ context.Context, script string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.scripts = append(s.scripts, script)
	if len(s.scripts) == 1 {
		return []*omnidriver.Result{{Statement: script}}, nil
	}
	return []*omnidriver.Result{{
		Statement: script,
		Columns:   []omnidriver.Column{{Name: "Name", TypeName: "VARCHAR"}, {Name: "Value", TypeName: "VARCHAR"}},
		Rows: [][]omnidriver.Value{{
			{Kind: omnidriver.String, String: "query_max_run_time"},
			{Kind: omnidriver.String, String: "1m"},
		}},
	}}, nil
}

func (s *recordingTrinoSession) Close() error {
	s.closed = true
	return nil
}

// TestTrinoQueryConnKeepsTheSessionForTheDriver pins that separate QueryConn calls
// reuse one lazily opened omni Session until Close releases both the session
// and the driver.
func TestTrinoQueryConnKeepsTheSessionForTheDriver(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sess := &recordingTrinoSession{}
	fake := &countingTrinoQuerierDriver{session: sess}
	driver := &Driver{spec: trinoSpec, OmniAdapter: util.OmniAdapter{Driver: fake}}
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}

	results, err := driver.QueryConn(ctx, "SET SESSION query_max_run_time = '1m'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)

	results, err = driver.QueryConn(ctx, "SHOW SESSION LIKE 'query_max_run_time'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, "1m", results[0].Rows[0].Values[1].GetStringValue())

	require.Equal(t, 1, fake.sessionOpens, "both QueryConn calls must share one session")
	require.Equal(t, []string{
		"SET SESSION query_max_run_time = '1m'",
		"SHOW SESSION LIKE 'query_max_run_time'",
	}, sess.scripts)

	require.NoError(t, driver.Close(ctx))
	require.True(t, sess.closed)
	require.True(t, fake.driverClosed)
}

// brokenTrinoSession is a session that cannot run a statement, as a connection the
// server closed under a canceled statement is.
type brokenTrinoSession struct{}

func (brokenTrinoSession) Query(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	return nil, errors.New("the session is gone")
}

func (brokenTrinoSession) Close() error { return nil }

// TestTrinoQueryConnDropsTheSessionAScriptCouldNotRun pins that a session a failed
// call took with it does not outlive the call: the editor's next statement must
// not run on a connection the server has closed.
func TestTrinoQueryConnDropsTheSessionAScriptCouldNotRun(t *testing.T) {
	d := newDriver(trinoSpec)
	d.SetSession(brokenTrinoSession{})
	_, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
	require.Error(t, err)
	require.Nil(t, d.Session(), "the next statement opens a new session")
}

type fakeTrinoSyncDumpDriver struct {
	instance     *omnidriver.Instance
	schema       *metadatapb.DatabaseSchemaMetadata
	syncDBErr    error
	dumpOut      string
	dumpErr      error
	dumpedSchema *metadatapb.DatabaseSchemaMetadata
}

func (*fakeTrinoSyncDumpDriver) Ping(context.Context) error { return nil }
func (*fakeTrinoSyncDumpDriver) Close() error               { return nil }

func (f *fakeTrinoSyncDumpDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeTrinoSyncDumpDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	if f.syncDBErr != nil {
		return nil, f.syncDBErr
	}
	return f.schema, nil
}

func (f *fakeTrinoSyncDumpDriver) Dump(_ context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	f.dumpedSchema = metadata
	if f.dumpErr != nil {
		return f.dumpErr
	}
	_, err := io.WriteString(out, f.dumpOut)
	return err
}

// TestTrinoSyncInstanceAndDatabaseDelegateToOmniDriver pins how the Trino adapter
// forwards SyncInstance, SyncDBSchema, and Dump to the omni driver.
func TestTrinoSyncInstanceAndDatabaseDelegateToOmniDriver(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantSchema := &metadatapb.DatabaseSchemaMetadata{
		Name: "memory",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "default",
			Tables: []*metadatapb.TableMetadata{{
				Name: "items",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "bigint"},
					{Name: "name", Type: "varchar(10)"},
				},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v_items",
				Definition: "SELECT id FROM memory.default.items",
			}},
		}},
	}
	fake := &fakeTrinoSyncDumpDriver{
		instance: &omnidriver.Instance{
			Version:   "482",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "memory"}, {Name: "system"}},
		},
		schema:  wantSchema,
		dumpOut: "CREATE SCHEMA IF NOT EXISTS \"default\";\n",
	}
	driver := &Driver{
		spec: trinoSpec,
		OmniAdapter: util.OmniAdapter{
			Driver:               fake,
			WithInstanceMetadata: true,
		},
	}

	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "482", instance.Version)
	require.NotNil(t, instance.Metadata)
	require.Len(t, instance.Databases, 2)
	require.Equal(t, "memory", instance.Databases[0].Name)

	gotSchema, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(wantSchema, gotSchema, protocmp.Transform()))

	var dump strings.Builder
	require.NoError(t, driver.Dump(ctx, &dump, gotSchema))
	require.Equal(t, fake.dumpOut, dump.String())
	require.Empty(t, cmp.Diff(wantSchema, fake.dumpedSchema, protocmp.Transform()))

	fake.syncDBErr = errors.New("catalog unavailable")
	_, err = driver.SyncDBSchema(ctx)
	require.ErrorIs(t, err, fake.syncDBErr)

	fake.dumpErr = errors.New("dump failed")
	err = driver.Dump(ctx, &dump, gotSchema)
	require.ErrorIs(t, err, fake.dumpErr)
}
