package gonative

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// distinctPartitionResponse decodes the per-partition response body for a
// DISTINCT query. Unlike aggregate pipelines, the per-partition Documents is
// an array of raw JSON values: one projected object per object-form DISTINCT,
// or one scalar per DISTINCT VALUE. We preserve them as json.RawMessage so the
// de-duplication stage can hash the canonical form without type-specific code.
type distinctPartitionResponse struct {
	Documents []json.RawMessage `json:"Documents"`
}

// parseDistinctPartitionResponse decodes a DISTINCT per-partition body.
func parseDistinctPartitionResponse(raw []byte) (*distinctPartitionResponse, error) {
	var r distinctPartitionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse distinct partition response")
	}
	return &r, nil
}

// canonicalJSON re-marshals the given JSON value so that semantically equal
// values produce byte-equal output (sorted object keys, no extraneous whitespace).
// Returns the canonical bytes so the caller can use them as a map key.
//
// The approach: unmarshal into `any`, which turns objects into map[string]any,
// then re-marshal — Go's encoding/json sorts map keys alphabetically in that
// path, which gives us the canonical ordering for free.
func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, errors.Wrap(err, "gonative: canonicalize")
	}
	return json.Marshal(v)
}

// distinctPipeline implements queryengine.QueryPipeline for unordered DISTINCT
// queries. It streams raw documents in the order partitions deliver them and
// emits each unique value the first time it is seen. The pipeline works
// uniformly for object DISTINCT (SELECT DISTINCT c.field FROM c) and
// DISTINCT VALUE (SELECT DISTINCT VALUE c.field FROM c) because both forms
// land in the Documents array as raw JSON values.
//
// The seen-set grows unbounded — same as the .NET SDK's
// Microsoft.Azure.Cosmos.UnorderedDistinctMap (see
// https://github.com/Azure/azure-cosmos-dotnet-v3/blob/master/Microsoft.Azure.Cosmos/src/Query/Core/Pipeline/Distinct/DistinctMap.UnorderedDistinctMap.cs).
// The OS is the memory ceiling. Queries whose DISTINCT cardinality could
// exhaust memory should scope themselves with a WHERE clause.
type distinctPipeline struct {
	query  string
	fanout rangeFanout
	closed bool

	// Pending items not yet returned to the caller. Populated during
	// ProvideData as each partition's response is de-duplicated against the
	// seen set.
	pending [][]byte

	// seen is the global hash-set of canonicalized documents. Keys are the
	// canonical JSON bytes rendered as a string — Go strings are immutable
	// and comparable, so they make lightweight map keys even for long JSON.
	seen map[string]struct{}
}

// newDistinctPipeline builds a pipeline for a plan whose distinctType is
// "Unordered". Returns an error if the plan's distinctType is "Ordered" —
// that form composes with Stage 3's ORDER BY merge and is not handled here.
func newDistinctPipeline(plan *planDoc, pkRangeIDs []string) (*distinctPipeline, error) {
	if plan.QueryInfo.DistinctType != "Unordered" {
		// "Ordered" is Stage 3's responsibility; anything else is unexpected.
		return nil, ErrUnsupportedPlanFeature
	}
	// DISTINCT plans carry an empty rewrittenQuery — run the original query
	// against each partition. The pipeline needs the original query text to
	// populate QueryRequest.Query; the caller passes it via CreateQueryPipeline.
	return &distinctPipeline{
		fanout: newRangeFanout(pkRangeIDs),
		seen:   make(map[string]struct{}),
	}, nil
}

// Query implements queryengine.QueryPipeline.
func (p *distinctPipeline) Query() string { return p.query }

// IsComplete implements queryengine.QueryPipeline. A DISTINCT pipeline is
// complete once every range has delivered its last page and every deduped
// item has been flushed to the caller.
func (p *distinctPipeline) IsComplete() bool {
	if len(p.pending) > 0 {
		return false
	}
	return p.fanout.allDrained()
}

// Close implements queryengine.QueryPipeline.
func (p *distinctPipeline) Close() { p.closed = true }

// Run drives one pipeline turn. Deduplicated items go out as soon as they are
// known, so a DISTINCT over many pages does not hold the whole result before
// the caller sees its first row:
//  1. Flush any pending deduplicated items.
//  2. Otherwise ask for the next page of every range that has one — the
//     initial fan-out on the first call.
func (p *distinctPipeline) Run() (*queryengine.PipelineResult, error) {
	if len(p.pending) > 0 {
		batch := p.pending
		p.pending = nil
		return &queryengine.PipelineResult{
			Items:       batch,
			IsCompleted: p.IsComplete(),
		}, nil
	}
	if reqs := p.fanout.requests(p.query); len(reqs) > 0 {
		return &queryengine.PipelineResult{Requests: reqs}, nil
	}
	return &queryengine.PipelineResult{IsCompleted: p.IsComplete()}, nil
}

// ProvideData implements queryengine.QueryPipeline. Each incoming partition
// response is parsed and its documents are fed through the global seen set.
// First-sightings become pending items; duplicates are silently dropped.
func (p *distinctPipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		resp, err := parseDistinctPartitionResponse(r.Data)
		if err != nil {
			return err
		}
		for _, doc := range resp.Documents {
			canonical, err := canonicalJSON(doc)
			if err != nil {
				return err
			}
			key := string(canonical)
			if _, ok := p.seen[key]; ok {
				continue
			}
			p.seen[key] = struct{}{}
			// Preserve the partition's original document bytes in the output —
			// callers receive what the gateway returned, not our canonicalized
			// form. Copy defensively: json.RawMessage is aliased to the input
			// buffer, which the SDK may reuse for later pages.
			p.pending = append(p.pending, append([]byte(nil), doc...))
		}
		if err := p.fanout.note(r); err != nil {
			return err
		}
	}
	return nil
}
