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
	return base.ConvertQueryType(t, analysis.Select, analysis.Explain, analysis.SelectInfoSchema, analysis.DDL, analysis.DML)
}
