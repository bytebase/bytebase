package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskingAdapterSeam(t *testing.T) {
	api, idx := classifyMaskableAPI("POST", "/users/_search")
	require.Equal(t, APIMaskSearch, api)
	require.Equal(t, "users", idx)

	body := analyzeRequestBody(`{
		"_source": ["name", "email"],
		"fields": ["id"],
		"highlight": {"fields": {"bio": {}}},
		"sort": [{"age": "desc"}],
		"aggs": {"by_status": {"terms": {"field": "status"}}},
		"query": {"term": {"email": "alice@example.com"}}
	}`)
	require.Equal(t, []BlockedFeature{BlockedFeatureAggs}, body.BlockedFeatures)
	require.False(t, body.SourceDisabled)
	require.Equal(t, []string{"name", "email"}, body.SourceFields)
	require.Equal(t, []string{"id"}, body.RequestedFields)
	require.Equal(t, []string{"bio"}, body.HighlightFields)
	require.Equal(t, []string{"age"}, body.SortFields)
	require.ElementsMatch(t, []string{"email"}, body.PredicateFields)

	req := AnalyzeRequest("POST", "/users/_search", `{"query":{"match":{"email":"alice@example.com"}},"aggs":{"c":{"value_count":{"field":"id"}}}}`)
	require.Equal(t, APIMaskSearch, req.API)
	require.Equal(t, "users", req.Index)
	require.Equal(t, []BlockedFeature{BlockedFeatureAggs}, req.BlockedFeatures)
	require.ElementsMatch(t, []string{"email"}, req.PredicateFields)
}
