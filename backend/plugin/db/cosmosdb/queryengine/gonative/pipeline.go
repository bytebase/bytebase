package gonative

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// aggPipelineBase holds state common to both the VALUE-form and aliased-form
// aggregate pipelines. Aggregators execute in a fixed positional order — the
// pipeline knows how many slots it needs before the first partition response
// arrives, and which slot each partial lands in.
//
// A range that answers over several pages contributes one partial per page;
// the aggregators fold them in as they arrive, so the final row is the same
// whether a range answered in one page or in twenty.
type aggPipelineBase struct {
	rewrittenQuery string
	fanout         rangeFanout
	aggregators    []aggregator // len == number of aggregates to emit
	kinds          []string     // positional: aggregate kind per aggregator
	emitted        bool
	closed         bool

	// skip and take carry the plan's OFFSET and LIMIT. An aggregate query
	// produces exactly one row, so they can only keep it or drop it; the
	// gateway strips both from the rewritten query and leaves them here.
	// take is negative when the plan sets no LIMIT.
	skip int
	take int
}

// requests returns the work to issue next: the initial fan-out across every
// range, then a follow-up for each range that still has pages to deliver.
func (b *aggPipelineBase) requests() []queryengine.QueryRequest {
	return b.fanout.requests(b.rewrittenQuery)
}

// keepsRow reports whether the plan's OFFSET/LIMIT leaves the single
// aggregate row in the result: OFFSET 1 skips past it and LIMIT 0 asks for
// nothing, and the gateway answers both with no rows at all.
func (b *aggPipelineBase) keepsRow() bool {
	return b.skip == 0 && b.take != 0
}

// newAggPipelineBase builds the state both aggregate pipelines share from the
// plan: the per-partition query, one aggregator per kind in positional order,
// and the plan's OFFSET/LIMIT.
func newAggPipelineBase(plan *planDoc, pkRangeIDs []string, kinds []string) (aggPipelineBase, error) {
	aggs := make([]aggregator, len(kinds))
	for i, kind := range kinds {
		a := newAggregator(kind)
		if a == nil {
			return aggPipelineBase{}, errors.Errorf("gonative: unsupported aggregate %q", kind)
		}
		aggs[i] = a
	}
	skip, take, err := planSkipTake(plan.QueryInfo)
	if err != nil {
		return aggPipelineBase{}, err
	}
	return aggPipelineBase{
		rewrittenQuery: plan.QueryInfo.RewrittenQuery,
		fanout:         newRangeFanout(pkRangeIDs),
		aggregators:    aggs,
		kinds:          kinds,
		skip:           skip,
		take:           take,
	}, nil
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
	base, err := newAggPipelineBase(plan, pkRangeIDs, plan.QueryInfo.Aggregates)
	if err != nil {
		return nil, err
	}
	return &valuePipeline{aggPipelineBase: base}, nil
}

// Query returns the gateway-rewritten per-partition query.
func (p *valuePipeline) Query() string { return p.rewrittenQuery }

// IsComplete returns true after the final row has been emitted.
func (p *valuePipeline) IsComplete() bool { return p.emitted }

// Close releases pipeline state. Idempotent and a no-op for pure-Go pipelines.
func (p *valuePipeline) Close() { p.closed = true }

// Run drives one pipeline turn. Must be called in the order:
//  1. Run() — returns per-partition Requests on first call, and a follow-up
//     request for every range whose last response carried a continuation.
//  2. Caller invokes ProvideData for each Request's partition response.
//  3. Run() — once every range has drained, finalizes aggregators and returns
//     the single result row.
func (p *valuePipeline) Run() (*queryengine.PipelineResult, error) {
	if p.emitted {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
	if reqs := p.requests(); len(reqs) > 0 {
		return &queryengine.PipelineResult{Requests: reqs}, nil
	}
	if !p.fanout.allDrained() {
		// Caller has not yet fed every range — no new work to surface.
		return &queryengine.PipelineResult{}, nil
	}
	item, err := p.finalRow()
	if err != nil {
		return nil, err
	}
	p.emitted = true
	if item == nil || !p.keepsRow() {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
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
				if err := feedAggregatePartial(p.aggregators[i], p.kinds[i], item); err != nil {
					return err
				}
			}
		}
		if err := p.fanout.note(r); err != nil {
			return err
		}
	}
	return nil
}

// finalRow produces the final emitted row for a VALUE-form aggregate query.
// For single-aggregate queries it is the finalized scalar itself; for
// multi-aggregate queries (plan.QueryInfo.Aggregates with len > 1) it is a
// JSON array of finalized scalars in positional order.
//
// Returns nil when a single aggregate finalized to undefined — MIN, MAX or AVG
// over an empty set — because the gateway answers those with no row. An
// undefined slot inside a multi-aggregate array has no such spelling and goes
// out as null.
func (p *valuePipeline) finalRow() ([]byte, error) {
	finals := make([]json.RawMessage, len(p.aggregators))
	for i, a := range p.aggregators {
		v, err := a.finalize()
		if err != nil {
			return nil, err
		}
		if v == nil {
			if len(p.aggregators) == 1 {
				return nil, nil
			}
			v = json.RawMessage(`null`)
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
}

func newAliasedPipeline(plan *planDoc, pkRangeIDs []string) (*aliasedPipeline, error) {
	if len(plan.QueryInfo.GroupByAliases) == 0 {
		return nil, errors.New("gonative: aliased plan has no aliases")
	}
	aliases := append([]string(nil), plan.QueryInfo.GroupByAliases...)
	kinds := make([]string, len(aliases))
	for i, alias := range aliases {
		kind, ok := plan.QueryInfo.GroupByAliasToAggregateType[alias]
		if !ok {
			return nil, errors.Errorf("gonative: alias %q missing from groupByAliasToAggregateType", alias)
		}
		kinds[i] = kind
	}
	base, err := newAggPipelineBase(plan, pkRangeIDs, kinds)
	if err != nil {
		return nil, err
	}
	return &aliasedPipeline{aggPipelineBase: base, aliases: aliases}, nil
}

func (p *aliasedPipeline) Query() string    { return p.rewrittenQuery }
func (p *aliasedPipeline) IsComplete() bool { return p.emitted }
func (p *aliasedPipeline) Close()           { p.closed = true }

func (p *aliasedPipeline) Run() (*queryengine.PipelineResult, error) {
	if p.emitted {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
	if reqs := p.requests(); len(reqs) > 0 {
		return &queryengine.PipelineResult{Requests: reqs}, nil
	}
	if !p.fanout.allDrained() {
		return &queryengine.PipelineResult{}, nil
	}
	item, err := p.finalRow()
	if err != nil {
		return nil, err
	}
	p.emitted = true
	if !p.keepsRow() {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
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
				if err := feedAggregatePartial(p.aggregators[slot], p.kinds[slot], partial); err != nil {
					return err
				}
			}
		}
		if err := p.fanout.note(r); err != nil {
			return err
		}
	}
	return nil
}

// finalRow builds `{"<alias1>": <v1>, ...}` preserving the original alias order
// from the plan (matters for deterministic output — map iteration is random).
// An alias whose aggregate finalized to undefined is left out of the object,
// which is how the gateway spells MIN, MAX or AVG over an empty set: the whole
// row comes back as `{}`.
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
		if finals[i] == nil {
			continue
		}
		if len(buf) > 1 {
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
