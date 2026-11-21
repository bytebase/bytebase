package doris

import (
	"errors"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Register the Doris editor validator.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/doris"
)

// TestDriverConfig pins the MySQL family's connection mapping: the data source's
// host, port and login, the request's database, the editor session's app name,
// and no TLS material or tunnel when the data source names neither.
func TestDriverConfig(t *testing.T) {
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "doris.example.com",
			Port:     "9030",
			Username: "bytebase",
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName:  "analytics",
			EngineVersion: "3.1.4",
		},
	}

	cfg, err := driverConfig(config, &util.OmniAdapter{})
	require.NoError(t, err)
	require.Equal(t, "doris.example.com", cfg.Host)
	require.Equal(t, "9030", cfg.Port)
	require.Equal(t, "analytics", cfg.Database)
	require.Equal(t, "bytebase", cfg.User)
	require.Equal(t, "secret", cfg.Password)
	require.Equal(t, "bytebase", cfg.ApplicationName)
	require.Equal(t, "3.1.4", cfg.EngineVersion)
	require.Nil(t, cfg.Dial)
	require.Nil(t, cfg.TLS)
	require.Empty(t, cfg.Params)
	require.False(t, cfg.ReadOnly)
}

// TestDriverConfigTLSPassesTheResolvedMaterial pins that the TLS material Bytebase
// resolved reaches every dial as the same config: the verification is built for
// the one host the data source names.
func TestDriverConfigTLSPassesTheResolvedMaterial(t *testing.T) {
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                 "doris.example.com",
			Port:                 "9030",
			Username:             "bytebase",
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                testCACertificate,
		},
		Password: "secret",
	}

	cfg, err := driverConfig(config, &util.OmniAdapter{})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)

	fromFirst, err := cfg.TLS("doris.example.com")
	require.NoError(t, err)
	fromSecond, err := cfg.TLS("another.example.com")
	require.NoError(t, err)
	require.Same(t, fromFirst, fromSecond)
	// Bytebase's verifying material is bound to the data source's own host and
	// verifies the chain itself, so it disables Go's built-in verification.
	require.NotNil(t, fromFirst.VerifyPeerCertificate)
}

// TestDriverConfigTLSWithoutVerification pins the data source's default: TLS is
// required but the server's certificate is not verified.
func TestDriverConfigTLSWithoutVerification(t *testing.T) {
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "doris.example.com",
			Port:     "9030",
			Username: "bytebase",
			UseSsl:   true,
		},
	}

	cfg, err := driverConfig(config, &util.OmniAdapter{})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	tlscfg, err := cfg.TLS("doris.example.com")
	require.NoError(t, err)
	require.True(t, tlscfg.InsecureSkipVerify)
}

// TestExecuteOptions pins the transaction mapping. The isolation level stays
// empty whatever the script's "-- txn-isolation" directive said: this driver has
// never applied one, and Doris's only isolation level is READ COMMITTED, so the
// request cannot be honoured as written.
func TestExecuteOptions(t *testing.T) {
	tests := []struct {
		name        string
		statement   string
		opts        db.ExecuteOptions
		wantMode    omnidriver.TransactionMode
		wantCreate  bool
		wantRetries int
	}{
		{
			name:      "the default wraps the script in a transaction",
			statement: "SELECT 1;",
			wantMode:  omnidriver.Transactional,
		},
		{
			name:      "-- txn-mode=off runs each statement on its own",
			statement: "-- txn-mode=off\nSELECT 1;",
			wantMode:  omnidriver.AutoCommit,
		},
		{
			name:        "create database runs outside a transaction",
			statement:   "-- txn-mode=on\n-- txn-isolation=serializable\nCREATE DATABASE app;",
			opts:        db.ExecuteOptions{CreateDatabase: true, MaximumRetries: 3},
			wantMode:    omnidriver.Transactional,
			wantCreate:  true,
			wantRetries: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, got := executeOptions(tc.statement, tc.opts)
			require.Equal(t, tc.wantMode, got.Transaction)
			require.Empty(t, got.Isolation)
			require.Equal(t, tc.wantCreate, got.CreateDatabase)
			require.Equal(t, tc.wantRetries, got.MaxRetries)
			require.NotNil(t, got.Log)
		})
	}
}

// TestExecuteLogger pins what each step of a script logs: one command entry with
// its byte range, the rows it affected, and the transaction control.
func TestExecuteLogger(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		// The task-run sheet sends the statement, not its range.
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	}
	l := util.NewExecuteLogger(opts)

	l.CommandExecute(omnidriver.Command{Text: "SELECT 1", ByteStart: 2, ByteEnd: 12})
	l.CommandResponse(1, []int64{1}, nil)
	l.TransactionControl(omnidriver.Begin, nil)
	l.TransactionControl(omnidriver.Rollback, errors.New("boom"))

	require.Len(t, logs, 4)
	require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
	require.Equal(t, "SELECT 1", logs[0].CommandExecute.Statement)
	require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
	require.Equal(t, int64(1), logs[1].CommandResponse.AffectedRows)
	require.Equal(t, storepb.TaskRunLog_TRANSACTION_CONTROL, logs[2].Type)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[2].TransactionControl.Type)
	require.Empty(t, logs[2].TransactionControl.Error)
	require.Equal(t, storepb.TaskRunLog_TRANSACTION_CONTROL, logs[3].Type)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_ROLLBACK, logs[3].TransactionControl.Type)
	require.Equal(t, "boom", logs[3].TransactionControl.Error)
}

// TestToRowValue pins the cell shapes the editor renders for each kind the Doris
// driver decodes.
func TestToRowValue(t *testing.T) {
	wallClock := time.Date(2026, 11, 17, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name  string
		value omnidriver.Value
		want  *v1pb.RowValue
	}{
		{
			name:  "null",
			value: omnidriver.Value{Kind: omnidriver.Null},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		},
		{
			name:  "bool",
			value: omnidriver.Value{Kind: omnidriver.Bool, Bool: true},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}},
		},
		{
			name:  "bytes",
			value: omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte{0x01}},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte{0x01}}},
		},
		{
			// The old path read a zero-length BIT/VARBIT value as NULL; keep that.
			name:  "zero-byte value is a null cell",
			value: omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte{}},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		},
		{
			name:  "double",
			value: omnidriver.Value{Kind: omnidriver.Float64, Float64: 1.5},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 1.5}},
		},
		{
			name:  "int64",
			value: omnidriver.Value{Kind: omnidriver.Int64, Int64: 42},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}},
		},
		{
			name:  "string",
			value: omnidriver.Value{Kind: omnidriver.String, String: "abc"},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "abc"}},
		},
		{
			name:  "timestamp keeps its fractional digits",
			value: omnidriver.Value{Kind: omnidriver.Timestamp, Time: wallClock, Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(wallClock),
				Accuracy:        3,
			}}},
		},
		{
			// Doris's driver has no other kind; a shape another engine decodes is
			// a null cell here rather than a crash.
			name:  "a kind doris does not produce",
			value: omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: wallClock},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, toRowValue(tc.value))
		})
	}
}

// TestToQueryResult pins the result shapes: a count for a statement with no rows,
// columns and rows for a result set, the truncation message at the cap, and the
// statement's own error in place of the rows.
func TestToQueryResult(t *testing.T) {
	latency := 5 * time.Millisecond

	t.Run("affected rows", func(t *testing.T) {
		affected := int64(3)
		got := toQueryResult(&omnidriver.Result{
			Statement:    "INSERT INTO t VALUES (1), (2), (3)",
			AffectedRows: &affected,
			Latency:      latency,
		}, 0)
		require.Equal(t, "INSERT INTO t VALUES (1), (2), (3)", got.Statement)
		require.Equal(t, durationpb.New(latency), got.Latency)
		require.Equal(t, int64(1), got.RowsCount)
		require.Len(t, got.Rows, 1)
		require.Equal(t, "Affected Rows", got.ColumnNames[0])
		require.Equal(t, int64(3), got.Rows[0].Values[0].GetInt64Value())
	})

	t.Run("columns and rows", func(t *testing.T) {
		got := toQueryResult(&omnidriver.Result{
			Statement: "SELECT id, name FROM t",
			Columns: []omnidriver.Column{
				{Name: "id", TypeName: "INT"},
				{Name: "name", TypeName: "VARCHAR"},
			},
			Rows: [][]omnidriver.Value{
				{{Kind: omnidriver.Int64, Int64: 1}, {Kind: omnidriver.String, String: "a"}},
				{{Kind: omnidriver.Int64, Int64: 2}, {Kind: omnidriver.Null}},
			},
			Latency: latency,
		}, 0)
		require.Equal(t, []string{"id", "name"}, got.ColumnNames)
		require.Equal(t, []string{"INT", "VARCHAR"}, got.ColumnTypeNames)
		require.Equal(t, int64(2), got.RowsCount)
		require.Len(t, got.Rows, 2)
		require.Empty(t, got.Error)
	})

	t.Run("truncated at the cap", func(t *testing.T) {
		got := toQueryResult(&omnidriver.Result{
			Statement: "SELECT * FROM t",
			Columns:   []omnidriver.Column{{Name: "a", TypeName: "VARCHAR"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.String, String: "x"}}},
			Truncated: true,
		}, 1024)
		require.Equal(t, util.FormatMaximumSQLResultSizeMessage(1024), got.Error)
	})

	t.Run("statement error", func(t *testing.T) {
		got := toQueryResult(&omnidriver.Result{
			Statement: "SELECT * FROM missing",
			Err:       errors.New("Table [missing] does not exist"),
		}, 0)
		require.Equal(t, "Table [missing] does not exist", got.Error)
		require.Empty(t, got.Rows)
	})
}

// TestValidateSQLForEditorUsesDorisGrammar pins this driver to the Doris parser: a
// StarRocks-only spelling must be refused rather than quietly accepted, which is what
// routing the connection through a sibling engine's grammar would do.
func TestValidateSQLForEditorUsesDorisGrammar(t *testing.T) {
	// StarRocks's primary key table model; Doris has no PRIMARY KEY model.
	const starRocksOnlyDDL = "CREATE TABLE t (a INT) PRIMARY KEY(a) DISTRIBUTED BY HASH(a)"
	_, _, err := base.ValidateSQLForEditor(storepb.Engine_DORIS, starRocksOnlyDDL)
	require.Error(t, err)

	// A Doris CREATE TABLE parses, and is DDL rather than a read-only query.
	_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_DORIS, "CREATE TABLE t (a INT) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)")
	require.NoError(t, err)
	require.False(t, allQuery)
}

// testCACertificate is a self-signed certificate's PEM, used only to build a
// verifying TLS config.
const testCACertificate = `-----BEGIN CERTIFICATE-----
MIIBpzCCAUygAwIBAgIUfJ2rDKzqCP9t+bWDuGZj04cN/5QwCgYIKoZIzj0EAwIw
FDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI2MTExODA0MDczMFoXDTM2MTExNTA0
MDczMFowFDESMBAGA1UEAwwJbG9jYWxob3N0MFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAERLQGtvJGM2Mto0fIgDaQwQFlOSksNdpx0c1H9Y+bY/qf6LanLrjFZt9f
e+51wPk+WP29FWBmI8wd2SOUUQ0gVqN8MHowHQYDVR0OBBYEFG8EK3nhHsi9TYt9
aYus2uKvYPzWMB8GA1UdIwQYMBaAFG8EK3nhHsi9TYt9aYus2uKvYPzWMA8GA1Ud
EwEB/wQFMAMBAf8wJwYDVR0RBCAwHoIJbG9jYWxob3N0ghFkb3Jpcy5leGFtcGxl
LmNvbTAKBggqhkjOPQQDAgNJADBGAiEAw0gLqM7TOqf0j0p7tjhsH1UKCceIVCQY
/BQBsf/k7fgCIQDKZ/on64l/759Dj5nDzziQ2LjnT/6ScLS7W65WXAUv5Q==
-----END CERTIFICATE-----`

// TestQueryOptionsCarriesTheRequest pins the mapping QueryConn hands the
// driver, masking included: a masked caller's rows must be withheld by the
// engine that returns a statement's untraced rows, and the adapter's job is to
// say that masking is on.
func TestQueryOptionsCarriesTheRequest(t *testing.T) {
	options := queryOptions(db.QueryContext{
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
