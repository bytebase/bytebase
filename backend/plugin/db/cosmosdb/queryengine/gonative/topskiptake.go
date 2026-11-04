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
// scope; the dispatch rejects those plans with ErrUnsupportedPlanFeature. The
// gateway itself already caps each partition's response size when TOP is
// requested, so concatenation + cap at N is always bounded.
type topPipeline struct {
	query          string
	n              int
	pkRangeIDs     []string
	partitionsDone map[string]bool
	issued         bool
	closed         bool

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
		n:              n,
		pkRangeIDs:     append([]string(nil), pkRangeIDs...),
		partitionsDone: make(map[string]bool, len(pkRangeIDs)),
	}, nil
}

// Query implements queryengine.QueryPipeline.
func (p *topPipeline) Query() string { return p.query }

// IsComplete returns true once either (a) we've emitted N rows or (b) every
// partition has delivered and all pending rows have been flushed.
func (p *topPipeline) IsComplete() bool {
	if p.emitted >= p.n {
		return true
	}
	if !p.issued {
		return false
	}
	if len(p.pending) > 0 {
		return false
	}
	for _, id := range p.pkRangeIDs {
		if !p.partitionsDone[id] {
			return false
		}
	}
	return true
}

// Close implements queryengine.QueryPipeline.
func (p *topPipeline) Close() { p.closed = true }

// Run drives one pipeline turn.
func (p *topPipeline) Run() (*queryengine.PipelineResult, error) {
	if !p.issued {
		// Edge case: TOP 0 — emit zero rows without firing any partition work.
		if p.n == 0 {
			p.issued = true
			return &queryengine.PipelineResult{IsCompleted: true}, nil
		}
		reqs := make([]queryengine.QueryRequest, len(p.pkRangeIDs))
		for i, id := range p.pkRangeIDs {
			reqs[i] = queryengine.QueryRequest{
				PartitionKeyRangeID: id,
				Id:                  uint64(i),
				Query:               p.query,
				IncludeParameters:   true,
			}
		}
		p.issued = true
		return &queryengine.PipelineResult{Requests: reqs}, nil
	}
	if len(p.pending) > 0 {
		batch := p.pending
		p.pending = nil
		p.emitted += len(batch)
		return &queryengine.PipelineResult{Items: batch, IsCompleted: p.IsComplete()}, nil
	}
	return &queryengine.PipelineResult{IsCompleted: p.IsComplete()}, nil
}

// ProvideData consumes each partition's response and appends rows to `pending`
// up to the overall TOP N limit. Anything beyond N is dropped.
func (p *topPipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		if p.accumulated >= p.n {
			// Already have enough rows; just mark remaining partitions done
			// so IsComplete can trip.
			p.partitionsDone[r.PartitionKeyRangeID] = true
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
		if r.NextContinuation != "" {
			return errors.New("gonative: top partition returned a continuation token (multi-page not supported in Stage 4)")
		}
		p.partitionsDone[r.PartitionKeyRangeID] = true
	}
	return nil
}
