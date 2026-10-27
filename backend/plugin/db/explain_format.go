package db

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ExplainFormat maps the plan format a caller asked for onto the format the
// parsers take. The statement an explain request runs is built there — see
// base.ExplainStatement — and this is the only part of that which speaks the
// API's enum.
func ExplainFormat(format v1pb.QueryOption_ExplainFormat) base.ExplainFormat {
	switch format {
	case v1pb.QueryOption_TEXT:
		return base.ExplainFormatText
	case v1pb.QueryOption_JSON:
		return base.ExplainFormatJSON
	case v1pb.QueryOption_XML:
		return base.ExplainFormatXML
	case v1pb.QueryOption_YAML:
		return base.ExplainFormatYAML
	default:
		return base.ExplainFormatDefault
	}
}

// SupportedExplainFormats lists the formats an engine's explain path accepts.
func SupportedExplainFormats(engine storepb.Engine) []v1pb.QueryOption_ExplainFormat {
	switch engine {
	case storepb.Engine_POSTGRES:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON, v1pb.QueryOption_XML, v1pb.QueryOption_YAML}
	case storepb.Engine_MSSQL:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_XML}
	case storepb.Engine_SPANNER, storepb.Engine_MONGODB:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_JSON}
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB,
		storepb.Engine_ORACLE, storepb.Engine_CLICKHOUSE, storepb.Engine_TRINO,
		storepb.Engine_OCEANBASE, storepb.Engine_COCKROACHDB, storepb.Engine_SNOWFLAKE,
		storepb.Engine_HIVE:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON}
	// These engines have no plan to ask for: CQL and the document and key-value
	// APIs state none, and an engine listed here is refused an explain request
	// outright, since its driver would otherwise run the statement as written.
	case storepb.Engine_REDIS, storepb.Engine_DYNAMODB,
		storepb.Engine_CASSANDRA, storepb.Engine_COSMOSDB,
		storepb.Engine_ELASTICSEARCH:
		return nil
	default:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT}
	}
}

// ExplainResultFormat returns the format of a query plan result. The boolean is
// false when the engine uses the explain path for something other than a plan.
func ExplainResultFormat(engine storepb.Engine, format v1pb.QueryOption_ExplainFormat) (v1pb.QueryOption_ExplainFormat, bool) {
	if engine == storepb.Engine_BIGQUERY {
		return v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, false
	}
	// MySQL has no text format to name: the plain EXPLAIN that every other
	// format builds follows the session's explain_format from 8.0.32, so the
	// JSON we ask for outright is the only answer the request settles.
	if engine == storepb.Engine_MYSQL && format != v1pb.QueryOption_JSON {
		return v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, true
	}
	if format != v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED {
		return format, true
	}
	if engine == storepb.Engine_SPANNER || engine == storepb.Engine_MONGODB {
		return v1pb.QueryOption_JSON, true
	}
	return v1pb.QueryOption_TEXT, true
}

// TypedExplainPlan describes the plan a result carries because the caller's own
// statement asked for one, from the engine parser's DescribeExplain:
//
//	r.QueryPlan = db.TypedExplainPlan(mysqlparser.DescribeExplain(statement))
//
// A format the enum does not name is left unspecified, which is the answer for
// MySQL's omitted FORMAT: it follows the session's explain_format, so the
// statement does not settle it, and the viewer reads unknown as text.
func TypedExplainPlan(format string, executed, isExplain bool) *v1pb.QueryResult_QueryPlan {
	if !isExplain {
		return nil
	}
	return &v1pb.QueryResult_QueryPlan{
		Format:   v1pb.QueryOption_ExplainFormat(v1pb.QueryOption_ExplainFormat_value[strings.ToUpper(format)]),
		Executed: executed,
	}
}
