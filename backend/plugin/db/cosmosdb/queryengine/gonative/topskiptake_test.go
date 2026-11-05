package gonative

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mkTopBody wraps an arbitrary set of JSON documents into the partition
// response shape the gateway produces for a standalone TOP query.
func mkTopBody(t *testing.T, docs ...string) []byte {
	t.Helper()
	raw := make([]json.RawMessage, len(docs))
	for i, d := range docs {
		raw[i] = json.RawMessage(d)
	}
	body, err := json.Marshal(map[string]any{"Documents": raw})
	require.NoError(t, err)
	return body
}

func intPtr(v int) *int { return &v }

func newTopPlan(n int) *planDoc {
	p := &planDoc{}
	p.QueryInfo.Top = intPtr(n)
	return p
}

func newOffsetLimitPlan(offset, limit int) *planDoc {
	p := &planDoc{}
	p.QueryInfo.Offset = intPtr(offset)
	p.QueryInfo.Limit = intPtr(limit)
	p.QueryInfo.RewrittenQuery = `SELECT c.name FROM c OFFSET 0 LIMIT 5`
	return p
}

func newSkipTake(t *testing.T, plan *planDoc, pkRangeIDs ...string) *skipTakePipeline {
	t.Helper()
	p, err := newSkipTakePipeline(plan, pkRangeIDs, `SELECT * FROM c`)
	require.NoError(t, err)
	return p
}

// driveTop feeds the given per-partition bodies through a skip/take pipeline
// and returns the emitted rows.
func driveTop(t *testing.T, p *skipTakePipeline, feed map[string][]byte) [][]byte {
	t.Helper()
	res, err := p.Run()
	require.NoError(t, err)
	if p.take == 0 {
		// Edge case: TOP 0 completes immediately without any partition work.
		assert.Empty(t, res.Requests)
		return nil
	}
	require.NotEmpty(t, res.Requests, "expected fan-out for non-zero TOP")

	results := make([]queryengine.QueryResult, 0, len(res.Requests))
	for _, req := range res.Requests {
		body, ok := feed[req.PartitionKeyRangeID]
		require.True(t, ok, "missing fixture for %s", req.PartitionKeyRangeID)
		results = append(results, queryengine.NewQueryResult(req.PartitionKeyRangeID, body, ""))
	}
	require.NoError(t, p.ProvideData(results))

	var out [][]byte
	for !p.IsComplete() {
		r, err := p.Run()
		require.NoError(t, err)
		out = append(out, r.Items...)
	}
	return out
}

// ---------------------------------------------------------------- validation

func TestNewSkipTakePipeline_RequiresAWindow(t *testing.T) {
	_, err := newSkipTakePipeline(&planDoc{}, []string{"0"}, `SELECT * FROM c`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires TOP, OFFSET or LIMIT")
}

func TestNewSkipTakePipeline_RejectsNegativeN(t *testing.T) {
	_, err := newSkipTakePipeline(newTopPlan(-1), []string{"0"}, `SELECT * FROM c`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative TOP")
}

// A bare TOP plan carries no rewrittenQuery, so the range answers the user's
// own query; an OFFSET/LIMIT plan carries one widened to cover the window.
func TestNewSkipTakePipeline_QueryComesFromThePlanWhenRewritten(t *testing.T) {
	assert.Equal(t, `SELECT * FROM c`, newSkipTake(t, newTopPlan(3), "0").Query())
	assert.Equal(t, `SELECT c.name FROM c OFFSET 0 LIMIT 5`,
		newSkipTake(t, newOffsetLimitPlan(2, 3), "0").Query())
}

// ---------------------------------------------------------------- semantics

func TestTopPipeline_SinglePartition_EmitsFirstN(t *testing.T) {
	plan := newTopPlan(3)
	p, err := newSkipTakePipeline(plan, []string{"0"}, `SELECT * FROM c`)
	require.NoError(t, err)

	rows := driveTop(t, p, map[string][]byte{
		"0": mkTopBody(t, `{"id":"a"}`, `{"id":"b"}`, `{"id":"c"}`, `{"id":"d"}`, `{"id":"e"}`),
	})
	assert.Equal(t, [][]byte{
		[]byte(`{"id":"a"}`), []byte(`{"id":"b"}`), []byte(`{"id":"c"}`),
	}, rows)
}

func TestTopPipeline_FewerRowsThanN_EmitsAll(t *testing.T) {
	plan := newTopPlan(10)
	p, err := newSkipTakePipeline(plan, []string{"0"}, `SELECT * FROM c`)
	require.NoError(t, err)

	rows := driveTop(t, p, map[string][]byte{
		"0": mkTopBody(t, `{"id":1}`, `{"id":2}`),
	})
	assert.Len(t, rows, 2)
	assert.True(t, p.IsComplete())
}

func TestTopPipeline_MultiPartition_CapsAtN(t *testing.T) {
	plan := newTopPlan(4)
	p, err := newSkipTakePipeline(plan, []string{"A", "B", "C"}, `SELECT * FROM c`)
	require.NoError(t, err)

	rows := driveTop(t, p, map[string][]byte{
		"A": mkTopBody(t, `{"p":"A","i":1}`, `{"p":"A","i":2}`, `{"p":"A","i":3}`),
		"B": mkTopBody(t, `{"p":"B","i":1}`, `{"p":"B","i":2}`),
		"C": mkTopBody(t, `{"p":"C","i":1}`, `{"p":"C","i":2}`),
	})
	assert.Len(t, rows, 4, "total emission must be capped at N=4 despite 7 inputs")
}

func TestTopPipeline_ZeroN_EmitsNothing(t *testing.T) {
	plan := newTopPlan(0)
	p, err := newSkipTakePipeline(plan, []string{"0"}, `SELECT * FROM c`)
	require.NoError(t, err)

	rows := driveTop(t, p, nil)
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

func TestTopPipeline_EmptyPartitions(t *testing.T) {
	plan := newTopPlan(5)
	p, err := newSkipTakePipeline(plan, []string{"A", "B"}, `SELECT * FROM c`)
	require.NoError(t, err)

	rows := driveTop(t, p, map[string][]byte{
		"A": mkTopBody(t),
		"B": mkTopBody(t),
	})
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

// A TOP larger than a page reads as many pages as it takes to collect N rows.
func TestTopPipeline_ReadsPagesUntilNRowsAreInHand(t *testing.T) {
	p := newSkipTake(t, newTopPlan(5), "0")

	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkTopBody(t, `{"x":1}`, `{"x":2}`), continuation: "0-page-2"},
			{body: mkTopBody(t, `{"x":3}`, `{"x":4}`), continuation: "0-page-3"},
			{body: mkTopBody(t, `{"x":5}`, `{"x":6}`)},
		},
	})
	assert.Len(t, rows, 5)
	assert.Equal(t, map[string]int{"0": 3}, read)
}

// Once N rows are in hand the pipeline stops, however many pages the ranges
// still hold.
func TestTopPipeline_StopsAtNWithoutReadingTheNextPage(t *testing.T) {
	p := newSkipTake(t, newTopPlan(4), "0")

	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkTopBody(t, `{"x":1}`, `{"x":2}`), continuation: "0-page-2"},
			{body: mkTopBody(t, `{"x":3}`, `{"x":4}`), continuation: "0-page-3"},
			{body: mkTopBody(t, `{"x":5}`, `{"x":6}`)},
		},
	})
	assert.Len(t, rows, 4)
	assert.Equal(t, map[string]int{"0": 2}, read)
}

// ------------------------------------------------- OFFSET/LIMIT, no ORDER BY

// OFFSET/LIMIT without an ORDER BY is ordinary result paging. The gateway
// widens the per-partition query to cover the window, so every range offers
// at least as many rows as the window needs and the cut happens here.
func TestSkipTakePipeline_OffsetLimitCutsOneWindowGlobally(t *testing.T) {
	p := newSkipTake(t, newOffsetLimitPlan(2, 3), "0", "1")
	rows := driveTop(t, p, map[string][]byte{
		"0": mkTopBody(t, `{"x":1}`, `{"x":2}`, `{"x":3}`),
		"1": mkTopBody(t, `{"x":4}`, `{"x":5}`, `{"x":6}`),
	})
	assert.Equal(t, []string{`{"x":3}`, `{"x":4}`, `{"x":5}`}, stringsOf(rows))
}

func TestSkipTakePipeline_OffsetPastEveryRowEmitsNothing(t *testing.T) {
	p := newSkipTake(t, newOffsetLimitPlan(10, 3), "0")
	rows := driveTop(t, p, map[string][]byte{
		"0": mkTopBody(t, `{"x":1}`, `{"x":2}`),
	})
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

// A window that straddles a page boundary has to keep reading the range.
func TestSkipTakePipeline_OffsetLimitReadsPagesUntilTheWindowIsFull(t *testing.T) {
	p := newSkipTake(t, newOffsetLimitPlan(3, 2), "0")
	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkTopBody(t, `{"x":1}`, `{"x":2}`), continuation: "0-page-2"},
			{body: mkTopBody(t, `{"x":3}`, `{"x":4}`), continuation: "0-page-3"},
			{body: mkTopBody(t, `{"x":5}`, `{"x":6}`)},
		},
	})
	assert.Equal(t, []string{`{"x":4}`, `{"x":5}`}, stringsOf(rows))
	assert.Equal(t, map[string]int{"0": 3}, read)
}
