package googlesql

import (
	"github.com/bytebase/omni/googlesql/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// MapQueryType maps an omni analysis.QueryType onto bytebase's base.QueryType.
// The two enums are defined to correspond 1:1 (omni's QueryType doc states the
// values mirror base.QueryType so this switch is total); an unexpected value
// falls back to QueryTypeUnknown.
func MapQueryType(t analysis.QueryType) base.QueryType {
	switch t {
	case analysis.Select:
		return base.Select
	case analysis.Explain:
		return base.Explain
	case analysis.SelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.DDL:
		return base.DDL
	case analysis.DML:
		return base.DML
	case analysis.Unknown:
		return base.QueryTypeUnknown
	default:
		return base.QueryTypeUnknown
	}
}
