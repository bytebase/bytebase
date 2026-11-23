package spanner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	spannerdriver "github.com/bytebase/omni/spanner/driver"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestToRowValue(t *testing.T) {
	tests := []struct {
		name  string
		value odriver.Value
		want  string
	}{
		{name: "null", value: odriver.Value{Kind: odriver.Null}, want: "null"},
		{name: "bool", value: odriver.Value{Kind: odriver.Bool, Bool: true}, want: "bool"},
		{name: "bytes", value: odriver.Value{Kind: odriver.Bytes, Bytes: []byte("hi")}, want: "bytes"},
		{name: "double", value: odriver.Value{Kind: odriver.Float64, Float64: 1.5}, want: "double"},
		{name: "int64", value: odriver.Value{Kind: odriver.Int64, Int64: 42}, want: "int64"},
		{name: "text", value: odriver.Value{Kind: odriver.String, String: "x"}, want: "string"},
		{name: "date is text", value: odriver.Value{Kind: odriver.String, String: "2024-01-15"}, want: "string"},
		{
			name:  "timestamp",
			value: odriver.Value{Kind: odriver.Timestamp, Time: time.Date(2024, 1, 15, 10, 30, 45, 123456000, time.UTC), Precision: 6},
			want:  "timestamp",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := toRowValue(test.value)
			require.NotNil(t, got)
			switch test.want {
			case "null":
				require.Equal(t, util.NullRowValue, got)
			case "bool":
				require.True(t, got.GetBoolValue())
			case "bytes":
				require.Equal(t, []byte("hi"), got.GetBytesValue())
			case "double":
				require.Equal(t, 1.5, got.GetDoubleValue())
			case "int64":
				require.Equal(t, int64(42), got.GetInt64Value())
			case "string":
				require.Equal(t, test.value.String, got.GetStringValue())
			case "timestamp":
				require.Equal(t, timestamppb.New(test.value.Time), got.GetTimestampValue().GetGoogleTimestamp())
				require.Equal(t, int32(6), got.GetTimestampValue().GetAccuracy())
			default:
				t.Fatalf("unknown want %q", test.want)
			}
		})
	}
}

// TestSpannerConfig checks what the adapter resolves into spanner/driver's
// Config: the resource name's two parts, the credential document, the caller's
// extra parameters and TLS.
func TestSpannerConfig(t *testing.T) {
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			ProjectId:  "p",
			InstanceId: "i",
			Host:       "10.0.0.1",
			Port:       "443",
			IamExtension: &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"authorized_user"}`},
			},
			ExtraConnectionParameters: map[string]string{"unknown": "x"},
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "d"},
	}
	got, err := spannerConfig(config)
	require.NoError(t, err)
	require.Equal(t, "p", got.Params[spannerdriver.ParamProject])
	require.Equal(t, "i", got.Params[spannerdriver.ParamInstance])
	require.Equal(t, "d", got.Database)
	require.Equal(t, "10.0.0.1", got.Host)
	require.Equal(t, "443", got.Port)
	require.Equal(t, `{"type":"authorized_user"}`, got.Password, "the data source's credential is the driver's password")
	require.Equal(t, "x", got.Params["unknown"])
	require.Nil(t, got.TLS, "TLS stays the client's unless the caller asks for it")

	// The emulator takes no credential, and the Google client refuses both at
	// once, so naming the emulator drops it.
	config.DataSource.ExtraConnectionParameters["emulator"] = "1"
	got, err = spannerConfig(config)
	require.NoError(t, err)
	require.Empty(t, got.Password)
	delete(config.DataSource.ExtraConnectionParameters, "emulator")

	// A caller that resolved a credential itself passes it as the password.
	config.Password = `{"type":"service_account"}`
	got, err = spannerConfig(config)
	require.NoError(t, err)
	require.Equal(t, `{"type":"service_account"}`, got.Password)

	// An SSL data source asks for TLS, bound to the host the driver dials.
	config.DataSource.UseSsl = true
	config.DataSource.VerifyTlsCertificate = true
	got, err = spannerConfig(config)
	require.NoError(t, err)
	require.NotNil(t, got.TLS)
	tlsConfig, err := got.TLS("10.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "10.0.0.1", tlsConfig.ServerName)
}

func TestOpenRejectsInvalidCredentialJSON(t *testing.T) {
	_, err := db.Open(context.Background(), storepb.Engine_SPANNER, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			ProjectId:  "default",
			InstanceId: "default",
			Host:       "127.0.0.1",
			Port:       "9010",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "adapter"},
		Password:          "not a document",
	})
	require.ErrorContains(t, err, "failed to parse GCP credential JSON")
}

func TestOpenPassesGoogleSQLOnlyOption(t *testing.T) {
	var (
		gotCfg  odriver.Config
		gotOpts []spannerdriver.Options
	)
	fake := &fakeSpannerDriver{
		instance: &odriver.Instance{
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "gsql_db"}},
		},
	}
	origOpen := openSpannerDriver
	openSpannerDriver = func(_ context.Context, cfg odriver.Config, opts ...spannerdriver.Options) (odriver.Driver, error) {
		gotCfg = cfg
		gotOpts = opts
		return fake, nil
	}
	t.Cleanup(func() { openSpannerDriver = origOpen })

	ctx := context.Background()
	opened, err := (&Driver{}).Open(ctx, storepb.Engine_SPANNER, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			ProjectId:  "p",
			InstanceId: "i",
			Host:       "127.0.0.1",
			Port:       "9010",
			ExtraConnectionParameters: map[string]string{
				spannerdriver.ParamEmulator: "1",
			},
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "gsql_db"},
		Password:          `{"type":"service_account"}`,
	})
	require.NoError(t, err)
	require.Equal(t, "p", gotCfg.Params[spannerdriver.ParamProject])
	require.Equal(t, "i", gotCfg.Params[spannerdriver.ParamInstance])
	require.Equal(t, "gsql_db", gotCfg.Database)
	require.Empty(t, gotCfg.Password)
	require.Equal(t, []spannerdriver.Options{{GoogleSQLOnly: true}}, gotOpts)

	inst, err := opened.SyncInstance(ctx)
	require.NoError(t, err)
	require.Nil(t, inst.Metadata)
	require.Len(t, inst.Databases, 1)
	require.Equal(t, "gsql_db", inst.Databases[0].Name)
}

func toQueryResult(result *odriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	return util.OmniResultToQueryResultWith(result, maxResultBytes, spannerQueryResultOptions(false))
}

func TestToQueryResult(t *testing.T) {
	maxBytes := int64(1024 * 1024)
	latency := time.Millisecond

	query := toQueryResult(&odriver.Result{
		Statement: "SELECT 1 AS n",
		Columns:   []odriver.Column{{Name: "n", TypeName: "INT64"}},
		Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
		Latency:   latency,
	}, maxBytes)
	require.Equal(t, "SELECT 1 AS n", query.Statement)
	require.Equal(t, []string{"n"}, query.ColumnNames)
	require.Equal(t, []string{"INT64"}, query.ColumnTypeNames)
	require.Equal(t, int64(1), query.RowsCount)
	require.Equal(t, int64(1), query.Rows[0].Values[0].GetInt64Value())

	affected := toQueryResult(&odriver.Result{
		Statement:    "UPDATE Singers SET Rating = 5 WHERE Id = 1",
		AffectedRows: func() *int64 { value := int64(1); return &value }(),
		Latency:      latency,
	}, maxBytes)
	require.Equal(t, []string{"Affected Rows"}, affected.ColumnNames)
	require.Equal(t, []string{"INT"}, affected.ColumnTypeNames)
	require.Equal(t, int64(1), affected.RowsCount)
	require.Equal(t, int64(1), affected.Rows[0].Values[0].GetInt64Value())

	ddl := toQueryResult(&odriver.Result{Statement: "CREATE TABLE T (Id INT64) PRIMARY KEY (Id)", Latency: latency}, maxBytes)
	require.Empty(t, ddl.ColumnNames)
	require.Empty(t, ddl.Rows)
	require.Equal(t, int64(0), ddl.RowsCount)

	failed := toQueryResult(&odriver.Result{Statement: "SELECT * FROM Missing", Err: errors.New("Table not found"), Latency: latency}, maxBytes)
	require.Equal(t, "Table not found", failed.Error)
	require.Empty(t, failed.Rows)
	require.Equal(t, int64(0), failed.RowsCount)

	truncated := toQueryResult(&odriver.Result{
		Statement: "SELECT Id FROM Singers",
		Columns:   []odriver.Column{{Name: "Id", TypeName: "INT64"}},
		Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
		Truncated: true,
	}, 300)
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(300), truncated.Error)
	require.Equal(t, int64(1), truncated.RowsCount)
}

type fakeSpannerSession struct {
	gotScript string
	gotOpts   odriver.QueryOptions
	results   []*odriver.Result
	closed    bool
}

func (s *fakeSpannerSession) Query(_ context.Context, script string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

func (s *fakeSpannerSession) Close() error {
	s.closed = true
	return nil
}

type fakeSpannerDriver struct {
	session       *fakeSpannerSession
	executeScript string
	executeOpts   odriver.ExecuteOptions
	executeFn     func(script string, opts odriver.ExecuteOptions) (int64, error)
	instance      *odriver.Instance
	instanceErr   error
	schema        *metadatapb.DatabaseSchemaMetadata
	schemaErr     error
	dumpDDL       string
}

func (*fakeSpannerDriver) Ping(context.Context) error { return nil }
func (*fakeSpannerDriver) Close() error               { return nil }
func (f *fakeSpannerDriver) Session(context.Context) (odriver.Session, error) {
	return f.session, nil
}

func (f *fakeSpannerDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeSpannerDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
	if f.instanceErr != nil {
		return nil, f.instanceErr
	}
	if f.instance != nil {
		return f.instance, nil
	}
	return &odriver.Instance{}, nil
}

func (f *fakeSpannerDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, f.schemaErr
}

func (f *fakeSpannerDriver) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(w, f.dumpDDL)
	return err
}

func TestSpannerAdapterSeam(t *testing.T) {
	ctx := context.Background()
	sess := &fakeSpannerSession{
		results: []*odriver.Result{{
			Statement: "SELECT 1",
			Columns:   []odriver.Column{{Name: "n", TypeName: "INT64"}},
			Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
			Plan:      &odriver.Plan{Format: "JSON"},
		}},
	}
	var logs []*storepb.TaskRunLog
	impl := &fakeSpannerDriver{
		session: sess,
		executeFn: func(script string, opts odriver.ExecuteOptions) (int64, error) {
			opts.Log.TransactionControl(odriver.Begin, nil)
			opts.Log.CommandExecute(odriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(1, []int64{1}, nil)
			opts.Log.TransactionControl(odriver.Commit, nil)
			return 1, nil
		},
		instance: &odriver.Instance{
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "app"}},
		},
		schema:  &metadatapb.DatabaseSchemaMetadata{Name: "app"},
		dumpDDL: "CREATE TABLE Singers (SingerId INT64 NOT NULL) PRIMARY KEY (SingerId);\n",
	}
	d := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:           impl,
			NotFoundResource: "database",
			DatabaseName:     "app",
		},
	}

	// SyncInstance delegates to omni/spanner/driver via OmniAdapter.
	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Nil(t, inst.Metadata)
	require.Len(t, inst.Databases, 1)
	require.Equal(t, "app", inst.Databases[0].Name)

	impl.instanceErr = errors.New("permission denied")
	_, err = d.SyncInstance(ctx)
	require.ErrorContains(t, err, "permission denied")
	impl.instanceErr = nil

	// QueryConn forwards options, clears QueryPlan when Explain is true, and closes ephemeral session.
	got, err := d.QueryConn(ctx, "SELECT 1", db.QueryContext{Limit: 2, Explain: true, MaximumSQLResultSize: 2048})
	require.NoError(t, err)
	require.True(t, sess.closed)
	require.Equal(t, odriver.QueryOptions{Limit: 2, Explain: true, MaxResultBytes: 2048}, sess.gotOpts)
	require.Len(t, got, 1)
	require.Nil(t, got[0].QueryPlan)

	// Execute parses transaction directive and suppresses per-statement command logs.
	affected, err := d.Execute(ctx, "-- txn-mode=off\nUPDATE Singers SET FirstName = 'a' WHERE SingerId = 1", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
	require.Equal(t, odriver.AutoCommit, impl.executeOpts.Transaction)
	require.True(t, impl.executeOpts.CreateDatabase)
	require.Equal(t, 2, impl.executeOpts.MaxRetries)
	require.Len(t, logs, 2, "Spanner logs transaction control entries only")
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[0].GetTransactionControl().GetType())
	require.Equal(t, storepb.TaskRunLog_TransactionControl_COMMIT, logs[1].GetTransactionControl().GetType())

	// SyncDBSchema translates ErrNotFound and Dump delegates.
	schema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "app", schema.GetName())

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, schema))
	require.Contains(t, out.String(), "CREATE TABLE Singers")

	impl.schemaErr = odriver.ErrNotFound
	_, err = d.SyncDBSchema(ctx)
	var bbErr *common.Error
	require.ErrorAs(t, err, &bbErr)
	require.Equal(t, common.NotFound, bbErr.Code)
	require.Contains(t, err.Error(), `database "app" not found`)
}
