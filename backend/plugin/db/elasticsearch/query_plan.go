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

	// Every value is kept as the caller wrote it: decoding a number here would
	// round a long past 2^53, and the search that ran would not be theirs.
	body := map[string]json.RawMessage{}
	if len(request.Data) > 0 {
		// A search carries one JSON document; the newline-delimited endpoints
		// that carry several are not searches this profiles.
		if err := json.Unmarshal([]byte(request.Data[0]), &body); err != nil {
			return nil, errors.Wrapf(err, "the search body is not a JSON object to profile")
		}
	}
	body[profileField] = json.RawMessage("true")
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

// profileOnly returns the plan a profiled search answered with: its profile
// and the time the search took, and nothing else the response carried.
//
// The hits are dropped rather than returned. Masking is off for an explain
// request — every other engine's explain states a plan and no data, so there
// has never been anything there to mask — and a search profiled through this
// path would otherwise hand back documents that an ordinary query would have
// masked. The plan is what an explain asks for; the documents are not.
//
// The search's own `took` goes with the profile, since that is what the whole
// search cost and no node within it states that.
func profileOnly(response map[string]any) (string, error) {
	profile, ok := response[profileField]
	if !ok {
		return "", errors.Errorf("the search answered with no profile to explain")
	}
	plan := map[string]any{profileField: profile}
	if took, ok := response[tookField]; ok {
		plan[tookField] = took
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return "", errors.Wrapf(err, "failed to encode the profile")
	}
	return string(encoded), nil
}

// explainQueryPlan marks a result as the plan of a search that ran, which is
// the only kind Elasticsearch states.
func explainQueryPlan() *v1pb.QueryResult_QueryPlan {
	return &v1pb.QueryResult_QueryPlan{
		Format:   v1pb.QueryOption_JSON,
		Executed: true,
	}
}
