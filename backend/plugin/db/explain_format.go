package db

import (
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
	case storepb.Engine_SPANNER:
		return []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_JSON}
	case storepb.Engine_MONGODB, storepb.Engine_REDIS, storepb.Engine_DYNAMODB,
		storepb.Engine_CASSANDRA, storepb.Engine_COSMOSDB, storepb.Engine_DATABRICKS,
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
	if engine == storepb.Engine_MYSQL {
		return v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, true
	}
	if format != v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED {
		return format, true
	}
	if engine == storepb.Engine_SPANNER {
		return v1pb.QueryOption_JSON, true
	}
	return v1pb.QueryOption_TEXT, true
}
