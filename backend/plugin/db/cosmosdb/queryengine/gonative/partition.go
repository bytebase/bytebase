package gonative

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Per-partition response shapes observed in
// queryengine/gonative/testdata after the gateway has rewritten an
// aggregate query. Two shapes:
//
//	VALUE form — plan.QueryInfo.Aggregates non-empty:
//	  {"Documents":[[{"item":18}]]}
//	  Each Documents[i] is an array of one item-object per aggregate in
//	  plan.QueryInfo.Aggregates (positional match). MIN/MAX carry item2 here
//	  too: {"item":1,"item2":{"min":1,"count":5}}.
//
//	Aliased form — plan.QueryInfo.GroupByAliasToAggregateType non-empty:
//	  {"Documents":[{"payload":{"alias":{"item":...,"item2":{...}}}}]}
//	  Each Documents[i] is a single object keyed by "payload"; payload is
//	  keyed by alias name. Item2 is present for MIN/MAX (reliable partial
//	  {min|max, count}); for COUNT/SUM/AVG only Item is populated.

// aggItem is one aggregate's partial inside a partition document. Item is the
// aggregate's own value; Item2 is the cross-partition partial the gateway adds
// for MIN and MAX, which carries the count of rows the partition contributed.
type aggItem struct {
	Item  json.RawMessage `json:"item"`
	Item2 json.RawMessage `json:"item2,omitempty"`
}

// valuePartitionResponse decodes VALUE-form per-partition bodies.
// Documents[i] is an array whose length equals the number of aggregates.
type valuePartitionResponse struct {
	Documents [][]aggItem `json:"Documents"`
}

// parseValuePartitionResponse decodes a per-partition response body for a
// VALUE-form aggregate query.
func parseValuePartitionResponse(raw []byte) (*valuePartitionResponse, error) {
	var r valuePartitionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse value partition response")
	}
	return &r, nil
}

// aliasedDoc is one entry of Documents for an aliased aggregate query.
type aliasedDoc struct {
	Payload map[string]aggItem `json:"payload"`
}

// aliasedPartitionResponse decodes aliased-form per-partition bodies.
type aliasedPartitionResponse struct {
	Documents []aliasedDoc `json:"Documents"`
}

// parseAliasedPartitionResponse decodes a per-partition response body for an
// aliased-form aggregate query (the single-group GROUP BY protocol).
func parseAliasedPartitionResponse(raw []byte) (*aliasedPartitionResponse, error) {
	var r aliasedPartitionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse aliased partition response")
	}
	return &r, nil
}
