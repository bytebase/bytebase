package gonative

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// topPartitionResponse decodes an unwrapped per-partition response — the
// Documents array carries raw user-shaped rows directly (no orderByItems /
// payload wrapping). Used for TOP and for OFFSET/LIMIT without an ORDER BY,
// where each partition returns whatever matches and the engine cuts the page.
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

// skipTakePipeline implements queryengine.QueryPipeline for the plans whose
// only operators cut a window out of an unordered result: `TOP n`, and
// `OFFSET n LIMIT m` with no ORDER BY. Rows from each partition are
// concatenated, the first `skip` are dropped and the next `take` emitted.
//
// Which rows those are is unspecified without an ORDER BY, which is what
// makes the window safe to cut here: the gateway widens the per-partition
// query to `OFFSET 0 LIMIT (offset+limit)` (or leaves TOP n on it), so every
// range offers at least as many rows as the window needs and no row the
// window should hold is missing from the union.
//
// A window larger than a response page needs several pages from a range. The
// pipeline keeps asking for them until it holds take rows, at which point it
// stops: IsComplete trips on the emitted count and no further page is
// requested, however many rows the ranges still hold.
type skipTakePipeline struct {
	query  string
	skip   int
	take   int // negative means unlimited
	fanout rangeFanout
	closed bool

	// pending items ready to flush to the caller, already capped at take.
	pending [][]byte
	// skipped counts the rows dropped by skip so far.
	skipped int
	// emitted tracks how many rows have been drained via Run(); we stop
	// accepting/emitting once we hit take.
	emitted int
	// accumulated counts pending + emitted. Keeps ProvideData from growing
	// `pending` past take so downstream flushes never over-emit.
	accumulated int
}

// newSkipTakePipeline builds a pipeline for a plan whose only operators are
// TOP and/or OFFSET/LIMIT. Composition with another operator is the
// dispatch's call, not this constructor's: CreateQueryPipeline rejects a plan
// carrying a field this pipeline cannot honour before it gets here.
func newSkipTakePipeline(plan *planDoc, pkRangeIDs []string, query string) (*skipTakePipeline, error) {
	if plan.QueryInfo.Top == nil && plan.QueryInfo.Offset == nil && plan.QueryInfo.Limit == nil {
		return nil, errors.New("gonative: skipTakePipeline requires TOP, OFFSET or LIMIT")
	}
	skip, take, err := planSkipTake(plan.QueryInfo)
	if err != nil {
		return nil, err
	}
	// A bare TOP plan carries no rewrittenQuery, so each range answers the
	// user's own query, TOP and all. An OFFSET/LIMIT plan carries one widened
	// to `OFFSET 0 LIMIT (offset+limit)`, and running that is what lets this
	// pipeline cut the window globally rather than per range.
	if plan.QueryInfo.RewrittenQuery != "" {
		query = plan.QueryInfo.RewrittenQuery
	}
	return &skipTakePipeline{
		query:  query,
		skip:   skip,
		take:   take,
		fanout: newRangeFanout(pkRangeIDs),
	}, nil
}

// Query implements queryengine.QueryPipeline.
func (p *skipTakePipeline) Query() string { return p.query }

// IsComplete returns true once either (a) we've emitted take rows — which
// covers TOP 0 and LIMIT 0, so the pipeline never fires a request for them —
// or (b) every range has delivered its last page and all pending rows have
// been flushed.
func (p *skipTakePipeline) IsComplete() bool {
	if p.take >= 0 && p.emitted >= p.take {
		return true
	}
	if len(p.pending) > 0 {
		return false
	}
	return p.fanout.allDrained()
}

// Close implements queryengine.QueryPipeline.
func (p *skipTakePipeline) Close() { p.closed = true }

// Run drives one pipeline turn: flush the rows already in hand, then ask for
// the next page of every range that has one — the initial fan-out on the first
// call.
func (p *skipTakePipeline) Run() (*queryengine.PipelineResult, error) {
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

// ProvideData consumes each partition's response, drops the first `skip` rows
// and appends the rest to `pending` up to `take` rows in all.
func (p *skipTakePipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		if err := p.fanout.note(r); err != nil {
			return err
		}
		if p.full() {
			// Already have enough rows — leave this response unread.
			continue
		}
		resp, err := parseTopPartitionResponse(r.Data)
		if err != nil {
			return err
		}
		for _, doc := range resp.Documents {
			if p.full() {
				break
			}
			if p.skipped < p.skip {
				p.skipped++
				continue
			}
			// Defensive copy — json.RawMessage aliases the input buffer.
			p.pending = append(p.pending, append([]byte(nil), doc...))
			p.accumulated++
		}
	}
	return nil
}

// full reports whether the window already holds every row it can take.
func (p *skipTakePipeline) full() bool {
	return p.take >= 0 && p.accumulated >= p.take
}
