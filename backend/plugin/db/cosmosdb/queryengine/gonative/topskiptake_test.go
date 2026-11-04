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

// driveTop feeds the given per-partition bodies through a TOP pipeline and
// returns the emitted rows.
func driveTop(t *testing.T, p *topPipeline, feed map[string][]byte) [][]byte {
	t.Helper()
	res, err := p.Run()
	require.NoError(t, err)
	if p.n == 0 {
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

func TestNewTopPipeline_RequiresNonNilTop(t *testing.T) {
	plan := &planDoc{}
	_, err := newTopPipeline(plan, []string{"0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-nil Top")
}

func TestNewTopPipeline_RejectsNegativeN(t *testing.T) {
	plan := newTopPlan(-1)
	_, err := newTopPipeline(plan, []string{"0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative TOP")
}

func TestNewTopPipeline_RejectsCompositionWithOrderBy(t *testing.T) {
	plan := newTopPlan(5)
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	_, err := newTopPipeline(plan, []string{"0"})
	require.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}

func TestNewTopPipeline_RejectsCompositionWithDistinct(t *testing.T) {
	plan := newTopPlan(5)
	plan.QueryInfo.DistinctType = "Unordered"
	_, err := newTopPipeline(plan, []string{"0"})
	require.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}

func TestNewTopPipeline_RejectsCompositionWithAggregate(t *testing.T) {
	plan := newTopPlan(5)
	plan.QueryInfo.Aggregates = []string{"Count"}
	_, err := newTopPipeline(plan, []string{"0"})
	require.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}

// ---------------------------------------------------------------- semantics

func TestTopPipeline_SinglePartition_EmitsFirstN(t *testing.T) {
	plan := newTopPlan(3)
	p, err := newTopPipeline(plan, []string{"0"})
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
	p, err := newTopPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := driveTop(t, p, map[string][]byte{
		"0": mkTopBody(t, `{"id":1}`, `{"id":2}`),
	})
	assert.Len(t, rows, 2)
	assert.True(t, p.IsComplete())
}

func TestTopPipeline_MultiPartition_CapsAtN(t *testing.T) {
	plan := newTopPlan(4)
	p, err := newTopPipeline(plan, []string{"A", "B", "C"})
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
	p, err := newTopPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := driveTop(t, p, nil)
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

func TestTopPipeline_EmptyPartitions(t *testing.T) {
	plan := newTopPlan(5)
	p, err := newTopPipeline(plan, []string{"A", "B"})
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
	p, err := newTopPipeline(newTopPlan(5), []string{"0"})
	require.NoError(t, err)

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
	p, err := newTopPipeline(newTopPlan(4), []string{"0"})
	require.NoError(t, err)

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
