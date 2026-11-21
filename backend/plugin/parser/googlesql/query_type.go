package googlesql

import (
	"github.com/bytebase/omni/googlesql/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// MapQueryType maps an omni analysis.QueryType onto bytebase's base.QueryType.
func MapQueryType(t analysis.QueryType) base.QueryType {
	return base.OmniQueryType(t)
}
