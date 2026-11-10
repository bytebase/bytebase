package gonative

import (
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// fanoutRange is one partition key range's paging state.
type fanoutRange struct {
	id string
	// started is false until the range's first request goes out.
	started bool
	// next is the continuation token the range's next request must carry.
	// Empty on a started range means there is nothing to ask for: either a
	// response is still owed for it, or it has drained.
	next string
	// sent is the token the range's last request carried. It guards against a
	// token that never advances, which would otherwise page forever.
	sent    string
	drained bool
}

// rangeFanout tracks the per-range request state of a pipeline that runs one
// query against every partition key range the container reports.
//
// A range's result can span several response pages: the gateway caps a page by
// item count and by size, and signals the rest with a continuation token. The
// fanout hands that token back on the range's next request and treats the
// range as drained only once a response for it carries no token.
//
// The SDK's QueryRequest.Drain flag looks like it would do this, but its
// drain loop reuses the original request for every page without feeding the
// continuation back, so a drained request re-reads page one forever.
type rangeFanout struct {
	ranges []*fanoutRange
	byID   map[string]*fanoutRange
	nextID uint64
}

func newRangeFanout(rangeIDs []string) rangeFanout {
	f := rangeFanout{
		ranges: make([]*fanoutRange, 0, len(rangeIDs)),
		byID:   make(map[string]*fanoutRange, len(rangeIDs)),
	}
	for _, id := range rangeIDs {
		r := &fanoutRange{id: id}
		f.ranges = append(f.ranges, r)
		f.byID[id] = r
	}
	return f
}

// requests returns the work the caller should issue next: the initial fan-out
// on the first call, then one follow-up request per range holding a
// continuation token. It returns nil when every range has either drained or
// has a response outstanding.
func (f *rangeFanout) requests(query string) []queryengine.QueryRequest {
	return f.requestsWhere(query, nil)
}

// requestsWhere is requests narrowed to the ranges want reports true for. An
// operator that consumes its ranges unevenly uses it to leave a range's next
// page unread until it can use it: fetching a page early costs request charge
// and holds the rows in memory, and under a LIMIT the query may never need it
// at all. A range the predicate skips keeps its token and is offered again on
// the next turn.
func (f *rangeFanout) requestsWhere(query string, want func(id string) bool) []queryengine.QueryRequest {
	var reqs []queryengine.QueryRequest
	for _, r := range f.ranges {
		if r.drained || (r.started && r.next == "") {
			continue
		}
		if want != nil && !want(r.id) {
			continue
		}
		r.started = true
		r.sent = r.next
		r.next = ""
		reqs = append(reqs, queryengine.QueryRequest{
			PartitionKeyRangeID: r.id,
			Id:                  f.nextID,
			Query:               query,
			Continuation:        r.sent,
			IncludeParameters:   true,
		})
		f.nextID++
	}
	return reqs
}

// note records what a response says about its range's remaining pages.
func (f *rangeFanout) note(result queryengine.QueryResult) error {
	r, ok := f.byID[result.PartitionKeyRangeID]
	if !ok {
		return errors.Errorf("gonative: response for unknown partition key range %q", result.PartitionKeyRangeID)
	}
	if result.NextContinuation == "" {
		r.next = ""
		r.drained = true
		return nil
	}
	if result.NextContinuation == r.sent {
		return errors.Errorf("gonative: partition key range %q returned the same continuation token twice", r.id)
	}
	r.next = result.NextContinuation
	return nil
}

// drained reports whether the named range has delivered its last page.
func (f *rangeFanout) drained(id string) bool {
	r, ok := f.byID[id]
	return ok && r.drained
}

// allDrained reports whether every range has delivered its last page.
func (f *rangeFanout) allDrained() bool {
	for _, r := range f.ranges {
		if !r.drained {
			return false
		}
	}
	return true
}
