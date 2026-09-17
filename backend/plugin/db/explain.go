package db

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ExplainFormats returns the plan formats a request may name for engine, and
// the format of a plan when the request names none. ok is false when engine
// cannot explain.
func ExplainFormats(engine storepb.Engine) (formats []v1pb.QueryOption_ExplainFormat, defaultFormat v1pb.QueryOption_ExplainFormat, ok bool) {
	text := []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT}
	switch engine {
	case storepb.Engine_POSTGRES:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON, v1pb.QueryOption_XML, v1pb.QueryOption_YAML}, v1pb.QueryOption_TEXT, true
	case storepb.Engine_MSSQL:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_XML}, v1pb.QueryOption_TEXT, true
	case storepb.Engine_SPANNER:
		// Spanner returns its plan as JSON and has no text form.
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_JSON}, v1pb.QueryOption_JSON, true
	case storepb.Engine_MYSQL:
		// MySQL 8.0.32 and later answer an EXPLAIN without FORMAT in the
		// session's explain_format.
		return text, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, true
	case storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
		storepb.Engine_TIDB,
		storepb.Engine_REDSHIFT,
		storepb.Engine_COCKROACHDB,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_STARROCKS,
		storepb.Engine_DORIS,
		storepb.Engine_HIVE,
		storepb.Engine_TRINO,
		storepb.Engine_ORACLE,
		storepb.Engine_BIGQUERY:
		return text, v1pb.QueryOption_TEXT, true
	default:
		return nil, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, false
	}
}
