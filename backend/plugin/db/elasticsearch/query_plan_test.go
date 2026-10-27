package elasticsearch

import (
	"encoding/json"
	"slices"
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

// TestProfileOnly pins what an explain answers with: the profile and the time
// the search took, and nothing else it returned. The hits are dropped here
// rather than masked, because masking is off for an explain request.
func TestProfileOnly(t *testing.T) {
	response := map[string]any{
		"took": float64(102),
		"hits": map[string]any{
			"total": map[string]any{"value": float64(3)},
			"hits":  []any{map[string]any{"_source": map[string]any{"ssn": "111-22-3333"}}},
		},
		"aggregations": map[string]any{"by_city": map[string]any{"buckets": []any{}}},
		"profile":      map[string]any{"shards": []any{map[string]any{"id": "[n][customers][0]"}}},
	}
	encoded, err := profileOnly(response)
	require.NoError(t, err)

	var plan map[string]any
	require.NoError(t, json.Unmarshal([]byte(encoded), &plan))
	require.Equal(t, []string{"profile", "took"}, sortedKeys(plan))
	require.Equal(t, float64(102), plan["took"])
	// Nothing a document could hide in survives.
	require.NotContains(t, encoded, "ssn")
	require.NotContains(t, encoded, "111-22-3333")
	require.NotContains(t, encoded, "aggregations")
}

// TestExplainQueryPlan pins what the result says about its plan: JSON, and
// that it ran, which is what the viewer badges measured.
func TestExplainQueryPlan(t *testing.T) {
	plan := explainQueryPlan()
	require.Equal(t, v1pb.QueryOption_JSON, plan.Format)
	require.True(t, plan.Executed)
	// The plan is the result's rows here, not something carried beside them:
	// an explain returns no data to carry it beside.
	require.Empty(t, plan.Content)
}

func TestProfileOnlyWithoutAProfile(t *testing.T) {
	// A response with no profile is not a plan, and saying so beats answering
	// with the search's own hits.
	_, err := profileOnly(map[string]any{"took": float64(1), "hits": map[string]any{}})
	require.Error(t, err)
}

// TestProfiledSearchKeepsNumbersExact locks the caller's own body. Decoding a
// number on the way through would round a long past 2^53, and the search that
// ran would not be the one they wrote.
func TestProfiledSearchKeepsNumbersExact(t *testing.T) {
	const big = "9007199254740993"
	request := parser.Request{
		Method: "POST",
		URL:    "/events/_search",
		Data:   []string{`{"query":{"term":{"id":` + big + `}},"size":2}`},
	}
	profiled, err := profiledSearch(&request)
	require.NoError(t, err)
	require.Contains(t, profiled.Data[0], big)
	require.Contains(t, profiled.Data[0], `"profile":true`)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
