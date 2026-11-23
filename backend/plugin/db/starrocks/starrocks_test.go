package starrocks

import (
	"context"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Register the StarRocks editor validator.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/starrocks"
)

// TestValidateSQLForEditorUsesStarRocksGrammar pins this driver to the StarRocks
// parser: a StarRocks-only spelling must be accepted, which routing the
// connection through a sibling engine's grammar would not do. The driver no
// longer calls it — the statement classification is the driver contract's — but
// the registry it reads is still the engine's own.
func TestValidateSQLForEditorUsesStarRocksGrammar(t *testing.T) {
	// StarRocks generated column AS (expr); Doris wants GENERATED ALWAYS AS.
	const starRocksOnlyDDL = "CREATE TABLE t (a INT, b INT AS (a + 1)) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)"
	_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_STARROCKS, starRocksOnlyDDL)
	require.NoError(t, err)
	require.False(t, allQuery, "a CREATE TABLE is DDL, not a read-only query")
}

func TestDriverConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  db.ConnectionConfig
		assert  func(t *testing.T, cfg omnidriver.Config)
		wantErr string
	}{
		{
			name: "tcp host and the product's login",
			config: db.ConnectionConfig{
				DataSource:        &storepb.DataSource{Host: "starrocks", Port: "9030", Username: "root"},
				Password:          "secret",
				ConnectionContext: db.ConnectionContext{DatabaseName: "analytics", EngineVersion: "3.4.10"},
			},
			assert: func(t *testing.T, cfg omnidriver.Config) {
				require.Equal(t, "starrocks", cfg.Host)
				require.Equal(t, "9030", cfg.Port)
				require.Equal(t, "analytics", cfg.Database)
				require.Equal(t, "root", cfg.User)
				require.Equal(t, "secret", cfg.Password)
				require.Equal(t, "3.4.10", cfg.EngineVersion)
				// The driver prefixes a query statement with "/*app=bytebase*/".
				require.Equal(t, "bytebase", cfg.ApplicationName)
				require.Nil(t, cfg.Dial)
				require.Nil(t, cfg.TLS)
			},
		},
		{
			name: "a unix socket host is the driver's own convention",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "/tmp/starrocks.sock", Username: "root"},
			},
			assert: func(t *testing.T, cfg omnidriver.Config) {
				require.Equal(t, "/tmp/starrocks.sock", cfg.Host)
				require.Empty(t, cfg.Port)
			},
		},
		{
			name: "the data source's extra parameters are not mapped",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                      "starrocks",
					Username:                  "root",
					ExtraConnectionParameters: map[string]string{"parseTime": "true"},
				},
			},
			assert: func(t *testing.T, cfg omnidriver.Config) {
				// The connection carries the driver's three fixed parameters and
				// not the data source's.
				require.Empty(t, cfg.Params)
			},
		},
		{
			name: "TLS material without verification",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "starrocks", Username: "root", UseSsl: true},
			},
			assert: func(t *testing.T, cfg omnidriver.Config) {
				require.NotNil(t, cfg.TLS)
				tlscfg, err := cfg.TLS("starrocks")
				require.NoError(t, err)
				require.True(t, tlscfg.InsecureSkipVerify)
			},
		},
		{
			name: "TLS material with verification binds the data source's host",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                 "starrocks",
					Username:             "root",
					UseSsl:               true,
					VerifyTlsCertificate: true,
				},
			},
			assert: func(t *testing.T, cfg omnidriver.Config) {
				require.NotNil(t, cfg.TLS)
				tlscfg, err := cfg.TLS("starrocks")
				require.NoError(t, err)
				require.NotNil(t, tlscfg.VerifyPeerCertificate)
			},
		},
		{
			name: "incomplete TLS material is refused before the dial",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "starrocks", Username: "root", UseSsl: true, SslKey: "key"},
			},
			wantErr: "sql: tls config error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := util.MySQLFamilyDriverConfig(tc.config, &util.OmniAdapter{})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			tc.assert(t, cfg)
		})
	}
}

func TestExecuteOptions(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		opts      db.ExecuteOptions
		want      omnidriver.TransactionMode
	}{
		{name: "the default mode is transactional", statement: "SELECT 1;", want: omnidriver.Transactional},
		{name: "the directive turns transactions off", statement: "-- txn-mode=off\nSELECT 1;", want: omnidriver.AutoCommit},
		{name: "creating a database runs outside a transaction", statement: "-- txn-mode=on\n-- txn-isolation=serializable\nCREATE DATABASE app;", opts: db.ExecuteOptions{CreateDatabase: true}, want: omnidriver.Transactional},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, got := util.BuildExecuteOptions(tc.statement, tc.opts, util.MySQLForkExecuteConfig)
			require.Equal(t, tc.want, got.Transaction)
			require.Equal(t, tc.opts.CreateDatabase, got.CreateDatabase)
			require.Equal(t, tc.opts.MaximumRetries, got.MaxRetries)
			// The isolation directive is read and its level ignored, as before.
			require.Empty(t, got.Isolation)
		})
	}
}

func TestQueryOptions(t *testing.T) {
	got := util.MySQLForkQueryOptions(db.QueryContext{Limit: 7, Explain: true, MaximumSQLResultSize: 1 << 20})
	require.Equal(t, 7, got.Limit)
	require.True(t, got.Explain)
	require.Equal(t, int64(1<<20), got.MaxResultBytes)
	// The explain format and the schema are not mapped: StarRocks has one plan
	// format and no per-session schema.
	require.Empty(t, got.ExplainFormat)
	require.Empty(t, got.Schema)
}

func TestToRowValue(t *testing.T) {
	when := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	tests := []struct {
		name  string
		value omnidriver.Value
		want  *v1pb.RowValue
	}{
		{name: "null", value: omnidriver.Value{Kind: omnidriver.Null}, want: nullRowValue()},
		{name: "bool", value: omnidriver.Value{Kind: omnidriver.Bool, Bool: true}, want: boolRowValue(true)},
		{name: "int64", value: omnidriver.Value{Kind: omnidriver.Int64, Int64: 42}, want: intRowValue(42)},
		{name: "int32", value: omnidriver.Value{Kind: omnidriver.Int32, Int64: 7}, want: intRowValue(7)},
		{name: "double", value: omnidriver.Value{Kind: omnidriver.Float64, Float64: 2.5}, want: doubleRowValue(2.5)},
		{name: "float", value: omnidriver.Value{Kind: omnidriver.Float32, Float64: 1.5}, want: doubleRowValue(1.5)},
		{name: "string", value: omnidriver.Value{Kind: omnidriver.String, String: "hello"}, want: stringRowValue("hello")},
		{name: "bytes", value: omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte("abc")}, want: bytesRowValue([]byte("abc"))},
		// A zero-byte BIT/VARBIT value is a NULL cell.
		{name: "empty bytes is the old NULL cell", value: omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte{}}, want: nullRowValue()},
		{name: "timestamp", value: omnidriver.Value{Kind: omnidriver.Timestamp, Time: when, Precision: 3}, want: timestampRowValue(when, 3)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, util.MySQLForkRowValue(tc.value))
		})
	}
}

func TestToQueryResult(t *testing.T) {
	latency := 5 * time.Millisecond
	toQueryResult := func(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
		return util.OmniResultToQueryResultWith(result, maxResultBytes, util.MySQLForkQueryResultOptions())
	}

	t.Run("a result set keeps the driver's columns, rows and statement", func(t *testing.T) {
		got := toQueryResult(&omnidriver.Result{
			Statement: "SELECT 1;",
			Columns:   []omnidriver.Column{{Name: "id", TypeName: "INT"}},
			Rows: [][]omnidriver.Value{
				{{Kind: omnidriver.Int64, Int64: 1}},
				{{Kind: omnidriver.Null}},
			},
			Latency: latency,
		}, 0)
		require.Equal(t, &v1pb.QueryResult{
			Statement:       "SELECT 1;",
			ColumnNames:     []string{"id"},
			ColumnTypeNames: []string{"INT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{intRowValue(1)}},
				{Values: []*v1pb.RowValue{nullRowValue()}},
			},
			RowsCount: 2,
			Latency:   durationpb.New(latency),
		}, got)
	})

	t.Run("a statement with no rows reports its affected rows", func(t *testing.T) {
		affected := int64(3)
		got := toQueryResult(&omnidriver.Result{
			Statement:    "UPDATE t SET a = 1;",
			AffectedRows: &affected,
			Latency:      latency,
		}, 0)
		require.Equal(t, "UPDATE t SET a = 1;", got.Statement)
		require.Equal(t, []string{"Affected Rows"}, got.ColumnNames)
		require.Equal(t, []string{"INT"}, got.ColumnTypeNames)
		require.Equal(t, int64(1), got.RowsCount)
		require.Equal(t, intRowValue(3), got.Rows[0].Values[0])
		require.Equal(t, durationpb.New(latency), got.Latency)
	})

	t.Run("a truncated result reports the legacy cap message", func(t *testing.T) {
		got := toQueryResult(&omnidriver.Result{
			Statement: "SELECT id FROM t;",
			Columns:   []omnidriver.Column{{Name: "id", TypeName: "INT"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int64, Int64: 1}}},
			Truncated: true,
		}, 1<<20)
		require.Equal(t, "Output of query exceeds max allowed output size of 1MB", got.Error)
	})

	t.Run("a statement error is the result's error", func(t *testing.T) {
		got := toQueryResult(&omnidriver.Result{
			Statement: "SELECT * FROM missing;",
			Err:       errTest{"Error 5502 (42602): Unknown table"},
		}, 0)
		require.Equal(t, "Error 5502 (42602): Unknown table", got.Error)
		require.Empty(t, got.Rows)
	})

	t.Run("no statements is no results", func(t *testing.T) {
		require.Nil(t, util.OmniResultsToQueryResultsWith(nil, 0, util.MySQLForkQueryResultOptions()))
	})
}

func TestExecuteLogger(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}
	logger := util.NewExecuteLogger(opts)
	logger.TransactionControl(omnidriver.Begin, nil)
	logger.CommandExecute(omnidriver.Command{Text: "INSERT INTO t VALUES (1);", ByteStart: 4, ByteEnd: 30})
	logger.CommandResponse(1, []int64{1}, nil)
	logger.TransactionControl(omnidriver.Rollback, errTest{"boom"})

	require.Len(t, logs, 4)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[0].GetTransactionControl().GetType())
	require.Equal(t, "INSERT INTO t VALUES (1);", logs[1].GetCommandExecute().GetStatement())
	require.Equal(t, int64(1), logs[2].GetCommandResponse().GetAffectedRows())
	require.Equal(t, []int64{1}, logs[2].GetCommandResponse().GetAllAffectedRows())
	require.Equal(t, storepb.TaskRunLog_TransactionControl_ROLLBACK, logs[3].GetTransactionControl().GetType())
	require.Equal(t, "boom", logs[3].GetTransactionControl().GetError())
}

func TestSyncDatabaseError(t *testing.T) {
	notFound := &notFoundError{}
	driver := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:           &fakeSyncDriver{err: notFound},
			NotFoundResource: "database",
			DatabaseName:     "analytics",
		},
	}
	_, err := driver.SyncDBSchema(t.Context())
	require.Equal(t, "database \"analytics\" not found", err.Error())

	other := errTest{"Error 1040: Too many connections"}
	driver.Driver = &fakeSyncDriver{err: other}
	_, err = driver.SyncDBSchema(t.Context())
	require.Equal(t, other, err)
}

type fakeSyncDriver struct {
	omnidriver.Driver
	err error
}

func (*fakeSyncDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) { return nil, nil }
func (f *fakeSyncDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return nil, f.err
}

type errTest struct{ message string }

func (e errTest) Error() string { return e.message }

// notFoundError wraps the driver's not-found error the way the driver does.
type notFoundError struct{}

func (*notFoundError) Error() string { return "not found: database" }
func (*notFoundError) Is(target error) bool {
	return target == omnidriver.ErrNotFound
}

func nullRowValue() *v1pb.RowValue {
	return util.NullRowValue
}

func boolRowValue(v bool) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: v}}
}

func intRowValue(v int64) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: v}}
}

func doubleRowValue(v float64) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: v}}
}

func stringRowValue(v string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
}

func bytesRowValue(v []byte) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: v}}
}

func timestampRowValue(v time.Time, precision int32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
		TimestampValue: &v1pb.RowValue_Timestamp{
			GoogleTimestamp: timestamppb.New(v),
			Accuracy:        precision,
		},
	}}
}

// TestQueryOptionsCarriesTheRequest pins the mapping QueryConn hands the
// driver, masking included: a masked caller's rows must be withheld by the
// engine that returns a statement's untraced rows, and the adapter's job is to
// say that masking is on.
func TestQueryOptionsCarriesTheRequest(t *testing.T) {
	options := util.MySQLForkQueryOptions(db.QueryContext{
		Limit:                10,
		Explain:              true,
		MaximumSQLResultSize: 1024,
		MaskingEnabled:       true,
	})
	require.Equal(t, 10, options.Limit)
	require.True(t, options.Explain)
	require.Equal(t, int64(1024), options.MaxResultBytes)
	require.True(t, options.MaskingEnabled)
}
