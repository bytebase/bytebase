package base

// MongoDBMaskableAPI categorizes MongoDB collection reads for masking behavior.
type MongoDBMaskableAPI int

const (
	MongoDBMaskableAPIUnsupported MongoDBMaskableAPI = iota
	MongoDBMaskableAPIFind
	MongoDBMaskableAPIFindOne
	MongoDBMaskableAPIUnsupportedRead
	MongoDBMaskableAPIAggregate
)

// MongoDBJoinedCollection records a $lookup or $graphLookup join extracted from an aggregate pipeline.
type MongoDBJoinedCollection struct {
	AsField    string // The output field name added to each document (the "as" argument).
	Collection string // The source collection being joined (the "from" argument).
}

// MongoDBAnalysis contains MongoDB statement data needed by masking checks.
type MongoDBAnalysis struct {
	API               MongoDBMaskableAPI
	Operation         string
	Collection        string
	PredicateFields   []string
	UnsupportedStage  string                    // First pipeline stage that prevents masking. Only set when API == MongoDBMaskableAPIUnsupportedRead for aggregate pipelines.
	JoinedCollections []MongoDBJoinedCollection // Join info extracted from $lookup and $graphLookup stages.
}

// ElasticsearchMaskableAPI categorizes ES endpoints for masking decisions.
type ElasticsearchMaskableAPI int

const (
	ElasticsearchAPIUnsupported   ElasticsearchMaskableAPI = iota
	ElasticsearchAPIMaskSearch                             // covers _search and _msearch
	ElasticsearchAPIMaskGetDoc                             // covers GET /<index>/_doc/<id>
	ElasticsearchAPIMaskGetSource                          // covers GET /<index>/_source/<id>
	ElasticsearchAPIMaskMGet                               // covers _mget
	ElasticsearchAPIMaskExplain                            // covers _explain
	ElasticsearchAPIBlocked                                // endpoint must be rejected when masking is active
)

// ElasticsearchBlockedFeature identifies a feature in the request body that must be blocked.
type ElasticsearchBlockedFeature int

const (
	ElasticsearchBlockedFeatureAggs ElasticsearchBlockedFeature = iota
	ElasticsearchBlockedFeatureSuggest
	ElasticsearchBlockedFeatureScriptFields
	ElasticsearchBlockedFeatureRuntimeMappings
	ElasticsearchBlockedFeatureStoredFields
	ElasticsearchBlockedFeatureDocvalueFields
)

// ElasticsearchBlockedFeatureNames maps ElasticsearchBlockedFeature to human-readable names for error messages.
var ElasticsearchBlockedFeatureNames = map[ElasticsearchBlockedFeature]string{
	ElasticsearchBlockedFeatureAggs:            "aggregations",
	ElasticsearchBlockedFeatureSuggest:         "suggest",
	ElasticsearchBlockedFeatureScriptFields:    "script_fields",
	ElasticsearchBlockedFeatureRuntimeMappings: "runtime_mappings",
	ElasticsearchBlockedFeatureStoredFields:    "stored_fields",
	ElasticsearchBlockedFeatureDocvalueFields:  "docvalue_fields",
}

// ElasticsearchAnalysis is the result of analyzing an ES REST request for masking.
type ElasticsearchAnalysis struct {
	API             ElasticsearchMaskableAPI      // Classification of the endpoint
	Index           string                        // Target index name extracted from URL (empty if not applicable)
	BlockedFeatures []ElasticsearchBlockedFeature // Features found in request body that must be blocked
	SourceFields    []string                      // Fields requested via _source filtering (nil = all fields, empty = _source disabled)
	SourceDisabled  bool                          // True when _source: false is set
	RequestedFields []string                      // Fields requested via "fields" parameter
	HighlightFields []string                      // Fields requested via "highlight.fields"
	SortFields      []string                      // Field paths used in "sort"
	HasInnerHits    bool                          // True if query contains inner_hits
	PredicateFields []string                      // Field paths used in query predicates
}
