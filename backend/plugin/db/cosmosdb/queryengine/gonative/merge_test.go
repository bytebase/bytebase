package gonative

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mkOrderByBody constructs a partition response body carrying an arbitrary
// sequence of (orderByValue, payload) pairs. The sequence is NOT sorted here —
// callers pass a pre-sorted list because the gateway would have sorted within
// the partition.
func mkOrderByBody(t *testing.T, entries ...struct {
	ob, payload string
}) []byte {
	t.Helper()
	docs := make([]orderByDoc, 0, len(entries))
	for _, e := range entries {
		docs = append(docs, orderByDoc{
			OrderByItems: []orderByItem{{Item: json.RawMessage(e.ob)}},
			Payload:      json.RawMessage(e.payload),
		})
	}
	body, err := json.Marshal(struct {
		Documents []orderByDoc `json:"Documents"`
	}{docs})
	require.NoError(t, err)
	return body
}

// ordered is a shorthand for the (orderByItems[0].item, payload) tuple used
// in fixture construction.
type ordered = struct {
	ob, payload string
}

func newAscPipeline(t *testing.T, pkRangeIDs ...string) *orderByPipeline {
	t.Helper()
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.RewrittenQuery = `SELECT c._rid, [{"item":c.x}] AS orderByItems, c AS payload FROM c WHERE ({documentdb-formattableorderbyquery-filter}) ORDER BY c.x ASC`
	p, err := newOrderByPipeline(plan, pkRangeIDs)
	require.NoError(t, err)
	return p
}

func newDescPipeline(t *testing.T, pkRangeIDs ...string) *orderByPipeline {
	t.Helper()
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Descending"}
	plan.QueryInfo.RewrittenQuery = `SELECT c._rid, [{"item":c.x}] AS orderByItems, c AS payload FROM c WHERE ({documentdb-formattableorderbyquery-filter}) ORDER BY c.x DESC`
	p, err := newOrderByPipeline(plan, pkRangeIDs)
	require.NoError(t, err)
	return p
}

// runOnePass drives the pipeline through: initial Run() → feed → flush.
// Returns every emitted row as a JSON string.
func runOnePass(t *testing.T, p *orderByPipeline, feed map[string][]byte) []string {
	t.Helper()
	res, err := p.Run()
	require.NoError(t, err)
	require.NotEmpty(t, res.Requests, "expected initial fan-out")

	results := make([]queryengine.QueryResult, 0, len(res.Requests))
	for _, req := range res.Requests {
		body, ok := feed[req.PartitionKeyRangeID]
		require.True(t, ok, "missing fixture for %s", req.PartitionKeyRangeID)
		results = append(results, queryengine.NewQueryResult(req.PartitionKeyRangeID, body, ""))
	}
	require.NoError(t, p.ProvideData(results))

	var out []string
	for !p.IsComplete() {
		r, err := p.Run()
		require.NoError(t, err)
		for _, it := range r.Items {
			out = append(out, string(it))
		}
	}
	return out
}

// --------------------------------------------------------------- placeholder

func TestNewOrderByPipeline_SubstitutesFilterPlaceholder(t *testing.T) {
	p := newAscPipeline(t, "0")
	assert.NotContains(t, p.Query(), orderByQueryFilterPlaceholder)
	assert.Contains(t, p.Query(), "WHERE (true)",
		"initial per-partition request must substitute the placeholder with true")
}

// ----------------------------------------------------------- single-partition

func TestOrderByPipeline_SinglePartition_Ascending(t *testing.T) {
	p := newAscPipeline(t, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `5`, payload: `{"x":5}`},
			ordered{ob: `9`, payload: `{"x":9}`},
		),
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":5}`, `{"x":9}`}, rows)
}

func TestOrderByPipeline_SinglePartition_Descending(t *testing.T) {
	p := newDescPipeline(t, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `9`, payload: `{"x":9}`},
			ordered{ob: `5`, payload: `{"x":5}`},
			ordered{ob: `1`, payload: `{"x":1}`},
		),
	})
	assert.Equal(t, []string{`{"x":9}`, `{"x":5}`, `{"x":1}`}, rows)
}

// ----------------------------------------------------------- multi-partition

func TestOrderByPipeline_MultiPartitionKWayMerge_Ascending(t *testing.T) {
	// Three pre-sorted streams; expect interleaved global order.
	p := newAscPipeline(t, "A", "B", "C")
	rows := runOnePass(t, p, map[string][]byte{
		"A": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"id":"a1"}`},
			ordered{ob: `4`, payload: `{"id":"a4"}`},
			ordered{ob: `7`, payload: `{"id":"a7"}`},
		),
		"B": mkOrderByBody(t,
			ordered{ob: `2`, payload: `{"id":"b2"}`},
			ordered{ob: `5`, payload: `{"id":"b5"}`},
			ordered{ob: `8`, payload: `{"id":"b8"}`},
		),
		"C": mkOrderByBody(t,
			ordered{ob: `3`, payload: `{"id":"c3"}`},
			ordered{ob: `6`, payload: `{"id":"c6"}`},
			ordered{ob: `9`, payload: `{"id":"c9"}`},
		),
	})
	assert.Equal(t, []string{
		`{"id":"a1"}`, `{"id":"b2"}`, `{"id":"c3"}`,
		`{"id":"a4"}`, `{"id":"b5"}`, `{"id":"c6"}`,
		`{"id":"a7"}`, `{"id":"b8"}`, `{"id":"c9"}`,
	}, rows)
}

func TestOrderByPipeline_MultiPartitionKWayMerge_Descending(t *testing.T) {
	p := newDescPipeline(t, "A", "B")
	rows := runOnePass(t, p, map[string][]byte{
		"A": mkOrderByBody(t,
			ordered{ob: `9`, payload: `{"id":"a9"}`},
			ordered{ob: `5`, payload: `{"id":"a5"}`},
			ordered{ob: `1`, payload: `{"id":"a1"}`},
		),
		"B": mkOrderByBody(t,
			ordered{ob: `10`, payload: `{"id":"b10"}`},
			ordered{ob: `7`, payload: `{"id":"b7"}`},
			ordered{ob: `3`, payload: `{"id":"b3"}`},
		),
	})
	assert.Equal(t, []string{
		`{"id":"b10"}`, `{"id":"a9"}`, `{"id":"b7"}`, `{"id":"a5"}`, `{"id":"b3"}`, `{"id":"a1"}`,
	}, rows)
}

// --------------------------------------- type-aware ordering (Cosmos rules)

func TestOrderByPipeline_MixedTypes_Ascending(t *testing.T) {
	// Cosmos order: undefined < null < bool < number < string.
	// One partition emits numbers, another emits strings. ASC => numbers first.
	p := newAscPipeline(t, "nums", "strs")
	rows := runOnePass(t, p, map[string][]byte{
		"nums": mkOrderByBody(t,
			ordered{ob: `3`, payload: `{"kind":"num","v":3}`},
			ordered{ob: `7`, payload: `{"kind":"num","v":7}`},
		),
		"strs": mkOrderByBody(t,
			ordered{ob: `"AD"`, payload: `{"kind":"str","v":"AD"}`},
			ordered{ob: `"AE"`, payload: `{"kind":"str","v":"AE"}`},
		),
	})
	assert.Equal(t, []string{
		`{"kind":"num","v":3}`,
		`{"kind":"num","v":7}`,
		`{"kind":"str","v":"AD"}`,
		`{"kind":"str","v":"AE"}`,
	}, rows)
}

func TestOrderByPipeline_MixedTypes_Descending(t *testing.T) {
	p := newDescPipeline(t, "nums", "strs")
	rows := runOnePass(t, p, map[string][]byte{
		"nums": mkOrderByBody(t,
			ordered{ob: `7`, payload: `{"v":7}`},
			ordered{ob: `3`, payload: `{"v":3}`},
		),
		"strs": mkOrderByBody(t,
			ordered{ob: `"AE"`, payload: `{"v":"AE"}`},
			ordered{ob: `"AD"`, payload: `{"v":"AD"}`},
		),
	})
	// DESC: strings first (largest), then numbers.
	assert.Equal(t, []string{
		`{"v":"AE"}`, `{"v":"AD"}`, `{"v":7}`, `{"v":3}`,
	}, rows)
}

// --------------------------------------- empty and malformed partitions

func TestOrderByPipeline_EmptyPartition(t *testing.T) {
	p := newAscPipeline(t, "A", "B")
	rows := runOnePass(t, p, map[string][]byte{
		"A": mkOrderByBody(t), // empty
		"B": mkOrderByBody(t, ordered{ob: `5`, payload: `{"x":5}`}),
	})
	assert.Equal(t, []string{`{"x":5}`}, rows)
	assert.True(t, p.IsComplete())
}

func TestOrderByPipeline_AllPartitionsEmpty(t *testing.T) {
	p := newAscPipeline(t, "A", "B")
	rows := runOnePass(t, p, map[string][]byte{
		"A": mkOrderByBody(t),
		"B": mkOrderByBody(t),
	})
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

func TestOrderByPipeline_RejectsMultiKeyPlan(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending", "Descending"}
	plan.QueryInfo.RewrittenQuery = "SELECT ... ORDER BY c.a, c.b"
	_, err := newOrderByPipeline(plan, []string{"0"})
	require.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}

func TestOrderByPipeline_RejectsMissingPayload(t *testing.T) {
	p := newAscPipeline(t, "0")
	_, _ = p.Run()
	// Document with orderByItems but no payload.
	body := []byte(`{"Documents":[{"orderByItems":[{"item":1}]}]}`)
	err := p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: body},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing payload")
}

func TestOrderByPipeline_RejectsMissingOrderByItems(t *testing.T) {
	p := newAscPipeline(t, "0")
	_, _ = p.Run()
	body := []byte(`{"Documents":[{"payload":{"x":1}}]}`)
	err := p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: body},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing orderByItems")
}

// ------------------------------------------------------------- OFFSET/LIMIT

func newAscOffsetLimit(t *testing.T, offset, limit int, pkRangeIDs ...string) *orderByPipeline {
	t.Helper()
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.RewrittenQuery = `SELECT c._rid, [{"item":c.x}] AS orderByItems, c AS payload FROM c WHERE ({documentdb-formattableorderbyquery-filter}) ORDER BY c.x ASC`
	off := offset
	lim := limit
	plan.QueryInfo.Offset = &off
	plan.QueryInfo.Limit = &lim
	p, err := newOrderByPipeline(plan, pkRangeIDs)
	require.NoError(t, err)
	return p
}

func TestOrderByPipeline_Offset_SkipsFirstN(t *testing.T) {
	p := newAscOffsetLimit(t, 3, 10, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `2`, payload: `{"x":2}`},
			ordered{ob: `3`, payload: `{"x":3}`},
			ordered{ob: `4`, payload: `{"x":4}`},
			ordered{ob: `5`, payload: `{"x":5}`},
		),
	})
	assert.Equal(t, []string{`{"x":4}`, `{"x":5}`}, rows)
}

func TestOrderByPipeline_Limit_CapsAtN(t *testing.T) {
	p := newAscOffsetLimit(t, 0, 3, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `2`, payload: `{"x":2}`},
			ordered{ob: `3`, payload: `{"x":3}`},
			ordered{ob: `4`, payload: `{"x":4}`},
			ordered{ob: `5`, payload: `{"x":5}`},
		),
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":2}`, `{"x":3}`}, rows)
}

func TestOrderByPipeline_OffsetAndLimit_MultiPartition(t *testing.T) {
	// Global merged order: 1,2,3,4,5,6,7,8,9.
	// OFFSET 2 LIMIT 4 → 3,4,5,6.
	p := newAscOffsetLimit(t, 2, 4, "A", "B", "C")
	rows := runOnePass(t, p, map[string][]byte{
		"A": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"id":"a1"}`},
			ordered{ob: `4`, payload: `{"id":"a4"}`},
			ordered{ob: `7`, payload: `{"id":"a7"}`},
		),
		"B": mkOrderByBody(t,
			ordered{ob: `2`, payload: `{"id":"b2"}`},
			ordered{ob: `5`, payload: `{"id":"b5"}`},
			ordered{ob: `8`, payload: `{"id":"b8"}`},
		),
		"C": mkOrderByBody(t,
			ordered{ob: `3`, payload: `{"id":"c3"}`},
			ordered{ob: `6`, payload: `{"id":"c6"}`},
			ordered{ob: `9`, payload: `{"id":"c9"}`},
		),
	})
	assert.Equal(t, []string{
		`{"id":"c3"}`, `{"id":"a4"}`, `{"id":"b5"}`, `{"id":"c6"}`,
	}, rows)
}

func TestOrderByPipeline_OffsetBeyondAvailableRows(t *testing.T) {
	p := newAscOffsetLimit(t, 100, 10, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `2`, payload: `{"x":2}`},
		),
	})
	assert.Empty(t, rows, "OFFSET beyond the total must emit nothing")
	assert.True(t, p.IsComplete())
}

func TestOrderByPipeline_LimitZero_EmitsNothing(t *testing.T) {
	p := newAscOffsetLimit(t, 0, 0, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `2`, payload: `{"x":2}`},
		),
	})
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

func TestNewOrderByPipeline_RejectsNegativeOffset(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.RewrittenQuery = `SELECT ... ORDER BY c.x`
	neg := -1
	plan.QueryInfo.Offset = &neg
	_, err := newOrderByPipeline(plan, []string{"0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative OFFSET")
}

func TestNewOrderByPipeline_RejectsNegativeLimit(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.RewrittenQuery = `SELECT ... ORDER BY c.x`
	neg := -1
	plan.QueryInfo.Limit = &neg
	_, err := newOrderByPipeline(plan, []string{"0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative LIMIT")
}

func TestOrderByPipeline_RejectsContinuationToken(t *testing.T) {
	p := newAscPipeline(t, "0")
	_, _ = p.Run()
	body := mkOrderByBody(t, ordered{ob: `1`, payload: `{}`})
	err := p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: body, NextContinuation: "cont"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "continuation token")
}
