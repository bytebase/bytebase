package gonative

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// topPartitionResponse decodes a "no rewrite" per-partition response — the
// Documents array carries raw user-shaped rows directly (no orderByItems /
// payload wrapping). Used for standalone TOP, where the gateway runs the
// original query per partition and returns whatever matches.
type topPartitionResponse struct {
	Documents []json.RawMessage `json:"Documents"`
}

func parseTopPartitionResponse(raw []byte) (*topPartitionResponse, error) {
	var r topPartitionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse top partition response")
	}
	return &r, nil
}

// topPipeline implements queryengine.QueryPipeline for standalone TOP queries
// — plans with `top: N` and none of ORDER BY, DISTINCT, aggregates, or GROUP BY.
// Rows from each partition are concatenated and the first N are emitted.
//
// TOP composed with ORDER BY/DISTINCT/aggregates/GROUP BY is out of Stage 4's
// scope; the dispatch rejects those plans with ErrUnsupportedPlanFeature.
//
// A TOP larger than a response page needs several pages from a range. The
// pipeline keeps asking for them until it holds N rows, at which point it
// stops: IsComplete trips on the emitted count and no further page is
// requested, however many rows the ranges still hold.
type topPipeline struct {
	query  string
	n      int
	fanout rangeFanout
	closed bool

	// pending items ready to flush to the caller, already capped at N total.
	pending [][]byte
	// emitted tracks how many rows have been drained via Run(); we stop
	// accepting/emitting once we hit N.
	emitted int
	// accumulated counts pending + emitted. Keeps ProvideData from growing
	// `pending` past N so downstream flushes never over-emit.
	accumulated int
}

// newTopPipeline builds a pipeline for a plan whose only operator is TOP.
// Returns ErrUnsupportedPlanFeature when the plan carries any other operator
// that would require composition (ORDER BY, DISTINCT, aggregate, GROUP BY).
func newTopPipeline(plan *planDoc, pkRangeIDs []string) (*topPipeline, error) {
	if plan.QueryInfo.Top == nil {
		return nil, errors.New("gonative: topPipeline requires a non-nil Top")
	}
	// Composition with other operators is a follow-up. Stage 4 only handles
	// standalone TOP — the dispatcher routes composed plans elsewhere or
	// rejects them.
	if len(plan.QueryInfo.OrderBy) > 0 ||
		plan.QueryInfo.DistinctType == "Unordered" || plan.QueryInfo.DistinctType == "Ordered" ||
		len(plan.QueryInfo.Aggregates) > 0 ||
		len(plan.QueryInfo.GroupByAliasToAggregateType) > 0 ||
		len(plan.QueryInfo.GroupByExpressions) > 0 {
		return nil, ErrUnsupportedPlanFeature
	}
	n := *plan.QueryInfo.Top
	if n < 0 {
		return nil, errors.Errorf("gonative: negative TOP N=%d", n)
	}
	return &topPipeline{
		n:      n,
		fanout: newRangeFanout(pkRangeIDs),
	}, nil
}

// Query implements queryengine.QueryPipeline.
func (p *topPipeline) Query() string { return p.query }

// IsComplete returns true once either (a) we've emitted N rows — which covers
// TOP 0, so the pipeline never fires a request for it — or (b) every range has
// delivered its last page and all pending rows have been flushed.
func (p *topPipeline) IsComplete() bool {
	if p.emitted >= p.n {
		return true
	}
	if len(p.pending) > 0 {
		return false
	}
	return p.fanout.allDrained()
}

// Close implements queryengine.QueryPipeline.
func (p *topPipeline) Close() { p.closed = true }

// Run drives one pipeline turn: flush the rows already in hand, then ask for
// the next page of every range that has one — the initial fan-out on the first
// call.
func (p *topPipeline) Run() (*queryengine.PipelineResult, error) {
	if len(p.pending) > 0 {
		batch := p.pending
		p.pending = nil
		p.emitted += len(batch)
		return &queryengine.PipelineResult{Items: batch, IsCompleted: p.IsComplete()}, nil
	}
	if p.IsComplete() {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
	return &queryengine.PipelineResult{Requests: p.fanout.requests(p.query)}, nil
}

// ProvideData consumes each partition's response and appends rows to `pending`
// up to the overall TOP N limit. Anything beyond N is dropped.
func (p *topPipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		if err := p.fanout.note(r); err != nil {
			return err
		}
		if p.accumulated >= p.n {
			// Already have enough rows — leave this response unread.
			continue
		}
		resp, err := parseTopPartitionResponse(r.Data)
		if err != nil {
			return err
		}
		for _, doc := range resp.Documents {
			if p.accumulated >= p.n {
				break
			}
			// Defensive copy — json.RawMessage aliases the input buffer.
			p.pending = append(p.pending, append([]byte(nil), doc...))
			p.accumulated++
		}
	}
	return nil
}
