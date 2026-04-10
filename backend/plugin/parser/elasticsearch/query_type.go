package elasticsearch

import (
	es "github.com/bytebase/omni/elasticsearch"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ClassifyRequest determines the QueryType for an ElasticSearch REST API request.
func ClassifyRequest(method, url string) base.QueryType {
	return convertQueryType(es.ClassifyRequest(method, url))
}

// convertQueryType maps an omni QueryType to a bytebase base.QueryType.
func convertQueryType(qt es.QueryType) base.QueryType {
	switch qt {
	case es.QueryTypeSelect:
		return base.Select
	case es.QueryTypeExplain:
		return base.Explain
	case es.QueryTypeSelectInfoSchema:
		return base.SelectInfoSchema
	case es.QueryTypeDDL:
		return base.DDL
	case es.QueryTypeDML:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
}
