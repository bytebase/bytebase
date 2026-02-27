package elasticsearch

import (
	"encoding/json"
	"slices"
	"strings"
)

// MaskableAPI categorizes ES endpoints for masking decisions.
type MaskableAPI int

const (
	// APIUnsupported means the endpoint is not relevant to masking (no user data).
	APIUnsupported MaskableAPI = iota
	// APIMaskSearch covers _search and _msearch.
	APIMaskSearch
	// APIMaskGetDoc covers GET /<index>/_doc/<id>.
	APIMaskGetDoc
	// APIMaskGetSource covers GET /<index>/_source/<id>.
	APIMaskGetSource
	// APIMaskMGet covers _mget.
	APIMaskMGet
	// APIMaskExplain covers _explain.
	APIMaskExplain
	// APIBlocked means the endpoint must be rejected when masking is active.
	APIBlocked
)

// BlockedFeature identifies a feature in the request body that must be blocked.
type BlockedFeature int

const (
	BlockedFeatureAggs BlockedFeature = iota
	BlockedFeatureSuggest
	BlockedFeatureScriptFields
	BlockedFeatureRuntimeMappings
	BlockedFeatureStoredFields
	BlockedFeatureDocvalueFields
)

// BlockedFeatureNames maps BlockedFeature to human-readable names for error messages.
var BlockedFeatureNames = map[BlockedFeature]string{
	BlockedFeatureAggs:            "aggregations",
	BlockedFeatureSuggest:         "suggest",
	BlockedFeatureScriptFields:    "script_fields",
	BlockedFeatureRuntimeMappings: "runtime_mappings",
	BlockedFeatureStoredFields:    "stored_fields",
	BlockedFeatureDocvalueFields:  "docvalue_fields",
}

// RequestAnalysis is the result of analyzing an ES REST request for masking.
type RequestAnalysis struct {
	// API is the classification of the endpoint.
	API MaskableAPI
	// Index is the target index name extracted from the URL (empty if not applicable).
	Index string
	// BlockedFeatures lists features found in the request body that must be blocked.
	BlockedFeatures []BlockedFeature
	// SourceFields is the list of fields requested via _source filtering (nil = all fields).
	// An empty slice means _source is disabled.
	SourceFields []string
	// SourceDisabled is true when _source: false is set.
	SourceDisabled bool
	// RequestedFields is the list of fields requested via the "fields" parameter.
	RequestedFields []string
	// HighlightFields is the list of fields requested via "highlight.fields".
	HighlightFields []string
	// SortFields is the list of field paths used in "sort".
	SortFields []string
	// HasInnerHits is true if the query contains inner_hits.
	HasInnerHits bool
	// PredicateFields is the list of field paths used in query predicates.
	PredicateFields []string
}

// blockedEndpoints lists URL patterns that must be rejected when masking is active.
// These overlap with read patterns but cannot be safely masked.
// Includes both Elasticsearch and OpenSearch equivalents.
var blockedEndpoints = []string{
	// Elasticsearch endpoints.
	"_async_search",
	"_search/scroll",
	"_search_template",
	"_msearch/template",
	"_sql",
	"_eql/search",
	"_esql/query",
	"_terms_enum",
	"_termvectors",
	"_mtermvectors",
	"_knn_search",
	// OpenSearch equivalents (plugin-based paths).
	"_plugins/_asynchronous_search",
	"_plugins/_sql",
	"_plugins/_ppl",
}

// classifyMaskableAPI determines the MaskableAPI type and target index for an ES request.
func classifyMaskableAPI(method, url string) (MaskableAPI, string) {
	method = strings.ToUpper(method)

	// Strip query parameters.
	path := url
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}

	// Strip leading slash.
	path = strings.TrimPrefix(path, "/")

	// Only GET and POST can return user data.
	switch method {
	case "GET":
		if blocked, index := isBlockedEndpoint(path); blocked {
			return APIBlocked, index
		}
		return classifyGetForMasking(path)
	case "POST":
		if blocked, index := isBlockedEndpoint(path); blocked {
			return APIBlocked, index
		}
		return classifyPostForMasking(path)
	default:
		// HEAD, PUT, DELETE, PATCH do not return user data.
		return APIUnsupported, ""
	}
}

// isBlockedEndpoint checks whether the path matches a blocked endpoint pattern.
// Returns true and the extracted index if blocked.
func isBlockedEndpoint(path string) (bool, string) {
	for _, pattern := range blockedEndpoints {
		if strings.Contains(path, pattern) {
			return true, extractIndex(path)
		}
	}
	return false, ""
}

// classifyGetForMasking classifies a GET request path for masking.
func classifyGetForMasking(path string) (MaskableAPI, string) {
	for part := range strings.SplitSeq(path, "/") {
		switch part {
		case "_search", "_msearch":
			return APIMaskSearch, extractIndex(path)
		case "_doc":
			return APIMaskGetDoc, extractIndex(path)
		case "_source":
			return APIMaskGetSource, extractIndex(path)
		case "_mget":
			return APIMaskMGet, extractIndex(path)
		}
	}

	return APIUnsupported, ""
}

// classifyPostForMasking classifies a POST request path for masking.
func classifyPostForMasking(path string) (MaskableAPI, string) {
	for part := range strings.SplitSeq(path, "/") {
		switch part {
		case "_search", "_msearch":
			return APIMaskSearch, extractIndex(path)
		case "_mget":
			return APIMaskMGet, extractIndex(path)
		case "_explain":
			return APIMaskExplain, extractIndex(path)
		}
	}

	return APIUnsupported, ""
}

// extractIndex returns the first path segment if it does not start with "_".
// This extracts the index name from ES URL paths like "<index>/_search".
func extractIndex(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if strings.HasPrefix(first, "_") {
		return ""
	}
	return first
}

// AnalyzeRequest combines URL classification with body analysis for an ES request.
// Body analysis is only performed for search and explain APIs.
func AnalyzeRequest(method, url, body string) *RequestAnalysis {
	api, index := classifyMaskableAPI(method, url)
	result := &RequestAnalysis{
		API:   api,
		Index: index,
	}

	// Only analyze the body for APIs that accept a query body.
	switch api {
	case APIMaskSearch, APIMaskExplain:
		bodyResult := analyzeRequestBody(body)
		result.BlockedFeatures = bodyResult.BlockedFeatures
		result.SourceFields = bodyResult.SourceFields
		result.SourceDisabled = bodyResult.SourceDisabled
		result.RequestedFields = bodyResult.RequestedFields
		result.HighlightFields = bodyResult.HighlightFields
		result.SortFields = bodyResult.SortFields
		result.HasInnerHits = bodyResult.HasInnerHits
		result.PredicateFields = bodyResult.PredicateFields
	default:
		// No body analysis for other API types.
	}

	return result
}

// analyzeRequestBody parses the JSON body and extracts fields and blocked features.
func analyzeRequestBody(body string) *RequestAnalysis {
	result := &RequestAnalysis{}

	parsed := make(map[string]any)
	if json.Unmarshal([]byte(body), &parsed) != nil {
		return result
	}

	result.BlockedFeatures = detectBlockedFeatures(parsed)
	extractSource(parsed, result)
	result.RequestedFields = extractStringArray(parsed, "fields")
	result.HighlightFields = extractHighlightFields(parsed)
	result.SortFields = extractSortFields(parsed)
	result.HasInnerHits = containsKey(parsed, "inner_hits")
	result.PredicateFields = extractPredicateFields(parsed)

	return result
}

// detectBlockedFeatures checks for top-level keys that indicate blocked features.
func detectBlockedFeatures(parsed map[string]any) []BlockedFeature {
	var blocked []BlockedFeature

	if _, ok := parsed["aggs"]; ok {
		blocked = append(blocked, BlockedFeatureAggs)
	} else if _, ok := parsed["aggregations"]; ok {
		blocked = append(blocked, BlockedFeatureAggs)
	}
	if _, ok := parsed["suggest"]; ok {
		blocked = append(blocked, BlockedFeatureSuggest)
	}
	if _, ok := parsed["script_fields"]; ok {
		blocked = append(blocked, BlockedFeatureScriptFields)
	}
	if _, ok := parsed["runtime_mappings"]; ok {
		blocked = append(blocked, BlockedFeatureRuntimeMappings)
	}
	if _, ok := parsed["stored_fields"]; ok {
		blocked = append(blocked, BlockedFeatureStoredFields)
	}
	if _, ok := parsed["docvalue_fields"]; ok {
		blocked = append(blocked, BlockedFeatureDocvalueFields)
	}

	return blocked
}

// extractSource handles the three forms of _source in the request body.
func extractSource(parsed map[string]any, result *RequestAnalysis) {
	src, ok := parsed["_source"]
	if !ok {
		return
	}

	switch v := src.(type) {
	case bool:
		if !v {
			result.SourceDisabled = true
		}
	case []any:
		result.SourceFields = toStringSlice(v)
	case map[string]any:
		if includes, ok := v["includes"]; ok {
			if arr, ok := includes.([]any); ok {
				result.SourceFields = toStringSlice(arr)
			}
		}
	}
}

// extractStringArray extracts a string array value from a top-level key.
func extractStringArray(parsed map[string]any, key string) []string {
	val, ok := parsed[key]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	return toStringSlice(arr)
}

// extractHighlightFields extracts field names from highlight.fields.
func extractHighlightFields(parsed map[string]any) []string {
	highlight, ok := parsed["highlight"]
	if !ok {
		return nil
	}
	hlMap, ok := highlight.(map[string]any)
	if !ok {
		return nil
	}
	fields, ok := hlMap["fields"]
	if !ok {
		return nil
	}
	fieldsMap, ok := fields.(map[string]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(fieldsMap))
	for k := range fieldsMap {
		result = append(result, k)
	}
	slices.Sort(result)
	return result
}

// extractSortFields extracts field names from the sort array.
func extractSortFields(parsed map[string]any) []string {
	sortVal, ok := parsed["sort"]
	if !ok {
		return nil
	}
	arr, ok := sortVal.([]any)
	if !ok {
		return nil
	}
	var fields []string
	for _, item := range arr {
		switch v := item.(type) {
		case string:
			if strings.HasPrefix(v, "_") {
				continue
			}
			fields = append(fields, v)
		case map[string]any:
			for k := range v {
				if !strings.HasPrefix(k, "_") {
					fields = append(fields, k)
				}
			}
		}
	}
	return fields
}

// containsKey recursively searches for a key in a nested map structure.
func containsKey(data map[string]any, key string) bool {
	for k, v := range data {
		if k == key {
			return true
		}
		if containsKeyInValue(v, key) {
			return true
		}
	}
	return false
}

// containsKeyInValue checks if any nested map within v contains the given key.
func containsKeyInValue(v any, key string) bool {
	switch child := v.(type) {
	case map[string]any:
		return containsKey(child, key)
	case []any:
		for _, item := range child {
			if m, ok := item.(map[string]any); ok && containsKey(m, key) {
				return true
			}
		}
	}
	return false
}

// toStringSlice converts a []any to a []string, skipping non-string elements.
func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// directFieldClauses are clause types where the keys inside the clause object are field names.
var directFieldClauses = map[string]bool{
	"match": true, "match_phrase": true, "match_phrase_prefix": true, "match_bool_prefix": true,
	"term": true, "terms": true, "terms_set": true,
	"range":    true,
	"wildcard": true, "prefix": true, "fuzzy": true, "regexp": true,
	"geo_distance": true, "geo_bounding_box": true, "geo_shape": true, "geo_polygon": true,
	"span_term": true,
}

// compoundClauses maps clause types to the keys that contain sub-queries.
var compoundClauses = map[string][]string{
	"bool":           {"must", "must_not", "should", "filter"},
	"nested":         {"query"},
	"has_child":      {"query"},
	"has_parent":     {"query"},
	"dis_max":        {"queries"},
	"constant_score": {"filter"},
	"boosting":       {"positive", "negative"},
	"function_score": {"query"},
}

// extractPredicateFields extracts field names from ES query clauses.
func extractPredicateFields(parsed map[string]any) []string {
	queryVal, ok := parsed["query"]
	if !ok {
		return nil
	}
	queryMap, ok := queryVal.(map[string]any)
	if !ok {
		return nil
	}
	var fields []string
	extractFieldsFromQuery(queryMap, &fields)
	return fields
}

// extractFieldsFromQuery recursively walks ES query clauses to collect field names.
func extractFieldsFromQuery(queryMap map[string]any, fields *[]string) {
	for clauseType, clauseVal := range queryMap {
		clauseObj, ok := clauseVal.(map[string]any)
		if !ok {
			extractFieldsFromQueryArray(clauseVal, fields)
			continue
		}
		switch {
		case directFieldClauses[clauseType]:
			extractDirectFieldNames(clauseObj, fields)
		case clauseType == "exists":
			if fieldVal, ok := clauseObj["field"].(string); ok {
				*fields = append(*fields, fieldVal)
			}
		case compoundClauses[clauseType] != nil:
			extractCompoundClauseFields(clauseObj, compoundClauses[clauseType], fields)
		default:
			extractFieldsFromQuery(clauseObj, fields)
		}
	}
}

// extractFieldsFromQueryArray handles the case where a clause value is an array of query maps.
func extractFieldsFromQueryArray(val any, fields *[]string) {
	arr, ok := val.([]any)
	if !ok {
		return
	}
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			extractFieldsFromQuery(m, fields)
		}
	}
}

// extractDirectFieldNames extracts field names from a direct field clause object.
func extractDirectFieldNames(clauseObj map[string]any, fields *[]string) {
	for fieldName := range clauseObj {
		if !strings.HasPrefix(fieldName, "_") && fieldName != "boost" {
			*fields = append(*fields, fieldName)
		}
	}
}

// extractCompoundClauseFields recurses into sub-query keys of a compound clause.
func extractCompoundClauseFields(clauseObj map[string]any, subQueryKeys []string, fields *[]string) {
	for _, sqKey := range subQueryKeys {
		sqVal, ok := clauseObj[sqKey]
		if !ok {
			continue
		}
		switch sq := sqVal.(type) {
		case map[string]any:
			extractFieldsFromQuery(sq, fields)
		case []any:
			extractFieldsFromQueryArray(sq, fields)
		}
	}
}
