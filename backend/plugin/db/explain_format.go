package db

import (
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
