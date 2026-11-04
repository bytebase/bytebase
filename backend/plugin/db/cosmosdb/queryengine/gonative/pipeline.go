package gonative

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// aggPipelineBase holds state common to both the VALUE-form and aliased-form
// aggregate pipelines. Aggregators execute in a fixed positional order — the
// pipeline knows how many slots it needs before the first partition response
// arrives, and which slot each partial lands in.
type aggPipelineBase struct {
	rewrittenQuery string
	pkRangeIDs     []string
	aggregators    []aggregator // len == number of aggregates to emit
	partitionsDone map[string]bool
	issued         bool
	emitted        bool
	closed         bool
}

// issueRequests constructs the initial fan-out: one QueryRequest per
// partition-key-range, all carrying the rewritten query the gateway produced.
func (b *aggPipelineBase) issueRequests() []queryengine.QueryRequest {
	reqs := make([]queryengine.QueryRequest, len(b.pkRangeIDs))
	for i, id := range b.pkRangeIDs {
		reqs[i] = queryengine.QueryRequest{
			PartitionKeyRangeID: id,
			Id:                  uint64(i),
			Query:               b.rewrittenQuery,
			IncludeParameters:   true,
		}
	}
	b.issued = true
	return reqs
}

// allPartitionsDone reports whether every pk-range has finished streaming.
func (b *aggPipelineBase) allPartitionsDone() bool {
	if len(b.pkRangeIDs) == 0 {
		return true
	}
	for _, id := range b.pkRangeIDs {
		if !b.partitionsDone[id] {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------
// VALUE-form pipeline — plan.QueryInfo.Aggregates non-empty, hasSelectValue true.
// Emits one final scalar (single aggregate) or one JSON array (multi-aggregate).

type valuePipeline struct {
	aggPipelineBase
}

// newValuePipeline builds a VALUE-form pipeline for the given plan + pk-ranges.
// Returns an error if the plan references an aggregate kind the engine cannot
// handle, so the caller can surface ErrUnsupportedPlanFeature.
func newValuePipeline(plan *planDoc, pkRangeIDs []string) (*valuePipeline, error) {
	aggs := make([]aggregator, len(plan.QueryInfo.Aggregates))
	for i, kind := range plan.QueryInfo.Aggregates {
		a := newAggregator(kind)
		if a == nil {
			return nil, errors.Errorf("gonative: unsupported aggregate %q", kind)
		}
		aggs[i] = a
	}
	return &valuePipeline{
		aggPipelineBase: aggPipelineBase{
			rewrittenQuery: plan.QueryInfo.RewrittenQuery,
			pkRangeIDs:     pkRangeIDs,
			aggregators:    aggs,
			partitionsDone: make(map[string]bool, len(pkRangeIDs)),
		},
	}, nil
}

// Query returns the gateway-rewritten per-partition query.
func (p *valuePipeline) Query() string { return p.rewrittenQuery }

// IsComplete returns true after the final row has been emitted.
func (p *valuePipeline) IsComplete() bool { return p.emitted }

// Close releases pipeline state. Idempotent and a no-op for pure-Go pipelines.
func (p *valuePipeline) Close() { p.closed = true }

// Run drives one pipeline turn. Must be called in the order:
//  1. Run() — returns per-partition Requests on first call.
//  2. Caller invokes ProvideData for each Request's partition response.
//  3. Run() — finalizes aggregators and returns the single result row.
func (p *valuePipeline) Run() (*queryengine.PipelineResult, error) {
	if !p.issued {
		return &queryengine.PipelineResult{Requests: p.issueRequests()}, nil
	}
	if p.emitted {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
	if !p.allPartitionsDone() {
		// Caller has not yet fed every partition — no new work to surface.
		return &queryengine.PipelineResult{}, nil
	}
	item, err := p.finalRow()
	if err != nil {
		return nil, err
	}
	p.emitted = true
	return &queryengine.PipelineResult{Items: [][]byte{item}, IsCompleted: true}, nil
}

// ProvideData feeds per-partition response bodies into the pipeline.
func (p *valuePipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		resp, err := parseValuePartitionResponse(r.Data)
		if err != nil {
			return err
		}
		for _, doc := range resp.Documents {
			if len(doc) != len(p.aggregators) {
				return errors.Errorf("gonative: value partition document has %d items, expected %d", len(doc), len(p.aggregators))
			}
			for i, item := range doc {
				if err := p.aggregators[i].feedPartial(item.Item); err != nil {
					return err
				}
			}
		}
		// Stage 1 assumes one page per partition for aggregates; NextContinuation
		// should be empty. Flag explicitly if the assumption breaks.
		if r.NextContinuation != "" {
			return errors.New("gonative: aggregate partition returned a continuation token (multi-page not supported in Stage 1)")
		}
		p.partitionsDone[r.PartitionKeyRangeID] = true
	}
	return nil
}

// finalRow produces the final emitted row for a VALUE-form aggregate query.
// For single-aggregate queries it is the finalized scalar itself; for
// multi-aggregate queries (plan.QueryInfo.Aggregates with len > 1) it is a
// JSON array of finalized scalars in positional order.
func (p *valuePipeline) finalRow() ([]byte, error) {
	finals := make([]json.RawMessage, len(p.aggregators))
	for i, a := range p.aggregators {
		v, err := a.finalize()
		if err != nil {
			return nil, err
		}
		finals[i] = v
	}
	if len(finals) == 1 {
		return finals[0], nil
	}
	return json.Marshal(finals)
}

// -----------------------------------------------------------------------------
// Aliased-form pipeline — plan has groupByAliases + groupByAliasToAggregateType.
// Emits one object row: {"<alias1>": <v1>, "<alias2>": <v2>, ...}.

type aliasedPipeline struct {
	aggPipelineBase
	aliases []string // preserved order from plan.QueryInfo.GroupByAliases
	kinds   []string // positional: aggregate kind per alias
}

func newAliasedPipeline(plan *planDoc, pkRangeIDs []string) (*aliasedPipeline, error) {
	if len(plan.QueryInfo.GroupByAliases) == 0 {
		return nil, errors.New("gonative: aliased plan has no aliases")
	}
	aliases := append([]string(nil), plan.QueryInfo.GroupByAliases...)
	kinds := make([]string, len(aliases))
	aggs := make([]aggregator, len(aliases))
	for i, alias := range aliases {
		kind, ok := plan.QueryInfo.GroupByAliasToAggregateType[alias]
		if !ok {
			return nil, errors.Errorf("gonative: alias %q missing from groupByAliasToAggregateType", alias)
		}
		a := newAggregator(kind)
		if a == nil {
			return nil, errors.Errorf("gonative: unsupported aggregate %q for alias %q", kind, alias)
		}
		kinds[i] = kind
		aggs[i] = a
	}
	return &aliasedPipeline{
		aggPipelineBase: aggPipelineBase{
			rewrittenQuery: plan.QueryInfo.RewrittenQuery,
			pkRangeIDs:     pkRangeIDs,
			aggregators:    aggs,
			partitionsDone: make(map[string]bool, len(pkRangeIDs)),
		},
		aliases: aliases,
		kinds:   kinds,
	}, nil
}

func (p *aliasedPipeline) Query() string    { return p.rewrittenQuery }
func (p *aliasedPipeline) IsComplete() bool { return p.emitted }
func (p *aliasedPipeline) Close()           { p.closed = true }

func (p *aliasedPipeline) Run() (*queryengine.PipelineResult, error) {
	if !p.issued {
		return &queryengine.PipelineResult{Requests: p.issueRequests()}, nil
	}
	if p.emitted {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
	if !p.allPartitionsDone() {
		return &queryengine.PipelineResult{}, nil
	}
	item, err := p.finalRow()
	if err != nil {
		return nil, err
	}
	p.emitted = true
	return &queryengine.PipelineResult{Items: [][]byte{item}, IsCompleted: true}, nil
}

func (p *aliasedPipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		resp, err := parseAliasedPartitionResponse(r.Data)
		if err != nil {
			return err
		}
		for _, doc := range resp.Documents {
			for slot, alias := range p.aliases {
				partial, ok := doc.Payload[alias]
				if !ok {
					return errors.Errorf("gonative: aliased partition missing alias %q in payload", alias)
				}
				if err := p.feedAliasedPartial(slot, p.kinds[slot], partial); err != nil {
					return err
				}
			}
		}
		if r.NextContinuation != "" {
			return errors.New("gonative: aggregate partition returned a continuation token (multi-page not supported in Stage 1)")
		}
		p.partitionsDone[r.PartitionKeyRangeID] = true
	}
	return nil
}

// feedAliasedPartial routes an aliasedItem to the right aggregator. For
// Min/Max it prefers Item2 (the reliable `{min|max, count}` partial) when
// present, falling back to wrapping Item when the gateway short-circuited to
// a single-partition shape (Item2 absent).
func (p *aliasedPipeline) feedAliasedPartial(slot int, kind string, partial aliasedItem) error {
	switch kind {
	case "Min":
		if len(partial.Item2) > 0 {
			return p.aggregators[slot].feedPartial(partial.Item2)
		}
		synth := fmt.Appendf(nil, `{"min":%s,"count":1}`, partial.Item)
		return p.aggregators[slot].feedPartial(synth)
	case "Max":
		if len(partial.Item2) > 0 {
			return p.aggregators[slot].feedPartial(partial.Item2)
		}
		synth := fmt.Appendf(nil, `{"max":%s,"count":1}`, partial.Item)
		return p.aggregators[slot].feedPartial(synth)
	default:
		// Count, Sum, Avg — the Item itself carries the partial in the right shape.
		return p.aggregators[slot].feedPartial(partial.Item)
	}
}

// finalRow builds `{"<alias1>": <v1>, ...}` preserving the original alias order
// from the plan (matters for deterministic output — map iteration is random).
func (p *aliasedPipeline) finalRow() ([]byte, error) {
	finals := make([]json.RawMessage, len(p.aggregators))
	for i, a := range p.aggregators {
		v, err := a.finalize()
		if err != nil {
			return nil, err
		}
		finals[i] = v
	}
	// Build the JSON manually to preserve alias order; json.Marshal on a map
	// would alphabetize.
	var buf []byte
	buf = append(buf, '{')
	for i, alias := range p.aliases {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyRaw, err := json.Marshal(alias)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyRaw...)
		buf = append(buf, ':')
		buf = append(buf, finals[i]...)
	}
	buf = append(buf, '}')
	return buf, nil
}
