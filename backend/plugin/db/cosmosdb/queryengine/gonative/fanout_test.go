package gonative

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRangeFanout_InitialFanOutCoversEveryRange(t *testing.T) {
	f := newRangeFanout([]string{"0", "1", "2"})
	reqs := f.requests("SELECT 1")

	require.Len(t, reqs, 3)
	ids := make(map[string]bool)
	seen := make(map[uint64]bool)
	for _, req := range reqs {
		assert.Empty(t, req.Continuation, "the first page of a range carries no token")
		assert.Equal(t, "SELECT 1", req.Query)
		assert.True(t, req.IncludeParameters)
		assert.False(t, seen[req.Id], "request ids must be distinct")
		seen[req.Id] = true
		ids[req.PartitionKeyRangeID] = true
	}
	assert.Equal(t, map[string]bool{"0": true, "1": true, "2": true}, ids)
	assert.False(t, f.allDrained())
}

func TestRangeFanout_AsksNothingWhileResponsesAreOutstanding(t *testing.T) {
	f := newRangeFanout([]string{"0", "1"})
	require.Len(t, f.requests("q"), 2)
	assert.Empty(t, f.requests("q"),
		"a range with a response outstanding must not be asked again")
}

func TestRangeFanout_FollowsAContinuationToken(t *testing.T) {
	f := newRangeFanout([]string{"0", "1"})
	require.Len(t, f.requests("q"), 2)
	require.NoError(t, f.note(queryengine.NewQueryResultString("0", `{}`, "token-0")))
	require.NoError(t, f.note(queryengine.NewQueryResultString("1", `{}`, "")))

	reqs := f.requests("q")
	require.Len(t, reqs, 1, "only the range that reported a token has another page")
	assert.Equal(t, "0", reqs[0].PartitionKeyRangeID)
	assert.Equal(t, "token-0", reqs[0].Continuation)

	assert.True(t, f.drained("1"))
	assert.False(t, f.drained("0"))
	assert.False(t, f.allDrained())

	require.NoError(t, f.note(queryengine.NewQueryResultString("0", `{}`, "")))
	assert.True(t, f.allDrained())
	assert.Empty(t, f.requests("q"))
}

func TestRangeFanout_RejectsATokenThatDoesNotAdvance(t *testing.T) {
	f := newRangeFanout([]string{"0"})
	require.Len(t, f.requests("q"), 1)
	require.NoError(t, f.note(queryengine.NewQueryResultString("0", `{}`, "token")))
	require.Len(t, f.requests("q"), 1)

	err := f.note(queryengine.NewQueryResultString("0", `{}`, "token"))
	require.Error(t, err, "a token that repeats itself would page forever")
	assert.Contains(t, err.Error(), "same continuation token twice")
}

func TestRangeFanout_RejectsAnUnknownRange(t *testing.T) {
	f := newRangeFanout([]string{"0"})
	err := f.note(queryengine.NewQueryResultString("9", `{}`, ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown partition key range")
}

// ------------------------------------------------------- multi-page test driver

// page is one response the gateway returns for a range: a body, and the
// continuation token that says whether another page follows.
type page struct {
	body         []byte
	continuation string
}

// drivePages runs a pipeline to completion against a per-range script of
// pages, answering each request with that range's next page and asserting that
// a follow-up request carries the token the previous page handed out. It
// returns every emitted row in order, and how many pages of each range the
// pipeline read — a pipeline that stops early must not have read them all.
func drivePages(t *testing.T, p queryengine.QueryPipeline, feed map[string][]page) (rows [][]byte, read map[string]int) {
	t.Helper()
	next := make(map[string]int, len(feed))
	read = make(map[string]int, len(feed))
	// A pipeline that asks for the same page forever would otherwise hang the
	// test; the bound is well above the longest script here.
	for range 200 {
		if p.IsComplete() {
			return rows, read
		}
		res, err := p.Run()
		require.NoError(t, err)
		rows = append(rows, res.Items...)
		if len(res.Items) > 0 {
			continue
		}
		require.NotEmpty(t, res.Requests, "pipeline neither emitted a row nor asked for a page")

		results := make([]queryengine.QueryResult, 0, len(res.Requests))
		for _, req := range res.Requests {
			script, ok := feed[req.PartitionKeyRangeID]
			require.True(t, ok, "missing fixture for %s", req.PartitionKeyRangeID)
			i := next[req.PartitionKeyRangeID]
			require.Less(t, i, len(script), "range %s was asked for a page past its script", req.PartitionKeyRangeID)
			if i == 0 {
				assert.Empty(t, req.Continuation, "the first page of range %s carries no token", req.PartitionKeyRangeID)
			} else {
				assert.Equal(t, script[i-1].continuation, req.Continuation,
					"range %s must resume from the token its previous page returned", req.PartitionKeyRangeID)
			}
			next[req.PartitionKeyRangeID] = i + 1
			read[req.PartitionKeyRangeID] = i + 1
			results = append(results, queryengine.NewQueryResult(req.PartitionKeyRangeID, script[i].body, script[i].continuation))
		}
		require.NoError(t, p.ProvideData(results))
	}
	t.Fatal("pipeline did not complete")
	return rows, read
}
