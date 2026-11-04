package gonative

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// planDoc is the subset of the Cosmos query-plan document the engine reads.
//
// Observed shapes (see queryengine/gonative/testdata):
//
//	SELECT VALUE agg(x) FROM c            — "VALUE form"
//	  aggregates: ["Count"]
//	  hasSelectValue: true
//	  rewrittenQuery: SELECT VALUE [{"item": agg(...)}] FROM c
//
//	SELECT agg(x) AS alias FROM c         — "aliased form" (single-group GROUP BY)
//	  aggregates: []
//	  groupByAliases: ["alias"]
//	  groupByAliasToAggregateType: {"alias": "Count"|"Sum"|"Min"|"Max"|"Average"}
//	  rewrittenQuery: SELECT {"alias": {"item": ..., "item2": {min|max, count}}} AS payload FROM c
type planDoc struct {
	PartitionedQueryExecutionInfoVersion int           `json:"partitionedQueryExecutionInfoVersion"`
	QueryInfo                            planQueryInfo `json:"queryInfo"`
	QueryRanges                          []queryRange  `json:"queryRanges"`
}

// planQueryInfo holds the instruction set the engine uses to build a pipeline.
//
// Every field here is an instruction, so the pipeline the dispatch picks must
// claim each one the plan sets — see requireHandled. A field added to this
// struct is therefore rejected until a pipeline claims it, which is what keeps
// a new plan shape from losing an operator silently.
type planQueryInfo struct {
	RewrittenQuery              string            `json:"rewrittenQuery"`
	HasSelectValue              bool              `json:"hasSelectValue"`
	Aggregates                  []string          `json:"aggregates"`
	GroupByAliases              []string          `json:"groupByAliases"`
	GroupByAliasToAggregateType map[string]string `json:"groupByAliasToAggregateType"`
	DistinctType                string            `json:"distinctType,omitempty"`
	Top                         *int              `json:"top,omitempty"`
	Offset                      *int              `json:"offset,omitempty"`
	Limit                       *int              `json:"limit,omitempty"`
	OrderBy                     []string          `json:"orderBy,omitempty"`
	OrderByExpressions          []string          `json:"orderByExpressions,omitempty"`
	GroupByExpressions          []string          `json:"groupByExpressions,omitempty"`
	HasNonStreamingOrderBy      bool              `json:"hasNonStreamingOrderBy,omitempty"`
}

// queryRange mirrors the gateway's pk-range description inside the plan.
type queryRange struct {
	Min            string `json:"min"`
	Max            string `json:"max"`
	IsMinInclusive bool   `json:"isMinInclusive"`
	IsMaxInclusive bool   `json:"isMaxInclusive"`
}

// parsePlan decodes the gateway's query-plan JSON payload. It normalizes the
// gateway's "None" distinctType to the empty string, so every caller can read
// a set DistinctType as "this query has a DISTINCT".
func parsePlan(raw []byte) (*planDoc, error) {
	var p planDoc
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errors.Wrap(err, "gonative: parse query plan")
	}
	if p.QueryInfo.DistinctType == "None" {
		p.QueryInfo.DistinctType = ""
	}
	return &p, nil
}

// planSkipTake reads the plan's OFFSET, LIMIT and TOP as one global skip and
// take, which is what every pipeline applies them as: the gateway strips them
// from the rewritten query (or widens LIMIT to OFFSET+LIMIT) precisely so the
// client applies them once, over the merged result.
//
// take is negative when the query names no upper bound. No Cosmos syntax
// combines TOP with OFFSET/LIMIT, but a plan carrying both can only mean the
// tighter of the two.
func planSkipTake(qi planQueryInfo) (skip int, take int, err error) {
	take = -1
	for name, v := range map[string]*int{"OFFSET": qi.Offset, "LIMIT": qi.Limit, "TOP": qi.Top} {
		if v != nil && *v < 0 {
			return 0, 0, errors.Errorf("gonative: negative %s=%d", name, *v)
		}
	}
	if qi.Offset != nil {
		skip = *qi.Offset
	}
	if qi.Limit != nil {
		take = *qi.Limit
	}
	if qi.Top != nil && (take < 0 || *qi.Top < take) {
		take = *qi.Top
	}
	return skip, take, nil
}

// pkRangesResponse decodes the gateway's pk-ranges response just enough to
// extract the ordered partition-key-range IDs the pipeline needs to fan out.
type pkRangesResponse struct {
	PartitionKeyRanges []struct {
		ID string `json:"id"`
	} `json:"PartitionKeyRanges"`
}

// parsePKRangeIDs returns the partition-key-range IDs from the pk-ranges JSON
// body the SDK hands to CreateQueryPipeline.
func parsePKRangeIDs(raw []byte) ([]string, error) {
	var r pkRangesResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse pk ranges")
	}
	ids := make([]string, len(r.PartitionKeyRanges))
	for i, rng := range r.PartitionKeyRanges {
		ids[i] = rng.ID
	}
	return ids, nil
}
