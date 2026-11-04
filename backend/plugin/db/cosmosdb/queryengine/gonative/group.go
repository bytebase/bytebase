package gonative

import (
	"encoding/json"
	"slices"
	"strings"

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

	// sort is how to order the output rows: which group key to compare
	// first and which way, read from the query's ORDER BY — see groupSortKeys.
	sort []groupSortKey

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

	sortKeys, err := groupSortKeys(qi)
	if err != nil {
		return nil, err
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
		sort:    sortKeys,
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

// orderByKeyword is how the gateway spells the clause in a rewritten query.
const orderByKeyword = "ORDER BY"

// orderByKey is one key of an ORDER BY clause.
type orderByKey struct {
	expr       string
	descending bool
}

// groupSortKey names one group key to order the output rows by: its position
// in groupByItems, and which way to compare it.
type groupSortKey struct {
	index      int
	descending bool
}

// groupSortKeys works out how to order a GROUP BY's output rows. A GROUP BY
// the user also ordered reaches the engine as a plan with no order fields at
// all — orderBy and orderByExpressions stay empty and the ORDER BY survives
// only inside the rewritten query — so the clause is read from that text.
//
// The engine orders these rows by comparing their group keys, so every key the
// clause names must be one of the GROUP BY expressions. An order it cannot
// produce that way — by an aggregate, or by a column the query does not group
// by — fails the query rather than answering in an order nobody asked for. The
// gateway refuses those queries itself today, so this is the engine holding the
// line rather than a reachable error.
//
// Group keys the clause leaves out are appended ascending. The clause does not
// constrain them, and ordering them too is what makes two runs of one query
// answer alike.
func groupSortKeys(qi planQueryInfo) ([]groupSortKey, error) {
	keys, found := trailingOrderBy(qi.RewrittenQuery)
	if found && len(keys) == 0 {
		return nil, errors.Wrap(ErrUnsupportedPlanFeature, "ORDER BY is not a plain list of keys")
	}
	sortKeys := make([]groupSortKey, 0, len(qi.GroupByExpressions))
	named := make([]bool, len(qi.GroupByExpressions))
	for _, key := range keys {
		index := slices.Index(qi.GroupByExpressions, key.expr)
		if index < 0 {
			return nil, errors.Wrapf(ErrUnsupportedPlanFeature,
				"ORDER BY key %q is not one of the GROUP BY expressions", key.expr)
		}
		if named[index] {
			continue
		}
		named[index] = true
		sortKeys = append(sortKeys, groupSortKey{index: index, descending: key.descending})
	}
	for index, taken := range named {
		if !taken {
			sortKeys = append(sortKeys, groupSortKey{index: index})
		}
	}
	return sortKeys, nil
}

// trailingOrderBy splits the ORDER BY clause that ends query into its keys.
// found is false when the query has no trailing ORDER BY; keys is nil with
// found true when the clause is not the plain `<expression> [ASC|DESC]` list
// the gateway writes there, which the caller must treat as an order it cannot
// honour rather than as no order at all.
func trailingOrderBy(query string) (keys []orderByKey, found bool) {
	at := strings.LastIndex(strings.ToUpper(query), orderByKeyword)
	if at < 0 {
		return nil, false
	}
	tail := query[at+len(orderByKeyword):]
	for _, token := range strings.Fields(strings.ToUpper(tail)) {
		switch token {
		case "FROM", "WHERE", "GROUP", "JOIN":
			// A clause follows the match, so the match sits inside the query
			// body — in a string literal holding the words, say — and the
			// query has no trailing ORDER BY.
			return nil, false
		}
	}
	for _, item := range strings.Split(tail, ",") {
		fields := strings.Fields(item)
		var key orderByKey
		switch len(fields) {
		case 1:
			key.expr = fields[0]
		case 2:
			key.expr = fields[0]
			switch strings.ToUpper(fields[1]) {
			case "ASC":
			case "DESC":
				key.descending = true
			default:
				return nil, true
			}
		default:
			return nil, true
		}
		if strings.ContainsAny(key.expr, `'"`) {
			return nil, true
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, true
	}
	return keys, true
}

// sortGroups puts p.order in group-key order, in the direction the query
// asked for.
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
		order, err := compareGroupKeys(p.groups[a].items, p.groups[b].items, p.sort)
		if err != nil {
			// Unreachable: every key was checked orderable just above.
			return 0
		}
		return order
	})
}

// compareGroupKeys orders two group keys under Cosmos item ordering, comparing
// the columns sortKeys names in the order and direction it gives.
func compareGroupKeys(a, b []groupByItem, sortKeys []groupSortKey) (int, error) {
	for _, key := range sortKeys {
		if key.index >= len(a) || key.index >= len(b) {
			continue
		}
		order, err := compareItems(a[key.index].Item, b[key.index].Item)
		if err != nil {
			return 0, err
		}
		if order != 0 {
			if key.descending {
				return -order, nil
			}
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
