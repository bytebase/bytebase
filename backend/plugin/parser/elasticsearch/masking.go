package elasticsearch

import (
	"github.com/bytebase/omni/elasticsearch/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Type aliases so that internal code and external consumers (e.g. catalog_masking_elasticsearch.go)
// can continue using the short names. The canonical definitions live in the base package.
type (
	MaskableAPI     = base.ElasticsearchMaskableAPI
	BlockedFeature  = base.ElasticsearchBlockedFeature
	RequestAnalysis = base.ElasticsearchAnalysis
)

const (
	APIUnsupported   = base.ElasticsearchAPIUnsupported
	APIMaskSearch    = base.ElasticsearchAPIMaskSearch
	APIMaskGetDoc    = base.ElasticsearchAPIMaskGetDoc
	APIMaskGetSource = base.ElasticsearchAPIMaskGetSource
	APIMaskMGet      = base.ElasticsearchAPIMaskMGet
	APIMaskExplain   = base.ElasticsearchAPIMaskExplain
	APIBlocked       = base.ElasticsearchAPIBlocked
)

const (
	BlockedFeatureAggs            = base.ElasticsearchBlockedFeatureAggs
	BlockedFeatureSuggest         = base.ElasticsearchBlockedFeatureSuggest
	BlockedFeatureScriptFields    = base.ElasticsearchBlockedFeatureScriptFields
	BlockedFeatureRuntimeMappings = base.ElasticsearchBlockedFeatureRuntimeMappings
	BlockedFeatureStoredFields    = base.ElasticsearchBlockedFeatureStoredFields
	BlockedFeatureDocvalueFields  = base.ElasticsearchBlockedFeatureDocvalueFields
)

// BlockedFeatureNames maps BlockedFeature to human-readable names for error messages.
var BlockedFeatureNames = base.ElasticsearchBlockedFeatureNames

// AnalyzeRequest combines URL classification with body analysis for an ES request.
// Body analysis is only performed for search and explain APIs.
func AnalyzeRequest(method, url, body string) *RequestAnalysis {
	omniResult := analysis.AnalyzeRequest(method, url, body)
	return convertAnalysis(omniResult)
}

// classifyMaskableAPI determines the MaskableAPI type and target index for an ES request.
func classifyMaskableAPI(method, url string) (MaskableAPI, string) {
	omniAPI, index := analysis.ClassifyMaskableAPI(method, url)
	return convertMaskableAPI(omniAPI), index
}

// analyzeRequestBody parses the JSON body and extracts fields and blocked features.
// This is an internal adapter kept for test compatibility.
func analyzeRequestBody(body string) *RequestAnalysis {
	// To match the omni behavior exactly, we use a search endpoint to trigger body analysis.
	omniResult := analysis.AnalyzeRequest("POST", "test/_search", body)
	return convertAnalysis(omniResult)
}
