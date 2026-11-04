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
// per-partition requests. The engine substitutes "true" and leaves it there:
// a range's later pages are fetched with the gateway's own continuation token,
// which resumes that range's enumeration, so the filter never has to encode
// where the range left off. The placeholder exists for engines that resume a
// range from a client-side continuation token instead.
const orderByQueryFilterPlaceholder = "{documentdb-formattableorderbyquery-filter}"

// orderByItem is each element of the per-document orderByItems array. The
// gateway emits one per ORDER BY expression; the engine supports single-key
// ORDER BY only, so it reads orderByItems[0].
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

// partitionStream buffers one partition's ORDER BY documents, in
// gateway-sorted order. The pipeline pops from the head as the heap selects
// from this partition; an empty buffer means the partition has delivered
// everything it has so far, which is the end of its stream only once the
// fanout reports the range drained.
type partitionStream struct {
	id   string // the partition key range that feeds this stream
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
	// The heap orders with compareItemsTotal so that it always has a minimum,
	// and because that order follows the service's class ranking the minimum's
	// class is the right one. Which of two values inside the array or object
	// class comes first the service alone knows, so merge asks that question
	// of the row it is about to emit — see ambiguousFirst — rather than of
	// every pair the heap happens to sift.
	cmp, err := compareItemsTotal(a.OrderByItems[0].Item, b.OrderByItems[0].Item)
	if err != nil {
		// The bytes are not JSON; keep the heap well-defined and let the row
		// fail when it is emitted.
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
// Single-key ORDER BY only. Multi-key ORDER BY requires a Cosmos composite
// index on the container — if the container lacks one the gateway rejects the
// query outright; if it has one the gateway emits a plan with multiple entries
// in orderByExpressions, and the comparator would need to walk all of them.
// Both cases are deferred.
//
// The operators ORDER BY composes with all apply to the merged stream, after
// the k-way merge has put the rows in order:
//
//	OFFSET/LIMIT — the gateway widens the per-partition query to
//	  `OFFSET 0 LIMIT (offset+limit)` so every partition delivers enough rows
//	  to cover any interleaving, and the global skip + take happens here.
//	TOP — the per-partition query carries `TOP n`, so each partition delivers
//	  its own first n and the global first n is among them.
//	DISTINCT — the payload of each merged row is deduplicated against every
//	  row already emitted, before OFFSET/LIMIT counts it.
//
// A range answers over as many pages as its share of the result needs. The
// merge therefore runs only while every range that still has pages has a row
// buffered: the first range to run dry ends the turn, because its next page
// may sort ahead of every other range's head.
type orderByPipeline struct {
	rewrittenQuery string
	fanout         rangeFanout
	descending     bool

	partitionByID map[string]int
	partitions    []*partitionStream

	closed bool

	// offset and limit are the plan's global skip and take; a negative limit
	// means unlimited. skipped counts rows dropped by offset; emitted counts
	// rows forwarded to the caller — both drive the skip+take logic in merge.
	offset  int
	limit   int
	skipped int
	emitted int

	// seen is the DISTINCT seen-set, nil unless the plan carries a DISTINCT.
	// Keys are the canonical JSON of an emitted payload, as in
	// distinctPipeline: a sorted stream makes duplicates adjacent, but the
	// service reports distinctType "Unordered" for the object form even under
	// ORDER BY, so the engine keeps the whole set rather than the last row.
	seen map[string]struct{}

	// pending emits produced by k-way merging when all partitions have data.
	pending [][]byte
}

// newOrderByPipeline constructs a pipeline for a plan with exactly one ORDER BY
// expression. Returns ErrUnsupportedPlanFeature when the plan names multiple
// ORDER BY keys, which the comparator cannot walk.
func newOrderByPipeline(plan *planDoc, pkRangeIDs []string) (*orderByPipeline, error) {
	if len(plan.QueryInfo.OrderBy) != 1 {
		return nil, ErrUnsupportedPlanFeature
	}
	if plan.QueryInfo.OrderBy[0] != "Ascending" && plan.QueryInfo.OrderBy[0] != "Descending" {
		return nil, ErrUnsupportedPlanFeature
	}
	// Substitute the formattable-order-by-query-filter placeholder with
	// "true": every request this pipeline makes, first page or later, selects
	// the whole range and relies on the continuation token to resume it.
	query := strings.ReplaceAll(plan.QueryInfo.RewrittenQuery, orderByQueryFilterPlaceholder, "true")

	offset, limit, err := planSkipTake(plan.QueryInfo)
	if err != nil {
		return nil, err
	}

	var seen map[string]struct{}
	if plan.QueryInfo.DistinctType != "" {
		seen = make(map[string]struct{})
	}

	partByID := make(map[string]int, len(pkRangeIDs))
	partitions := make([]*partitionStream, len(pkRangeIDs))
	for i, id := range pkRangeIDs {
		partByID[id] = i
		partitions[i] = &partitionStream{id: id}
	}
	return &orderByPipeline{
		rewrittenQuery: query,
		fanout:         newRangeFanout(pkRangeIDs),
		descending:     plan.QueryInfo.OrderBy[0] == "Descending",
		partitionByID:  partByID,
		partitions:     partitions,
		offset:         offset,
		limit:          limit,
		seen:           seen,
	}, nil
}

func (p *orderByPipeline) Query() string { return p.rewrittenQuery }
func (p *orderByPipeline) Close()        { p.closed = true }

// IsComplete returns true once every range has delivered its last page, the
// merge has consumed every buffered row and the last pending row has been
// handed to the caller, or once a LIMIT has been satisfied (in which case
// unprocessed partition rows are intentionally discarded).
func (p *orderByPipeline) IsComplete() bool {
	if len(p.pending) > 0 {
		return false
	}
	// The caller has everything it asked for. Any rows still sitting in
	// partition streams were produced speculatively by the gateway (it fetched
	// OFFSET+LIMIT rows per partition to cover any interleaving) and are safe
	// to drop.
	if p.windowFull() {
		return true
	}
	if !p.fanout.allDrained() {
		return false
	}
	for _, s := range p.partitions {
		if _, ok := s.peek(); ok {
			return false
		}
	}
	return true
}

// Run drives one pipeline turn.
//  1. Merge whatever the buffered pages make certain, and emit it.
//  2. Otherwise ask for the next page of every range the merge is waiting on
//     — the initial fan-out across all ranges on the first call.
func (p *orderByPipeline) Run() (*queryengine.PipelineResult, error) {
	if err := p.merge(); err != nil {
		return nil, err
	}

	if len(p.pending) > 0 {
		batch := p.pending
		p.pending = nil
		return &queryengine.PipelineResult{Items: batch, IsCompleted: p.IsComplete()}, nil
	}
	if p.IsComplete() {
		return &queryengine.PipelineResult{IsCompleted: true}, nil
	}
	return &queryengine.PipelineResult{Requests: p.fanout.requestsWhere(p.rewrittenQuery, p.needsPage)}, nil
}

// needsPage reports whether the merge can use another page of the named range
// yet. Only a range whose buffered rows are all consumed can: the merge pops
// in sort order, so a range that still has a row is never what blocks it, and
// fetching that range's next page early would buy a page the merge cannot use
// — a whole range's worth of them, for a query whose ORDER BY puts that
// range's rows last and whose LIMIT may stop before reaching them.
func (p *orderByPipeline) needsPage(id string) bool {
	idx, ok := p.partitionByID[id]
	if !ok {
		return false
	}
	_, buffered := p.partitions[idx].peek()
	return !buffered
}

// ProvideData parses each response and buffers its documents against the
// range that produced them. Validates that every document has at least one
// orderByItems entry and a payload.
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
		idx, ok := p.partitionByID[r.PartitionKeyRangeID]
		if !ok {
			return errors.Errorf("gonative: unexpected partition id %q", r.PartitionKeyRangeID)
		}
		if err := p.fanout.note(r); err != nil {
			return err
		}
		s := p.partitions[idx]
		// Drop the rows the merge has already emitted before buffering the new
		// page, so a range's buffer stays the size of its unread rows rather
		// than of everything it has ever returned.
		if s.head > 0 {
			s.docs = s.docs[s.head:]
			s.head = 0
		}
		s.docs = append(s.docs, resp.Documents...)
	}
	return nil
}

// merge flushes as much of the k-way merge into p.pending as the buffered
// pages make certain. Popping the heap's top, advancing that range and
// re-pushing it emits the globally-next row; a range that runs dry while it
// still has pages ends the turn instead, because its next page may sort ahead
// of every other range's head.
//
// DISTINCT, OFFSET and LIMIT apply here, in that order: a row that repeats an
// already-emitted payload is dropped before it counts against OFFSET, the
// first `offset` surviving rows are discarded, and emission stops once `limit`
// rows have been collected (when limit >= 0). All three are tracked on the
// pipeline so the next turn resumes where this one stopped.
func (p *orderByPipeline) merge() error {
	if p.waitingForPage() {
		return nil
	}
	h := &mergeHeap{partitions: p.partitions, descending: p.descending}
	for i, s := range p.partitions {
		if _, ok := s.peek(); ok {
			h.idx = append(h.idx, i)
		}
	}
	heap.Init(h)

	for h.Len() > 0 {
		top, ok := heap.Pop(h).(int)
		if !ok {
			// Unreachable: the heap is seeded with partition indexes only.
			return nil
		}
		s := p.partitions[top]
		doc := s.pop()
		if err := p.ambiguousFirst(doc, top); err != nil {
			return err
		}
		fresh, err := p.firstSighting(doc.Payload)
		if err != nil {
			return err
		}
		switch {
		case !fresh:
			// DISTINCT already emitted this payload.
		case p.skipped < p.offset:
			p.skipped++
		default:
			// Defensive copy — json.RawMessage aliases the underlying buffer.
			p.pending = append(p.pending, append([]byte(nil), doc.Payload...))
			p.emitted++
		}
		if p.windowFull() {
			// The caller has everything it asked for. Putting the range back
			// would compare its next row against the other heads, and a row
			// outside the window has no business failing the query.
			return nil
		}
		if _, ok := s.peek(); ok {
			heap.Push(h, top)
			continue
		}
		if !p.fanout.drained(s.id) {
			return nil
		}
	}
	return nil
}

// ambiguousFirst reports an error when another range's head ties with doc for
// the position doc is about to take, in a way the engine cannot settle: the
// same composite class, a different value. The service orders within those
// classes by its index's own encoding, so which of the two comes first is not
// knowable here.
//
// Every scalar is unambiguous, and so is a composite no other head shares a
// class with — including one the heap reached only because the ranges below it
// drained. A later row of the same range is not a competitor: the service
// already placed it behind doc in that range's own stream.
func (p *orderByPipeline) ambiguousFirst(doc *orderByDoc, from int) error {
	if len(doc.OrderByItems) == 0 {
		return nil
	}
	item := doc.OrderByItems[0].Item
	class := detectJSONType(item)
	if class != jsonArray && class != jsonObject {
		return nil
	}
	for i, stream := range p.partitions {
		if i == from {
			continue
		}
		head, ok := stream.peek()
		if !ok || len(head.OrderByItems) == 0 {
			continue
		}
		other := head.OrderByItems[0].Item
		if detectJSONType(other) != class {
			continue
		}
		order, err := compareComposites(item, other)
		if err != nil {
			return err
		}
		if order != 0 {
			return errors.Wrapf(ErrUnsupportedPlanFeature,
				"gonative: ORDER BY cannot order two different %ss", class)
		}
	}
	return nil
}

// windowFull reports whether LIMIT, or the TOP folded into it, is satisfied.
func (p *orderByPipeline) windowFull() bool {
	return p.limit >= 0 && p.emitted >= p.limit
}

// firstSighting reports whether a DISTINCT lets this payload through, and
// records it. Always true for a plan without a DISTINCT.
func (p *orderByPipeline) firstSighting(payload json.RawMessage) (bool, error) {
	if p.seen == nil {
		return true, nil
	}
	canonical, err := canonicalJSON(payload)
	if err != nil {
		return false, err
	}
	key := string(canonical)
	if _, ok := p.seen[key]; ok {
		return false, nil
	}
	p.seen[key] = struct{}{}
	return true, nil
}

// waitingForPage reports whether a range has nothing buffered and pages still
// to come, which is what makes the merge's next pop uncertain.
func (p *orderByPipeline) waitingForPage() bool {
	for _, s := range p.partitions {
		if _, buffered := s.peek(); !buffered && !p.fanout.drained(s.id) {
			return true
		}
	}
	return false
}
