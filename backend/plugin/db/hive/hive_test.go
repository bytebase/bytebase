package hive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestHiveConfig maps the data source and the caller's parameters onto the
// driver config. The login is the part the driver this replaced never passed on.
func TestHiveConfig(t *testing.T) {
	ds := &storepb.DataSource{
		Host:     "hive.example.com",
		Port:     "10001",
		Username: "hive",
		ExtraConnectionParameters: map[string]string{
			"hive.exec.mode": "local",
		},
	}
	cfg, err := hiveConfig(db.ConnectionConfig{
		DataSource:        ds,
		Password:          "secret",
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	})
	require.NoError(t, err)
	require.Equal(t, "hive.example.com", cfg.Host)
	require.Equal(t, "10001", cfg.Port)
	require.Equal(t, "app", cfg.Database)
	require.Equal(t, "hive", cfg.User)
	require.Equal(t, "secret", cfg.Password)
	require.Equal(t, map[string]string{"hive.exec.mode": "local"}, cfg.Params)
	require.Nil(t, cfg.TLS)
	require.Nil(t, cfg.Dial)
}

// TestHiveConfigDefaults: a data source that names no port leaves the choice to
// the driver, which opens on HiveServer2's 10000.
func TestHiveConfigDefaults(t *testing.T) {
	cfg, err := hiveConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{Host: "hive.example.com"}})
	require.NoError(t, err)
	require.Empty(t, cfg.Port)
	require.Empty(t, cfg.Database)
	require.Empty(t, cfg.Params)
}

// TestHiveConfigRefusesSSH: gohive v2 holds its dialer where a caller cannot
// set one, so an SSH data source is refused instead of dialling the direct host
// and silently ignoring the jump host.
func TestHiveConfigRefusesSSH(t *testing.T) {
	_, err := hiveConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{
		Host:    "hive.example.com",
		SshHost: "jump.example.com",
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "forward a local port to hive.example.com:10000")
}

// TestHiveConfigTLS: a data source that asks for SSL gets a TLS config for the
// host the driver dials, which the driver this replaced ignored.
func TestHiveConfigTLS(t *testing.T) {
	cfg, err := hiveConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{
		Host:   "hive.example.com",
		UseSsl: true,
	}})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	tlsConfig, err := cfg.TLS("hive.example.com")
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
}

func TestExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", util.OmniExplainFormat(v1pb.QueryOption_JSON, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_TEXT, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "TEXT", v1pb.QueryOption_JSON))
}

// TestToRowValue keeps the product's shapes: a NULL is null, a 32-bit number is
// Int32, a FLOAT is Float and a DOUBLE is Double.
func TestToRowValue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value odriver.Value
		want  *v1pb.RowValue
	}{
		{name: "null", value: odriver.Value{Kind: odriver.Null}, want: util.NullRowValue},
		{name: "bool", value: odriver.Value{Kind: odriver.Bool, Bool: true}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}}},
		{name: "bytes", value: odriver.Value{Kind: odriver.Bytes, Bytes: []byte("b")}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte("b")}}},
		{name: "int32", value: odriver.Value{Kind: odriver.Int32, Int64: 7}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: 7}}},
		{name: "int64", value: odriver.Value{Kind: odriver.Int64, Int64: 1 << 40}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1 << 40}}},
		{name: "float", value: odriver.Value{Kind: odriver.Float32, Float64: 1.5}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: 1.5}}},
		{name: "double", value: odriver.Value{Kind: odriver.Float64, Float64: 1.5}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 1.5}}},
		{name: "string", value: odriver.Value{Kind: odriver.String, String: "s"}, want: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "s"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, toRowValue(tc.value))
		})
	}
}

// TestToQueryResult maps a result onto the product's: upper-case type names, one
// row count, and the error a statement carries. A cut result reports the legacy
// cap message rather than the statement's own error.
func toQueryResult(result *odriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	return util.OmniResultToQueryResultWith(result, queryContext.MaximumSQLResultSize, hiveQueryResultOptions(queryContext))
}

func TestToQueryResult(t *testing.T) {
	result := &odriver.Result{
		Statement: "SELECT 1",
		Columns:   []odriver.Column{{Name: "id", TypeName: "int"}, {Name: "note", TypeName: "string"}},
		Rows:      [][]odriver.Value{{{Kind: odriver.Int32, Int64: 1}, {Kind: odriver.String, String: "a"}}},
	}
	converted := toQueryResult(result, db.QueryContext{})
	require.Equal(t, "SELECT 1", converted.Statement)
	require.Equal(t, []string{"id", "note"}, converted.ColumnNames)
	require.Equal(t, []string{"INT", "STRING"}, converted.ColumnTypeNames)
	require.Equal(t, int64(1), converted.RowsCount)
	require.Len(t, converted.Rows, 1)
	require.Empty(t, converted.Error)
	require.Nil(t, converted.QueryPlan)

	result.Truncated = true
	result.Err = errors.New("boom")
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(64), toQueryResult(result, db.QueryContext{MaximumSQLResultSize: 64}).Error)

	result.Truncated = false
	require.Equal(t, "boom", toQueryResult(result, db.QueryContext{MaximumSQLResultSize: 64}).Error)
}

// TestToQueryResultRowLimit: the policy's row cap reaches the result even though
// Hive has no parser to rewrite the statement with a LIMIT, and a plan is never
// cut.
func TestToQueryResultRowLimit(t *testing.T) {
	result := &odriver.Result{
		Statement: "SELECT id FROM t",
		Columns:   []odriver.Column{{Name: "id", TypeName: "INT"}},
		Rows: [][]odriver.Value{
			{{Kind: odriver.Int32, Int64: 1}},
			{{Kind: odriver.Int32, Int64: 2}},
			{{Kind: odriver.Int32, Int64: 3}},
		},
	}
	cut := toQueryResult(result, db.QueryContext{Limit: 2})
	require.Equal(t, int64(2), cut.RowsCount)
	require.Len(t, cut.Rows, 2)
	require.Equal(t, int32(2), cut.Rows[1].Values[0].GetInt32Value())

	plan := toQueryResult(result, db.QueryContext{Limit: 2, Explain: true})
	require.Equal(t, int64(3), plan.RowsCount)
	require.Len(t, plan.Rows, 3)
}

// TestSSHDataSourceIsRefused proves the refusal reaches the product's contract
// rather than a direct dial.
func TestSSHDataSourceIsRefused(t *testing.T) {
	_, err := newDriver().Open(t.Context(), storepb.Engine_HIVE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Host: "hive.example.com", SshHost: "jump.example.com"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "forward a local port")
}

type fakeHiveSession struct {
	gotScript string
	gotOpts   odriver.QueryOptions
	results   []*odriver.Result
	closed    bool
}

func (s *fakeHiveSession) Query(_ context.Context, script string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

func (s *fakeHiveSession) Close() error {
	s.closed = true
	return nil
}

type fakeHiveDriver struct {
	session       *fakeHiveSession
	executeScript string
	executeOpts   odriver.ExecuteOptions
	executeFn     func(script string, opts odriver.ExecuteOptions) (int64, error)
	instance      *odriver.Instance
	schema        *metadatapb.DatabaseSchemaMetadata
	dumpDDL       string
}

func (*fakeHiveDriver) Ping(context.Context) error { return nil }
func (*fakeHiveDriver) Close() error               { return nil }
func (f *fakeHiveDriver) Session(context.Context) (odriver.Session, error) {
	return f.session, nil
}

func (f *fakeHiveDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeHiveDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeHiveDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (f *fakeHiveDriver) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(w, f.dumpDDL)
	return err
}

func TestHiveAdapterSeam(t *testing.T) {
	ctx := context.Background()
	sess := &fakeHiveSession{
		results: []*odriver.Result{{
			Statement: "SELECT id FROM orders",
			Columns:   []odriver.Column{{Name: "id", TypeName: "int"}},
			Rows: [][]odriver.Value{
				{{Kind: odriver.Int32, Int64: 1}},
				{{Kind: odriver.Int32, Int64: 2}},
			},
		}},
	}
	var logs []*storepb.TaskRunLog
	impl := &fakeHiveDriver{
		session: sess,
		executeFn: func(script string, opts odriver.ExecuteOptions) (int64, error) {
			opts.Log.CommandExecute(odriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(0, nil, nil)
			return 0, nil
		},
		instance: &odriver.Instance{
			Version:   "4.0.1",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "app"}},
		},
		schema:  &metadatapb.DatabaseSchemaMetadata{Name: "app"},
		dumpDDL: "CREATE TABLE `app`.`orders` (`id` INT);\n",
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

	// QueryConn forwards schema, explain format, and max result bytes, applies Limit client-side, and closes its ephemeral session.
	got, err := d.QueryConn(ctx, "SELECT id FROM orders", db.QueryContext{
		Schema:               "app",
		Limit:                1,
		MaximumSQLResultSize: 1024,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
	})
	require.NoError(t, err)
	require.True(t, sess.closed)
	require.Equal(t, odriver.QueryOptions{
		Schema:         "app",
		ExplainFormat:  "JSON",
		MaxResultBytes: 1024,
	}, sess.gotOpts)
	require.Len(t, got, 1)
	require.Equal(t, []string{"INT"}, got[0].ColumnTypeNames)
	require.Equal(t, int64(1), got[0].RowsCount)

	// Execute runs in AutoCommit mode without parsing transaction directives and logs each command.
	_, err = d.Execute(ctx, "-- txn-mode=on\nCREATE TABLE t (id INT);", db.ExecuteOptions{
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, "-- txn-mode=on\nCREATE TABLE t (id INT);", impl.executeScript)
	require.Equal(t, odriver.AutoCommit, impl.executeOpts.Transaction)
	require.Equal(t, 2, impl.executeOpts.MaxRetries)
	require.Len(t, logs, 2)
	require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
	require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)

	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "4.0.1", inst.Version)

	schema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "app", schema.GetName())

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, schema))
	require.Contains(t, out.String(), "CREATE TABLE `app`.`orders`")
}
