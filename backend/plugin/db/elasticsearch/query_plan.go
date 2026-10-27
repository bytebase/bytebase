package elasticsearch

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
)

// Elasticsearch states no plan for a search it has not run. Its Profile API is
// the only tree it publishes — what each Lucene query, collector and
// aggregation did, and how long each took — and it comes back only with a
// search that ran. `_validate/query?explain` runs nothing but answers with the
// rewritten query as a line of text, which is not a plan to draw.
//
// So an explain request here runs the search with profiling on, which makes it
// a read: the API gate requires the statement's own permission as well as
// bb.sql.explain, as it does for a typed EXPLAIN ANALYZE. A request that is not
// a search is refused rather than run, so an explain can never carry out a
// write.

const (
	searchEndpoint = "_search"
	profileField   = "profile"
	tookField      = "took"
)

// profiledSearch returns the request with Elasticsearch's profiling turned on,
// or an error where there is no search to profile.
func profiledSearch(request *parser.Request) (*parser.Request, error) {
	if err := searchable(request); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if len(request.Data) > 0 {
		// A search carries one JSON document; the newline-delimited endpoints
		// that carry several are not searches this profiles.
		if err := json.Unmarshal([]byte(request.Data[0]), &body); err != nil {
			return nil, errors.Wrapf(err, "the search body is not a JSON object to profile")
		}
	}
	body[profileField] = true
	profiled, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode the profiled search")
	}

	// The request is the caller's; the one that runs is a copy of it.
	next := *request
	next.Data = []string{string(profiled)}
	return &next, nil
}

// searchable reports whether the request is one Elasticsearch profiles: a
// search, read with GET or POST. A URI search carrying no body is one too — the
// profile goes in a body of its own.
func searchable(request *parser.Request) error {
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	if method != "GET" && method != "POST" {
		return errors.Errorf("%s is not a search to explain: Elasticsearch profiles a search read with GET or POST", request.Method)
	}
	path, _, _ := strings.Cut(request.URL, "?")
	if !strings.EqualFold(strings.TrimRight(path, "/"), searchEndpoint) &&
		!strings.HasSuffix(strings.ToLower(strings.TrimRight(path, "/")), "/"+searchEndpoint) {
		return errors.Errorf("%s is not a search to explain: Elasticsearch states a plan for _search alone", request.URL)
	}
	if len(request.Data) > 1 {
		return errors.Errorf("%s is not a search to explain: a search carries one body, and these endpoints carry several", request.URL)
	}
	return nil
}

// queryPlanOf returns the plan a profiled search answered with, and removes it
// from the response: the viewer draws it beside the result rather than in it,
// and a profile in the grid is a page of JSON in one cell.
//
// The search's own `took` goes with it, since that is what the whole search
// cost and no node in the profile states it.
func queryPlanOf(response map[string]any) *v1pb.QueryResult_QueryPlan {
	profile, ok := response[profileField]
	if !ok {
		return nil
	}
	delete(response, profileField)
	carried := map[string]any{profileField: profile}
	if took, ok := response[tookField]; ok {
		carried[tookField] = took
	}
	content, err := json.Marshal(carried)
	if err != nil {
		return nil
	}
	return &v1pb.QueryResult_QueryPlan{
		Format: v1pb.QueryOption_JSON,
		// The profile records the search that returned these hits.
		Executed: true,
		Content:  string(content),
	}
}
