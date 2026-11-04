package gonative

import (
	"container/heap"
	"encoding/json"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// orderByQueryFilterPlaceholder is the token the gateway embeds into ORDER BY
// rewritten queries. The client-side engine must substitute it before firing
// per-partition requests — with "true" on the initial page, and with a
// continuation filter comparing the last emitted order-by key on subsequent
// pages. Stage 3 handles the single-page case only; continuation is deferred.
const orderByQueryFilterPlaceholder = "{documentdb-formattableorderbyquery-filter}"

// orderByItem is each element of the per-document orderByItems array. The
// gateway emits one per ORDER BY expression; Stage 3 only supports single-key
// ORDER BY so we read orderByItems[0].
type orderByItem struct {
	Item json.RawMessage `json:"item"`
}

// orderByDoc is one Documents[i] entry in an ORDER BY partition response.
// Payload is the user's projected row — what we eventually emit.
type orderByDoc struct {
	OrderByItems []orderByItem   `json:"orderByItems"`
	Payload      json.RawMessage `json:"payload"`
}

// orderByPartitionResponse decodes an ORDER BY per-partition response body.
type orderByPartitionResponse struct {
	Documents []orderByDoc `json:"Documents"`
}

func parseOrderByPartitionResponse(raw []byte) (*orderByPartitionResponse, error) {
	var r orderByPartitionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, errors.Wrap(err, "gonative: parse order-by partition response")
	}
	return &r, nil
}

// partitionStream holds one partition's ORDER BY documents, in gateway-sorted
// order. The pipeline pops from the head as the heap selects from this
// partition; emptying the slice means the partition is drained.
type partitionStream struct {
	docs []orderByDoc
	head int // index of next doc to emit; len(docs)-head is the remaining count
}

func (s *partitionStream) peek() (*orderByDoc, bool) {
	if s.head >= len(s.docs) {
		return nil, false
	}
	return &s.docs[s.head], true
}

func (s *partitionStream) pop() *orderByDoc {
	d := &s.docs[s.head]
	s.head++
	return d
}

// mergeHeap implements heap.Interface over partition indices. The comparator
// uses compareItems on each partition's current head's orderByItems[0], with
// the sort direction flipped for descending ORDER BY.
type mergeHeap struct {
	partitions []*partitionStream
	idx        []int // heap of partition indices
	descending bool
}

func (h *mergeHeap) Len() int { return len(h.idx) }

func (h *mergeHeap) Less(i, j int) bool {
	// We never call Less when a partition is empty — the pipeline drops empty
	// partitions from the heap before they can be compared.
	a, _ := h.partitions[h.idx[i]].peek()
	b, _ := h.partitions[h.idx[j]].peek()
	cmp, err := compareItems(a.OrderByItems[0].Item, b.OrderByItems[0].Item)
	if err != nil {
		// compareItems rejects unorderable shapes (arrays, objects). For
		// ORDER BY scalars this should never happen; if it does, fall back to
		// stable-by-partition-index order so the heap remains well-defined.
		return h.idx[i] < h.idx[j]
	}
	if h.descending {
		return cmp > 0
	}
	return cmp < 0
}

func (h *mergeHeap) Swap(i, j int) { h.idx[i], h.idx[j] = h.idx[j], h.idx[i] }

func (h *mergeHeap) Push(x any) { h.idx = append(h.idx, x.(int)) }

func (h *mergeHeap) Pop() any {
	n := len(h.idx)
	out := h.idx[n-1]
	h.idx = h.idx[:n-1]
	return out
}

// orderByPipeline implements queryengine.QueryPipeline for plans with a single
// ORDER BY expression. It emits rows in the requested order by k-way-merging
// each partition's gateway-sorted stream.
//
// Stage 3 scope is single-key ORDER BY only. Multi-key ORDER BY requires a
// Cosmos composite index on the container — if the container lacks one the
// gateway rejects the query outright; if it has one the gateway emits a plan
// with multiple entries in orderByExpressions, and the comparator would need
// to walk all of them. Both cases are deferred.
//
// Stage 5 extends the pipeline with OFFSET/LIMIT awareness. Per Cosmos
// semantics, OFFSET/LIMIT always composes with ORDER BY; the gateway rewrites
// the per-partition query to `OFFSET 0 LIMIT (offset+limit)` so every
// partition delivers enough rows to satisfy any interleaving, and this
// pipeline handles the global skip + take after merging.
type orderByPipeline struct {
	rewrittenQuery string
	pkRangeIDs     []string
	descending     bool

	partitionsDone map[string]bool
	partitionByID  map[string]int
	partitions     []*partitionStream

	heap   *mergeHeap
	issued bool
	closed bool

	// offset and limit implement plan.QueryInfo.Offset/Limit. A negative limit
	// (the sentinel returned by newOrderByPipeline when the plan has no limit)
	// means "unlimited". skipped counts rows dropped by offset; emitted counts
	// rows forwarded to the caller — both drive the skip+take logic in drain.
	offset  int
	limit   int
	skipped int
	emitted int

	// pending emits produced by k-way merging when all partitions have data.
	pending [][]byte
}

// newOrderByPipeline constructs a pipeline for a plan with exactly one ORDER BY
// expression. Returns ErrUnsupportedPlanFeature when the plan names multiple
// ORDER BY keys — Stage 3 doesn't yet support that composition.
func newOrderByPipeline(plan *planDoc, pkRangeIDs []string) (*orderByPipeline, error) {
	if len(plan.QueryInfo.OrderBy) != 1 {
		return nil, ErrUnsupportedPlanFeature
	}
	if plan.QueryInfo.OrderBy[0] != "Ascending" && plan.QueryInfo.OrderBy[0] != "Descending" {
		return nil, ErrUnsupportedPlanFeature
	}
	// Substitute the formattable-order-by-query-filter placeholder with "true"
	// for the initial request per partition. Multi-page continuation (which
	// would substitute with a tail-comparison filter) is out of scope here.
	query := strings.ReplaceAll(plan.QueryInfo.RewrittenQuery, orderByQueryFilterPlaceholder, "true")

	// OFFSET/LIMIT always composes with ORDER BY per Cosmos syntax; defaults
	// mean "no skip, unlimited take".
	offset := 0
	if plan.QueryInfo.Offset != nil {
		if *plan.QueryInfo.Offset < 0 {
			return nil, errors.Errorf("gonative: negative OFFSET=%d", *plan.QueryInfo.Offset)
		}
		offset = *plan.QueryInfo.Offset
	}
	limit := -1
	if plan.QueryInfo.Limit != nil {
		if *plan.QueryInfo.Limit < 0 {
			return nil, errors.Errorf("gonative: negative LIMIT=%d", *plan.QueryInfo.Limit)
		}
		limit = *plan.QueryInfo.Limit
	}

	partByID := make(map[string]int, len(pkRangeIDs))
	partitions := make([]*partitionStream, len(pkRangeIDs))
	for i, id := range pkRangeIDs {
		partByID[id] = i
		partitions[i] = &partitionStream{}
	}
	return &orderByPipeline{
		rewrittenQuery: query,
		pkRangeIDs:     append([]string(nil), pkRangeIDs...),
		descending:     plan.QueryInfo.OrderBy[0] == "Descending",
		partitionsDone: make(map[string]bool, len(pkRangeIDs)),
		partitionByID:  partByID,
		partitions:     partitions,
		offset:         offset,
		limit:          limit,
	}, nil
}

func (p *orderByPipeline) Query() string { return p.rewrittenQuery }
func (p *orderByPipeline) Close()        { p.closed = true }

// IsComplete returns true once every partition's gateway response has been
// merged and the last pending row has been handed to the caller, or once a
// LIMIT has been satisfied (in which case unprocessed partition rows are
// intentionally discarded).
func (p *orderByPipeline) IsComplete() bool {
	if !p.issued {
		return false
	}
	if len(p.pending) > 0 {
		return false
	}
	// LIMIT reached — the caller has everything it asked for. Any rows still
	// sitting in partition streams were produced speculatively by the gateway
	// (it fetched OFFSET+LIMIT rows per partition to cover any interleaving)
	// and are safe to drop.
	if p.limit >= 0 && p.emitted >= p.limit {
		return true
	}
	for _, id := range p.pkRangeIDs {
		if !p.partitionsDone[id] {
			return false
		}
	}
	if p.heap != nil && p.heap.Len() > 0 {
		return false
	}
	for _, s := range p.partitions {
		if s.head < len(s.docs) {
			return false
		}
	}
	return true
}

// Run drives one pipeline turn.
//  1. First call: fan out one request per pk-range carrying the (substituted)
//     rewritten query.
//  2. After every partition has provided data, drain the heap into `pending`,
//     then emit.
func (p *orderByPipeline) Run() (*queryengine.PipelineResult, error) {
	if !p.issued {
		reqs := make([]queryengine.QueryRequest, len(p.pkRangeIDs))
		for i, id := range p.pkRangeIDs {
			reqs[i] = queryengine.QueryRequest{
				PartitionKeyRangeID: id,
				Id:                  uint64(i),
				Query:               p.rewrittenQuery,
				IncludeParameters:   true,
			}
		}
		p.issued = true
		return &queryengine.PipelineResult{Requests: reqs}, nil
	}

	if p.heap == nil && p.allPartitionsReady() {
		p.initHeap()
		p.drain()
	}

	if len(p.pending) > 0 {
		batch := p.pending
		p.pending = nil
		return &queryengine.PipelineResult{Items: batch, IsCompleted: p.IsComplete()}, nil
	}
	return &queryengine.PipelineResult{IsCompleted: p.IsComplete()}, nil
}

// ProvideData parses and stores each partition's response. Validates that
// every document has at least one orderByItems entry and a payload.
func (p *orderByPipeline) ProvideData(results []queryengine.QueryResult) error {
	for _, r := range results {
		resp, err := parseOrderByPartitionResponse(r.Data)
		if err != nil {
			return err
		}
		for i, doc := range resp.Documents {
			if len(doc.OrderByItems) == 0 {
				return errors.Errorf("gonative: order-by doc %d missing orderByItems", i)
			}
			if len(doc.Payload) == 0 {
				return errors.Errorf("gonative: order-by doc %d missing payload", i)
			}
		}
		if r.NextContinuation != "" {
			return errors.New("gonative: order-by partition returned a continuation token (multi-page not supported in Stage 3)")
		}
		idx, ok := p.partitionByID[r.PartitionKeyRangeID]
		if !ok {
			return errors.Errorf("gonative: unexpected partition id %q", r.PartitionKeyRangeID)
		}
		p.partitions[idx].docs = append(p.partitions[idx].docs, resp.Documents...)
		p.partitionsDone[r.PartitionKeyRangeID] = true
	}
	return nil
}

func (p *orderByPipeline) allPartitionsReady() bool {
	for _, id := range p.pkRangeIDs {
		if !p.partitionsDone[id] {
			return false
		}
	}
	return true
}

// initHeap seeds the merge heap with every non-empty partition's head.
func (p *orderByPipeline) initHeap() {
	h := &mergeHeap{partitions: p.partitions, descending: p.descending}
	for i, s := range p.partitions {
		if _, ok := s.peek(); ok {
			h.idx = append(h.idx, i)
		}
	}
	heap.Init(h)
	p.heap = h
}

// drain flushes the k-way merge into p.pending. After the single-page
// guarantee the heap holds at most one row per partition; popping the top,
// advancing that partition, and re-pushing (if not drained) emits the
// globally-next row. Runs to completion in one Run() turn.
//
// OFFSET/LIMIT (Stage 5) apply here: the first `offset` rows are discarded,
// and emission stops once `limit` rows have been collected (when limit >= 0).
// Both are tracked on the pipeline across calls so a partial drain would
// resume correctly — though in practice drain runs to completion.
func (p *orderByPipeline) drain() {
	for p.heap.Len() > 0 {
		if p.limit >= 0 && p.emitted >= p.limit {
			// LIMIT reached — stop popping and drop the remaining heap state
			// so IsComplete trips on the next turn.
			p.heap.idx = p.heap.idx[:0]
			return
		}
		top, ok := heap.Pop(p.heap).(int)
		if !ok {
			// Unreachable: the heap is seeded with partition indexes only.
			p.heap.idx = p.heap.idx[:0]
			return
		}
		s := p.partitions[top]
		doc := s.pop()
		if p.skipped < p.offset {
			p.skipped++
		} else {
			// Defensive copy — json.RawMessage aliases the underlying buffer.
			p.pending = append(p.pending, append([]byte(nil), doc.Payload...))
			p.emitted++
		}
		if _, ok := s.peek(); ok {
			heap.Push(p.heap, top)
		}
	}
}
