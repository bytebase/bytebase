package elasticsearch

import (
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// readOnlyPostEndpoints lists URL patterns that are read-only despite using POST.
// These endpoints accept a request body for query parameters but do not modify data.
var readOnlyPostEndpoints = []string{
	"_search",
	"_msearch",
	"_count",
	"_field_caps",
	"_validate/query",
	"_search_shards",
	"_terms_enum",
	"_termvectors",
	"_mtermvectors",
	"_search_template",
	"_msearch_template",
	"_render/template",
	"_rank_eval",
	"_knn_search",
	"_search_mvt",
	"_sql",
	"_esql/query",
	"_eql/search",
	"_fleet/_search",
	"_fleet/_msearch",
	"_graph/explore",
	"rollup_search",
	"async_search",
}

// infoSchemaEndpoints lists URL patterns for cluster/node metadata queries.
var infoSchemaEndpoints = []string{
	"_cat/",
	"_cluster/",
	"_nodes/",
}

// dmlPostEndpoints lists URL patterns for document write operations via POST.
var dmlPostEndpoints = []string{
	"_doc",
	"_create/",
	"_update/",
	"_bulk",
	"_delete_by_query",
	"_update_by_query",
	"_reindex",
}

// ClassifyRequest determines the QueryType for an ElasticSearch REST API request.
func ClassifyRequest(method, url string) base.QueryType {
	method = strings.ToUpper(method)
	urlLower := strings.ToLower(url)

	switch method {
	case "HEAD":
		return base.Select

	case "GET":
		if isInfoSchemaURL(urlLower) {
			return base.SelectInfoSchema
		}
		return base.Select

	case "DELETE":
		if isDocumentURL(urlLower) {
			return base.DML
		}
		return base.DDL

	case "PUT":
		if isDocumentWriteURL(urlLower) {
			return base.DML
		}
		return base.DDL

	case "PATCH":
		return base.DML

	case "POST":
		return classifyPostRequest(urlLower)

	default:
		return base.QueryTypeUnknown
	}
}

func classifyPostRequest(url string) base.QueryType {
	// Explain
	if strings.Contains(url, "_explain/") || strings.HasSuffix(url, "_explain") {
		return base.Explain
	}

	// Info schema (metadata reads)
	if isInfoSchemaURL(url) {
		return base.SelectInfoSchema
	}

	// Read-only POST endpoints
	if isReadOnlyPostURL(url) {
		return base.Select
	}

	// Document writes (DML)
	if isDMLPostURL(url) {
		return base.DML
	}

	// Default: DDL for other POST operations (index management, admin, etc.)
	return base.DDL
}

func isInfoSchemaURL(url string) bool {
	for _, pattern := range infoSchemaEndpoints {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func isReadOnlyPostURL(url string) bool {
	for _, pattern := range readOnlyPostEndpoints {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func isDMLPostURL(url string) bool {
	for _, pattern := range dmlPostEndpoints {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func isDocumentURL(url string) bool {
	// DELETE /{index}/_doc/{id} is DML (document deletion)
	return strings.Contains(url, "_doc/")
}

func isDocumentWriteURL(url string) bool {
	// PUT /{index}/_doc/{id}, PUT /{index}/_create/{id}, PUT /_bulk are DML
	return strings.Contains(url, "_doc/") ||
		strings.Contains(url, "_doc") ||
		strings.Contains(url, "_create/") ||
		strings.Contains(url, "_bulk")
}
