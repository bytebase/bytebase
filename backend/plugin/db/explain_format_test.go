package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainFormat(t *testing.T) {
	tests := []struct {
		format v1pb.QueryOption_ExplainFormat
		want   base.ExplainFormat
	}{
		{v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, base.ExplainFormatDefault},
		{v1pb.QueryOption_TEXT, base.ExplainFormatText},
		{v1pb.QueryOption_JSON, base.ExplainFormatJSON},
		{v1pb.QueryOption_XML, base.ExplainFormatXML},
		{v1pb.QueryOption_YAML, base.ExplainFormatYAML},
	}
	for _, tc := range tests {
		require.Equalf(t, tc.want, ExplainFormat(tc.format), "%v", tc.format)
	}
	// A request with no option at all asks for no particular format.
	require.Equal(t, base.ExplainFormatDefault, ExplainFormat((*v1pb.QueryOption)(nil).GetExplainFormat()))
}

func TestExplainResultFormat(t *testing.T) {
	for _, tc := range []struct {
		name   string
		engine storepb.Engine
		format v1pb.QueryOption_ExplainFormat
		want   v1pb.QueryOption_ExplainFormat
		isPlan bool
	}{
		{name: "postgres default", engine: storepb.Engine_POSTGRES, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "postgres json", engine: storepb.Engine_POSTGRES, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mssql xml", engine: storepb.Engine_MSSQL, format: v1pb.QueryOption_XML, want: v1pb.QueryOption_XML, isPlan: true},
		{name: "spanner default", engine: storepb.Engine_SPANNER, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mysql unknown", engine: storepb.Engine_MYSQL, format: v1pb.QueryOption_TEXT, want: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, isPlan: true},
		// The one format MySQL asks for outright, and so the one it can report.
		{name: "mysql json", engine: storepb.Engine_MYSQL, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mariadb json", engine: storepb.Engine_MARIADB, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mariadb default", engine: storepb.Engine_MARIADB, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "tidb json", engine: storepb.Engine_TIDB, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "oracle json", engine: storepb.Engine_ORACLE, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "oracle default", engine: storepb.Engine_ORACLE, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "clickhouse json", engine: storepb.Engine_CLICKHOUSE, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "trino json", engine: storepb.Engine_TRINO, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		// MongoDB has one plan format, so it is the default as well.
		{name: "mongodb default", engine: storepb.Engine_MONGODB, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "mongodb json", engine: storepb.Engine_MONGODB, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "cockroachdb json", engine: storepb.Engine_COCKROACHDB, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "cockroachdb default", engine: storepb.Engine_COCKROACHDB, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "databricks default", engine: storepb.Engine_DATABRICKS, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "elasticsearch default", engine: storepb.Engine_ELASTICSEARCH, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "elasticsearch json", engine: storepb.Engine_ELASTICSEARCH, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "snowflake json", engine: storepb.Engine_SNOWFLAKE, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "snowflake default", engine: storepb.Engine_SNOWFLAKE, want: v1pb.QueryOption_TEXT, isPlan: true},
		{name: "hive json", engine: storepb.Engine_HIVE, format: v1pb.QueryOption_JSON, want: v1pb.QueryOption_JSON, isPlan: true},
		{name: "bigquery dry run", engine: storepb.Engine_BIGQUERY},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, isPlan := ExplainResultFormat(tc.engine, tc.format)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.isPlan, isPlan)
		})
	}
}

// TestSupportedExplainFormats pins which engines are refused an explain request
// outright. An explain request skips the read-only validation an ordinary query
// gets, on the understanding that the driver turns the statement into a plan, so
// an engine whose driver would run the statement as written has to report no
// format at all.
func TestSupportedExplainFormats(t *testing.T) {
	for _, engine := range []storepb.Engine{
		storepb.Engine_REDIS,
		storepb.Engine_DYNAMODB,
		storepb.Engine_CASSANDRA,
		storepb.Engine_COSMOSDB,
	} {
		require.Emptyf(t, SupportedExplainFormats(engine), "%v", engine)
	}
	// Elasticsearch has one plan and it is its profile, which it states only
	// for a search it ran: explaining there runs the search, and the driver
	// refuses to profile anything that is not one.
	require.Equal(t,
		[]v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_JSON},
		SupportedExplainFormats(storepb.Engine_ELASTICSEARCH))
	// Databricks plans a statement through its own SQL, so it takes the text
	// plan every engine without a reader of its own reports.
	require.Equal(t,
		[]v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT},
		SupportedExplainFormats(storepb.Engine_DATABRICKS))
}

// TestTypedExplainPlan pins how an engine parser's answer about the caller's own
// EXPLAIN reaches the result. A format the parser cannot name is the answer for
// MySQL's omitted FORMAT, and it has to stay unspecified rather than becoming
// text, which the viewer would take as a promise about what came back.
func TestTypedExplainPlan(t *testing.T) {
	for _, tc := range []struct {
		name      string
		format    string
		executed  bool
		isExplain bool
		want      *v1pb.QueryResult_QueryPlan
	}{
		{name: "not an explain at all"},
		{name: "json", format: "json", isExplain: true, want: &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_JSON}},
		{name: "text", format: "text", isExplain: true, want: &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_TEXT}},
		{name: "an executed plan", format: "xml", executed: true, isExplain: true, want: &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_XML, Executed: true}},
		{name: "a format the parser could not name", isExplain: true, want: &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED}},
		{name: "a format the enum does not have", format: "tree", isExplain: true, want: &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, TypedExplainPlan(tc.format, tc.executed, tc.isExplain))
		})
	}
}
