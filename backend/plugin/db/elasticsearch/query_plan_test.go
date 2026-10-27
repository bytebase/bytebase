package elasticsearch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
)

// TestProfiledSearch pins what an explain request sends: the caller's search
// with profiling on, and nothing else at all.
func TestProfiledSearch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request parser.Request
		want    map[string]any
	}{
		{
			name:    "a search keeps its body",
			request: parser.Request{Method: "POST", URL: "/customers/_search", Data: []string{`{"query":{"term":{"city":"berlin"}}}`}},
			want:    map[string]any{"query": map[string]any{"term": map[string]any{"city": "berlin"}}, "profile": true},
		},
		{
			name:    "a search with no body gets one",
			request: parser.Request{Method: "GET", URL: "/customers/_search"},
			want:    map[string]any{"profile": true},
		},
		{
			// A URI search asks in the query string; the profile goes in a body.
			name:    "a URI search is a search",
			request: parser.Request{Method: "GET", URL: "/customers/_search?q=city:berlin&size=1"},
			want:    map[string]any{"profile": true},
		},
		{
			name:    "a search of every index",
			request: parser.Request{Method: "POST", URL: "_search", Data: []string{`{"size":0}`}},
			want:    map[string]any{"size": float64(0), "profile": true},
		},
		{
			// Already asked for: the request says so and means it.
			name:    "a search that asked for the profile already",
			request: parser.Request{Method: "POST", URL: "/customers/_search", Data: []string{`{"profile":false}`}},
			want:    map[string]any{"profile": true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profiled, err := profiledSearch(&tc.request)
			require.NoError(t, err)
			require.Len(t, profiled.Data, 1)

			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(profiled.Data[0]), &body))
			require.Equal(t, tc.want, body)
			// The method and the URL are the caller's.
			require.Equal(t, tc.request.Method, profiled.Method)
			require.Equal(t, tc.request.URL, profiled.URL)
		})
	}
}

// TestProfiledSearchRefusesWhatIsNotASearch locks the line between explaining
// and doing. Profiling runs the request, so anything that is not a search —
// which is everything that writes — is refused rather than sent.
func TestProfiledSearchRefusesWhatIsNotASearch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request parser.Request
	}{
		{"an index of a document", parser.Request{Method: "PUT", URL: "/customers/_doc/1", Data: []string{`{"city":"berlin"}`}}},
		{"a delete", parser.Request{Method: "DELETE", URL: "/customers"}},
		{"a delete by query, which reads like a search", parser.Request{Method: "POST", URL: "/customers/_delete_by_query", Data: []string{`{"query":{"match_all":{}}}`}}},
		{"an update by query", parser.Request{Method: "POST", URL: "/customers/_update_by_query"}},
		{"a bulk write", parser.Request{Method: "POST", URL: "/_bulk", Data: []string{`{"index":{}}`, `{"city":"berlin"}`}}},
		{"a multi search, which carries several bodies", parser.Request{Method: "POST", URL: "/customers/_msearch", Data: []string{`{}`, `{"query":{"match_all":{}}}`}}},
		{"a search whose body is not an object", parser.Request{Method: "POST", URL: "/customers/_search", Data: []string{`"not a body"`}}},
		{"a path that merely ends in the word", parser.Request{Method: "POST", URL: "/customers/not_search"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := profiledSearch(&tc.request)
			require.Error(t, err)
		})
	}
}

// TestQueryPlanOf pins what the result carries: the profile and the search's
// own time, taken out of the rows so the grid keeps the answer.
func TestQueryPlanOf(t *testing.T) {
	response := map[string]any{
		"took":    float64(102),
		"hits":    map[string]any{"total": map[string]any{"value": float64(3)}},
		"profile": map[string]any{"shards": []any{map[string]any{"id": "[n][customers][0]"}}},
	}
	plan := queryPlanOf(response)
	require.NotNil(t, plan)
	require.Equal(t, v1pb.QueryOption_JSON, plan.Format)
	// The profile records the search that returned these hits.
	require.True(t, plan.Executed)

	var carried map[string]any
	require.NoError(t, json.Unmarshal([]byte(plan.Content), &carried))
	require.Equal(t, float64(102), carried["took"])
	require.Contains(t, carried, "profile")

	// The profile leaves the response, so it is not also a cell of the grid.
	require.NotContains(t, response, "profile")
	require.Contains(t, response, "hits")
	require.Contains(t, response, "took")
}

func TestQueryPlanOfWithoutAProfile(t *testing.T) {
	// A response that carries no profile leaves the result as it was.
	response := map[string]any{"took": float64(1), "hits": map[string]any{}}
	require.Nil(t, queryPlanOf(response))
	require.Len(t, response, 2)
}
