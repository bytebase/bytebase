package elasticsearch

import (
	es "github.com/bytebase/omni/elasticsearch"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// convertQueryType maps an omni QueryType to a bytebase base.QueryType.
func convertQueryType(qt es.QueryType) base.QueryType {
	return base.ConvertQueryType(qt, es.QueryTypeSelect, es.QueryTypeExplain, es.QueryTypeSelectInfoSchema, es.QueryTypeDDL, es.QueryTypeDML)
}
