package gonative

import (
	"encoding/json"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// groupByItem is one entry in the per-document groupByItems array. For a
// single-key GROUP BY the array has one element; multi-key GROUP BY would
// have more, and needs no special handling here because the engine hashes and
// orders the whole array.
type groupByItem struct {
	Item json.RawMessage `json:"item"`
}

// groupDoc is one Documents[i] entry in a GROUP BY per-partition response.
type groupDoc struct {
	GroupByItems []groupByItem              `json:"groupByItems"`
	Payload      map[string]json.RawMessage `json:"payload"`
}

// groupPartitionResponse decodes a GROUP BY per-partition body.
type groupPartitionResponse struct {
	Documents []groupDoc `json:"Documents"`
}

func parseGroupPartitionResponse(raw []byte) (*groupPartitionResponse, error) {
	var r groupPartitionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse group partition response")
	}
	return &r, nil
}

// groupState accumulates one group's worth of aggregate partials across every
// partition that contributed to that group. Group-key columns are captured
// once (they're identical for every partition that reports this key).
type groupState struct {
	// keyValues holds the raw payload value for each group-key column, in the
	// order those aliases appear in groupByAliases. The first partition to
	// report a given group populates these; subsequent partitions overwrite
	// with the same value (Cosmos guarantees equality).
	keyValues map[string]json.RawMessage
	// items is the group's key as the gateway reported it, kept for ordering
	// the output rows.
	items []groupByItem
	// aggregators are ordered per groupByAliases; entries whose alias is a
	// group-key column are nil.
	aggregators []aggregator
}

// groupPipeline implements queryengine.QueryPipeline for plans with GROUP BY.
// It collects per-partition group contributions into a hash map keyed on the
// canonical JSON of groupByItems, then emits one row per group once every
// partition has delivered.
//
// GROUP BY is non-streaming — all partitions must arrive before any row is
// emitted, and a partition that answers over several pages must be read to its
// last page. Memory is bounded by the cardinality of distinct group keys, not
// by the total row count across partitions.
type groupPipeline struct {
	query  string
	fanout rangeFanout
	closed bool

	// aliases preserves the order of groupByAliases; kinds[i] is the
	// aggregate kind for aliases[i] ("" when the alias is a group-key column).
	aliases []string
	kinds   []string

	// groups is the per-group state, keyed on the canonical JSON of
	// groupByItems. Grows to the cardinality of distinct group keys.
	groups map[string]*groupState
	// order holds those keys in the order the groups were first seen, which
	// is what the output falls back to when the keys cannot be ordered.
	order []string

	// descending says which way to order the output rows, and
	// directionKnown whether the data has told us yet. A GROUP BY the user
	// also ordered
	// reaches the engine as a plan with no order fields at all — the ORDER BY
	// survives only inside rewrittenQuery — so the direction can be read only
	// off the responses: the gateway returns each partition's own groups in
	// the requested order, and the first adjacent pair that differs says
	// which way they run.
	descending     bool
	directionKnown bool

	// pending holds fully finalized rows awaiting the caller's next Run().
	pending [][]byte
	// finalized is true after the first post-drain Run() has built the pending
	// set; subsequent Runs just flush.
	finalized bool
}

// newGroupPipeline builds a pipeline for a GROUP BY plan. It extracts the
// alias order and per-alias aggregate kinds from the plan's groupByAliases
// and groupByAliasToAggregateType fields; any alias missing from the map is
// treated as a group-key column.
//
// Composing GROUP BY with TOP, OFFSET/LIMIT or DISTINCT would mean wrapping
// this pipeline in another operator; CreateQueryPipeline rejects those plans
// before they reach here.
func newGroupPipeline(plan *planDoc, pkRangeIDs []string) (*groupPipeline, error) {
	qi := plan.QueryInfo
	if len(qi.GroupByExpressions) == 0 {
		return nil, errors.New("gonative: groupPipeline requires groupByExpressions")
	}
	if len(qi.GroupByAliases) == 0 {
		return nil, errors.New("gonative: groupPipeline requires groupByAliases")
	}
	aliases := append([]string(nil), qi.GroupByAliases...)
	kinds := make([]string, len(aliases))
	for i, alias := range aliases {
		kind, ok := qi.GroupByAliasToAggregateType[alias]
		if !ok {
			// Alias exists in the output but has no map entry. Treat as a
			// group-key column.
			kinds[i] = ""
			continue
		}
		kinds[i] = kind
		if kind != "" {
			if a := newAggregator(kind); a == nil {
				return nil, errors.Errorf("gonative: unsupported aggregate kind %q for alias %q", kind, alias)
			}
		}
	}

	return &groupPipeline{
		// GROUP BY plans always carry a non-empty rewrittenQuery that wraps the
		// user's projection into {groupByItems, payload}; running the original
		// query instead would return raw rows with no groupByItems field.
		query:   qi.RewrittenQuery,
		fanout:  newRangeFanout(pkRangeIDs),
		aliases: aliases,
		kinds:   kinds,
		groups:  make(map[string]*groupState),
	}, nil
}

// Query implements queryengine.QueryPipeline.
func (p *groupPipeline) Query() string { return p.query }

// IsComplete returns true once every range has delivered its last page and
// every finalized row has been flushed to the caller.
func (p *groupPipeline) IsComplete() bool {
	if len(p.pending) > 0 {
		return false
	}
	return p.fanout.allDrained() && p.finalized
}

// Close implements queryengine.QueryPipeline.
func (p *groupPipeline) Close() { p.closed = true }

// Run drives one pipeline turn:
//  1. Ask for the next page of every range that has one — the initial fan-out
//     across all ranges on the first call.
//  2. Once every range has drained, build the output rows by finalizing each
//     group's aggregators and emitting in alias order.
//  3. Flush pending rows on subsequent calls until drained.
func (p *groupPipeline) Run() (*queryengine.PipelineResult, error) {
	if reqs := p.fanout.requests(p.query); len(reqs) > 0 {
		return &queryengine.PipelineResult{Requests: reqs}, nil
	}

	if !p.finalized && p.fanout.allDrained() {
		if err := p.finalize(); err != nil {
			return nil, err
		}
	}

	if len(p.pending) > 0 {
		batch := p.pending
		p.pending = nil
		return &queryengine.PipelineResult{Items: batch, IsCompleted: p.IsComplete()}, nil
	}
	return &queryengine.PipelineResult{IsCompleted: p.IsComplete()}, nil
}

// ProvideData consumes each partition's response and folds every row into the
// appropriate group's state.
func (p *groupPipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		if err := p.fanout.note(r); err != nil {
			return err
		}
		resp, err := parseGroupPartitionResponse(r.Data)
		if err != nil {
			return err
		}
		p.noteOrdering(resp.Documents)
		for docIdx, doc := range resp.Documents {
			if len(doc.GroupByItems) == 0 {
				return errors.Errorf("gonative: group doc %d missing groupByItems", docIdx)
			}
			key, err := groupKey(doc.GroupByItems)
			if err != nil {
				return err
			}
			state, ok := p.groups[key]
			if !ok {
				state = p.initGroupState(doc)
				p.groups[key] = state
				p.order = append(p.order, key)
			}
			// Merge this partition's aggregate contributions into the group's state.
			for i, alias := range p.aliases {
				if p.kinds[i] == "" {
					// Group-key column: set once; subsequent partitions carry
					// the same value so no work needed.
					continue
				}
				rawItem, ok := doc.Payload[alias]
				if !ok {
					return errors.Errorf("gonative: group doc %d missing payload alias %q", docIdx, alias)
				}
				// The payload value is shaped {"item": <partial> [, "item2": ...]}.
				partial, err := extractAggregatePartial(p.kinds[i], rawItem)
				if err != nil {
					return err
				}
				if err := state.aggregators[i].feedPartial(partial); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// initGroupState creates a fresh state for a newly-seen group key. Captures
// the group-key column values from the first document reporting this key
// (they're identical for every subsequent partition reporting the same key)
// and allocates aggregators for the aggregate columns.
func (p *groupPipeline) initGroupState(doc groupDoc) *groupState {
	s := &groupState{
		keyValues:   make(map[string]json.RawMessage, len(p.aliases)),
		items:       append([]groupByItem(nil), doc.GroupByItems...),
		aggregators: make([]aggregator, len(p.aliases)),
	}
	for i, alias := range p.aliases {
		if p.kinds[i] == "" {
			// Group-key column: store raw value.
			if v, ok := doc.Payload[alias]; ok {
				s.keyValues[alias] = append([]byte(nil), v...)
			}
		} else {
			s.aggregators[i] = newAggregator(p.kinds[i])
		}
	}
	return s
}

// finalize produces one output row per group, in group-key order — see
// sortGroups. Walking p.groups instead would emit them in Go's randomized map
// order, which made the same query answer in a different order every run.
func (p *groupPipeline) finalize() error {
	p.sortGroups()
	for _, key := range p.order {
		row, err := p.buildRow(p.groups[key])
		if err != nil {
			return err
		}
		p.pending = append(p.pending, row)
	}
	p.finalized = true
	return nil
}

// noteOrdering reads the output direction off one partition's response, which
// is the only place a GROUP BY's ORDER BY reaches the engine. Gives up on a
// key it cannot compare: sortGroups checks for those itself.
func (p *groupPipeline) noteOrdering(docs []groupDoc) {
	if p.directionKnown {
		return
	}
	for i := 1; i < len(docs); i++ {
		order, err := compareGroupKeys(docs[i-1].GroupByItems, docs[i].GroupByItems)
		if err != nil {
			return
		}
		if order != 0 {
			p.descending = order > 0
			p.directionKnown = true
			return
		}
	}
}

// sortGroups puts p.order in group-key order, the direction taken from the
// gateway's own per-partition streams. A GROUP BY without an ORDER BY has no
// defined output order, so ordering those rows too costs nothing and is what
// makes every run answer the same way.
//
// Leaves first-seen order in place when any group key is a shape Cosmos
// cannot order (an array or an object), since the comparator would have to
// invent an answer for those.
func (p *groupPipeline) sortGroups() {
	for _, key := range p.order {
		for _, item := range p.groups[key].items {
			if detectJSONType(item.Item) == jsonUnorderable {
				return
			}
		}
	}
	slices.SortStableFunc(p.order, func(a, b string) int {
		order, err := compareGroupKeys(p.groups[a].items, p.groups[b].items)
		if err != nil {
			// Unreachable: every key was checked orderable just above.
			return 0
		}
		if p.descending {
			return -order
		}
		return order
	})
}

// compareGroupKeys orders two group keys under Cosmos item ordering, column
// by column.
func compareGroupKeys(a, b []groupByItem) (int, error) {
	for i := range min(len(a), len(b)) {
		order, err := compareItems(a[i].Item, b[i].Item)
		if err != nil {
			return 0, err
		}
		if order != 0 {
			return order, nil
		}
	}
	return len(a) - len(b), nil
}

// buildRow assembles the final JSON object for a single group, preserving
// the alias order from groupByAliases. JSON object key order in Go's encoder
// depends on sort for map marshaling, so we write the object manually.
//
// An alias whose aggregate has no value — MIN or MAX over a group where the
// column is undefined in every row — is left out of the object, which is how
// the gateway spells it.
func (p *groupPipeline) buildRow(s *groupState) ([]byte, error) {
	var buf []byte
	buf = append(buf, '{')
	for i, alias := range p.aliases {
		value := json.RawMessage("null")
		if p.kinds[i] == "" {
			if v, ok := s.keyValues[alias]; ok {
				value = v
			}
		} else {
			final, err := s.aggregators[i].finalize()
			if err != nil {
				return nil, err
			}
			if final == nil {
				continue
			}
			value = final
		}
		if len(buf) > 1 {
			buf = append(buf, ',')
		}
		aliasJSON, err := json.Marshal(alias)
		if err != nil {
			return nil, err
		}
		buf = append(buf, aliasJSON...)
		buf = append(buf, ':')
		buf = append(buf, value...)
	}
	buf = append(buf, '}')
	return buf, nil
}

// groupKey returns a canonical hash key for a groupByItems array. The key is
// the canonical JSON of the array — equal keys produce equal bytes.
func groupKey(items []groupByItem) (string, error) {
	// Build a slice of the inner "item" values, canonicalize each, then join.
	// Using a combined key of canonicalized element JSON keeps equality
	// deterministic even for complex multi-key GROUP BY.
	parts := make([]json.RawMessage, len(items))
	for i, it := range items {
		c, err := canonicalJSON(it.Item)
		if err != nil {
			return "", errors.Wrapf(err, "gonative: canonicalize group key part %d", i)
		}
		parts[i] = c
	}
	b, err := json.Marshal(parts)
	if err != nil {
		return "", errors.Wrap(err, "gonative: marshal group key")
	}
	return string(b), nil
}

// extractAggregatePartial decodes the payload value for an aggregate alias
// into the shape the corresponding aggregator's feedPartial expects.
//
// The wire format is {"item": <direct value> [, "item2": {"min"|"max":...,
// "count":...}]}. For Count/Sum/Avg the aggregator consumes "item" directly.
// For Min/Max: prefer "item2" (the reliable cross-partition partial carrying
// count); fall back to synthesizing a single-count partial from "item" when
// item2 is absent — see aggregatePartial, which both aggregate pipelines and
// this one route through.
func extractAggregatePartial(kind string, raw json.RawMessage) (json.RawMessage, error) {
	var wrap aggItem
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, errors.Wrap(err, "gonative: extract aggregate partial")
	}
	return aggregatePartial(kind, wrap), nil
}
