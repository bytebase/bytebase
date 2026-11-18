package hive

import (
	"errors"
	"testing"

	odriver "github.com/bytebase/omni/driver"
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
	require.Equal(t, "JSON", explainFormat(v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", explainFormat(v1pb.QueryOption_TEXT))
	require.Equal(t, "TEXT", explainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED))
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
func TestToQueryResult(t *testing.T) {
	result := &odriver.Result{
		Statement: "SELECT 1",
		Columns:   []odriver.Column{{Name: "id", TypeName: "int"}, {Name: "note", TypeName: "string"}},
		Rows:      [][]odriver.Value{{{Kind: odriver.Int32, Int64: 1}, {Kind: odriver.String, String: "a"}}},
	}
	converted := toQueryResult(result, 0)
	require.Equal(t, "SELECT 1", converted.Statement)
	require.Equal(t, []string{"id", "note"}, converted.ColumnNames)
	require.Equal(t, []string{"INT", "STRING"}, converted.ColumnTypeNames)
	require.Equal(t, int64(1), converted.RowsCount)
	require.Len(t, converted.Rows, 1)
	require.Empty(t, converted.Error)
	require.Nil(t, converted.QueryPlan)

	result.Truncated = true
	result.Err = errors.New("boom")
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(64), toQueryResult(result, 64).Error)

	result.Truncated = false
	require.Equal(t, "boom", toQueryResult(result, 64).Error)
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
