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

// LIMIT 0 needs no row from any range, so the pipeline completes without
// asking for a page at all.
func TestOrderByPipeline_LimitZero_EmitsNothing(t *testing.T) {
	p := newAscOffsetLimit(t, 0, 0, "0")
	res, err := p.Run()
	require.NoError(t, err)
	assert.Empty(t, res.Items)
	assert.Empty(t, res.Requests)
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

// ------------------------------------------------------------- several pages

func TestOrderByPipeline_ResumesARangeAcrossPages(t *testing.T) {
	p := newAscPipeline(t, "0", "1")
	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkOrderByBody(t,
				ordered{ob: `1`, payload: `{"x":1}`},
				ordered{ob: `5`, payload: `{"x":5}`},
			), continuation: "0-page-2"},
			{body: mkOrderByBody(t, ordered{ob: `9`, payload: `{"x":9}`})},
		},
		"1": {
			{body: mkOrderByBody(t, ordered{ob: `2`, payload: `{"x":2}`})},
		},
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":2}`, `{"x":5}`, `{"x":9}`}, asStrings(rows))
	assert.Equal(t, map[string]int{"0": 2, "1": 1}, read)
}

// A range that runs dry mid-merge must not be overtaken: its next page can
// hold a row that sorts ahead of every other range's head. Merging everything
// buffered before fetching that page would emit 1, 2, 4, 3 here.
func TestOrderByPipeline_WaitsForANextPageRatherThanOvertakeIt(t *testing.T) {
	p := newAscPipeline(t, "0", "1")
	rows, _ := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkOrderByBody(t, ordered{ob: `1`, payload: `{"x":1}`}), continuation: "0-page-2"},
			{body: mkOrderByBody(t, ordered{ob: `3`, payload: `{"x":3}`})},
		},
		"1": {
			{body: mkOrderByBody(t,
				ordered{ob: `2`, payload: `{"x":2}`},
				ordered{ob: `4`, payload: `{"x":4}`},
			)},
		},
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":2}`, `{"x":3}`, `{"x":4}`}, asStrings(rows))
}

func TestOrderByPipeline_DescendingResumesARangeAcrossPages(t *testing.T) {
	p := newDescPipeline(t, "0", "1")
	rows, _ := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkOrderByBody(t, ordered{ob: `9`, payload: `{"x":9}`}), continuation: "0-page-2"},
			{body: mkOrderByBody(t, ordered{ob: `5`, payload: `{"x":5}`})},
		},
		"1": {
			{body: mkOrderByBody(t,
				ordered{ob: `7`, payload: `{"x":7}`},
				ordered{ob: `1`, payload: `{"x":1}`},
			)},
		},
	})
	assert.Equal(t, []string{`{"x":9}`, `{"x":7}`, `{"x":5}`, `{"x":1}`}, asStrings(rows))
}

// A LIMIT the first page already satisfies ends the query there: the rows a
// range still holds were fetched speculatively, and its later pages are never
// asked for.
func TestOrderByPipeline_LimitStopsBeforeTheNextPage(t *testing.T) {
	p := newAscOffsetLimit(t, 0, 2, "0")
	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkOrderByBody(t,
				ordered{ob: `1`, payload: `{"x":1}`},
				ordered{ob: `2`, payload: `{"x":2}`},
			), continuation: "0-page-2"},
			{body: mkOrderByBody(t, ordered{ob: `3`, payload: `{"x":3}`})},
		},
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":2}`}, asStrings(rows))
	assert.Equal(t, map[string]int{"0": 1}, read)
}

// An OFFSET past the first page has to read the next one to answer at all.
func TestOrderByPipeline_OffsetReachesIntoTheNextPage(t *testing.T) {
	p := newAscOffsetLimit(t, 2, 2, "0")
	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkOrderByBody(t,
				ordered{ob: `1`, payload: `{"x":1}`},
				ordered{ob: `2`, payload: `{"x":2}`},
			), continuation: "0-page-2"},
			{body: mkOrderByBody(t,
				ordered{ob: `3`, payload: `{"x":3}`},
				ordered{ob: `4`, payload: `{"x":4}`},
			)},
		},
	})
	assert.Equal(t, []string{`{"x":3}`, `{"x":4}`}, asStrings(rows))
	assert.Equal(t, map[string]int{"0": 2}, read)
}

// Ranges whose sort values do not overlap are consumed one after the other, so
// the merge must leave the later range's pages unread while it works through
// the earlier one. Asking every range with a token for its next page every
// turn buys pages the merge cannot use — here it reads range 1 twice for a
// LIMIT that never returns one of its rows.
func TestOrderByPipeline_LeavesAnUnconsumedRangesPagesUnread(t *testing.T) {
	p := newAscOffsetLimit(t, 0, 2, "0", "1")
	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkOrderByBody(t, ordered{ob: `1`, payload: `{"x":1}`}), continuation: "0-page-2"},
			{body: mkOrderByBody(t, ordered{ob: `2`, payload: `{"x":2}`})},
		},
		"1": {
			{body: mkOrderByBody(t, ordered{ob: `10`, payload: `{"x":10}`}), continuation: "1-page-2"},
			{body: mkOrderByBody(t, ordered{ob: `11`, payload: `{"x":11}`})},
		},
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":2}`}, asStrings(rows))
	assert.Equal(t, map[string]int{"0": 2, "1": 1}, read,
		"range 1 still had a buffered row, so its second page was never needed")
}

// asStrings renders emitted rows for comparison against literal fixtures.
func asStrings(rows [][]byte) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, string(r))
	}
	return out
}

// ------------------------------------------------- TOP and DISTINCT composed

func newTopOrderByPipeline(t *testing.T, n int, pkRangeIDs ...string) *orderByPipeline {
	t.Helper()
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.Top = intPtr(n)
	plan.QueryInfo.RewrittenQuery = `SELECT TOP 5 c._rid, [{"item":c.x}] AS orderByItems, c AS payload FROM c WHERE ({documentdb-formattableorderbyquery-filter}) ORDER BY c.x ASC`
	p, err := newOrderByPipeline(plan, pkRangeIDs)
	require.NoError(t, err)
	return p
}

func newDistinctOrderByPipeline(t *testing.T, pkRangeIDs ...string) *orderByPipeline {
	t.Helper()
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.DistinctType = "Unordered"
	plan.QueryInfo.RewrittenQuery = `SELECT c._rid, [{"item":c.x}] AS orderByItems, {"x":c.x} AS payload FROM c WHERE ({documentdb-formattableorderbyquery-filter}) ORDER BY c.x`
	p, err := newOrderByPipeline(plan, pkRangeIDs)
	require.NoError(t, err)
	return p
}

// The per-partition query carries TOP n, so the merge sees up to n rows from
// every range and the cap has to be applied again over the merged stream.
func TestOrderByPipeline_TopCapsTheMergedStream(t *testing.T) {
	p := newTopOrderByPipeline(t, 3, "0", "1")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `4`, payload: `{"x":4}`},
			ordered{ob: `6`, payload: `{"x":6}`},
		),
		"1": mkOrderByBody(t,
			ordered{ob: `2`, payload: `{"x":2}`},
			ordered{ob: `3`, payload: `{"x":3}`},
			ordered{ob: `5`, payload: `{"x":5}`},
		),
	})
	assert.Equal(t, []string{`{"x":1}`, `{"x":2}`, `{"x":3}`}, rows)
}

// DISTINCT with an ORDER BY dedups the merged stream: the service reports
// distinctType "Unordered" for the object form even under ORDER BY, and the
// per-partition query carries no DISTINCT at all, so every duplicate arrives.
func TestOrderByPipeline_DistinctDedupsTheMergedStream(t *testing.T) {
	p := newDistinctOrderByPipeline(t, "0", "1")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `"c"`, payload: `{"x":"c"}`},
		),
		"1": mkOrderByBody(t,
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `"b"`, payload: `{"x":"b"}`},
			ordered{ob: `"c"`, payload: `{"x":"c"}`},
		),
	})
	assert.Equal(t, []string{`{"x":"a"}`, `{"x":"b"}`, `{"x":"c"}`}, rows)
}

// A duplicate must not spend a row of the LIMIT, or a paged DISTINCT comes
// back short.
func TestOrderByPipeline_DistinctDropsDuplicatesBeforeTheLimit(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.OrderBy = []string{"Ascending"}
	plan.QueryInfo.DistinctType = "Unordered"
	plan.QueryInfo.Limit = intPtr(2)
	plan.QueryInfo.RewrittenQuery = `SELECT c._rid, [{"item":c.x}] AS orderByItems, {"x":c.x} AS payload FROM c WHERE ({documentdb-formattableorderbyquery-filter}) ORDER BY c.x OFFSET 0 LIMIT 2`
	p, err := newOrderByPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `"b"`, payload: `{"x":"b"}`},
		),
	})
	assert.Equal(t, []string{`{"x":"a"}`, `{"x":"b"}`}, rows)
}

// A merge is only correct when its comparator agrees with the order inside the
// streams it merges, and inside the array and object classes nothing derived
// from the values does: the service answers these three [1,2], [2], [1,3],
// while canonical JSON would put them [1,2], [1,3], [2] — an order that is
// neither, and that a TOP 2 would cut in the wrong place.
func TestOrderByPipeline_RefusesAnOrderItCannotMerge(t *testing.T) {
	p := newAscPipeline(t, "0", "1")

	res, err := p.Run()
	require.NoError(t, err)
	require.NotEmpty(t, res.Requests)
	feed := map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `[1,2]`, payload: `{"x":[1,2]}`},
			ordered{ob: `[2]`, payload: `{"x":[2]}`}),
		"1": mkOrderByBody(t, ordered{ob: `[1,3]`, payload: `{"x":[1,3]}`}),
	}
	results := make([]queryengine.QueryResult, 0, len(res.Requests))
	for _, req := range res.Requests {
		results = append(results, queryengine.NewQueryResult(req.PartitionKeyRangeID, feed[req.PartitionKeyRangeID], ""))
	}
	require.NoError(t, p.ProvideData(results))

	_, err = p.Run()
	assert.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}

// One array among scalars is still ordered, because the class ranking is the
// service's own: every array sorts after every string.
func TestOrderByPipeline_OrdersOneCompositeAmongScalars(t *testing.T) {
	p := newAscPipeline(t, "0", "1")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `"a"`, payload: `{"x":"a"}`},
			ordered{ob: `[9]`, payload: `{"x":[9]}`}),
		"1": mkOrderByBody(t, ordered{ob: `"b"`, payload: `{"x":"b"}`}),
	})
	assert.Equal(t, []string{`{"x":"a"}`, `{"x":"b"}`, `{"x":[9]}`}, rows)
}

// Rows past the window never reach the caller, so an order the engine cannot
// produce among them must not fail the query: TOP 2 over these streams is
// 0, 1, and the arrays behind them are nobody's business.
func TestOrderByPipeline_StopsAtTheLimitBeforeComparingExcludedRows(t *testing.T) {
	p := newTopOrderByPipeline(t, 2, "0", "1")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `0`, payload: `{"x":0}`},
			ordered{ob: `[1]`, payload: `{"x":[1]}`}),
		"1": mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `[2]`, payload: `{"x":[2]}`}),
	})
	assert.Equal(t, []string{`{"x":0}`, `{"x":1}`}, rows)
}

// TOP 0 asks for nothing, so no range is asked for a page and no row is ever
// compared.
func TestOrderByPipeline_AsksNothingOfRowsUnderTopZero(t *testing.T) {
	p := newTopOrderByPipeline(t, 0, "0", "1")

	res, err := p.Run()
	require.NoError(t, err)
	assert.Empty(t, res.Requests)
	assert.True(t, p.IsComplete())
}

// A page in flight when the window fills is dropped rather than merged, so an
// order the engine cannot produce inside it cannot fail a query that is already
// answered.
func TestOrderByPipeline_IgnoresAPageThatLandsAfterTheLimit(t *testing.T) {
	p := newTopOrderByPipeline(t, 1, "0", "1")

	res, err := p.Run()
	require.NoError(t, err)
	require.Len(t, res.Requests, 2)
	require.NoError(t, p.ProvideData([]queryengine.QueryResult{
		queryengine.NewQueryResult("0", mkOrderByBody(t, ordered{ob: `0`, payload: `{"x":0}`}), "more-0"),
		queryengine.NewQueryResult("1", mkOrderByBody(t, ordered{ob: `1`, payload: `{"x":1}`}), "more-1"),
	}))

	flushed, err := p.Run()
	require.NoError(t, err)
	got := make([]string, 0, len(flushed.Items))
	for _, item := range flushed.Items {
		got = append(got, string(item))
	}
	assert.Equal(t, []string{`{"x":0}`}, got)
	require.True(t, p.IsComplete())

	require.NoError(t, p.ProvideData([]queryengine.QueryResult{
		queryengine.NewQueryResult("0", mkOrderByBody(t, ordered{ob: `[1]`, payload: `{"x":[1]}`}), ""),
		queryengine.NewQueryResult("1", mkOrderByBody(t, ordered{ob: `[2]`, payload: `{"x":[2]}`}), ""),
	}))

	// Nothing more may come out of it, and nothing in that page may fail it.
	late, err := p.Run()
	require.NoError(t, err)
	assert.Empty(t, late.Items)
	assert.True(t, p.IsComplete())
}

// Three ranges headed by a scalar and two arrays: the first row is the scalar
// whatever the arrays' order, so TOP 1 is answerable.
func TestOrderByPipeline_AnswersWhenTheFirstRowIsUnambiguous(t *testing.T) {
	p := newTopOrderByPipeline(t, 1, "0", "1", "2")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t, ordered{ob: `0`, payload: `{"x":0}`}),
		"1": mkOrderByBody(t, ordered{ob: `[1]`, payload: `{"x":[1]}`}),
		"2": mkOrderByBody(t, ordered{ob: `[2]`, payload: `{"x":[2]}`}),
	})
	assert.Equal(t, []string{`{"x":0}`}, rows)
}

// One range's own stream is already in the service's order, so two different
// arrays inside it are no trouble at all.
func TestOrderByPipeline_OrdersCompositesWithinOneRange(t *testing.T) {
	p := newAscPipeline(t, "0")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t,
			ordered{ob: `[1]`, payload: `{"x":[1]}`},
			ordered{ob: `[2]`, payload: `{"x":[2]}`}),
	})
	assert.Equal(t, []string{`{"x":[1]}`, `{"x":[2]}`}, rows)
}

// Two ranges holding the same value tie, and a tie is not an ambiguity.
func TestOrderByPipeline_OrdersEqualCompositesAcrossRanges(t *testing.T) {
	p := newAscPipeline(t, "0", "1")
	rows := runOnePass(t, p, map[string][]byte{
		"0": mkOrderByBody(t, ordered{ob: `[1]`, payload: `{"x":[1],"r":0}`}),
		"1": mkOrderByBody(t, ordered{ob: `[1]`, payload: `{"x":[1],"r":1}`}),
	})
	assert.Len(t, rows, 2)
}

// The ambiguity can surface mid-stream, once the scalars ahead of it are gone.
func TestOrderByPipeline_RefusesAnAmbiguityBehindTheScalars(t *testing.T) {
	p := newAscPipeline(t, "0", "1")

	res, err := p.Run()
	require.NoError(t, err)
	results := []queryengine.QueryResult{
		queryengine.NewQueryResult("0", mkOrderByBody(t,
			ordered{ob: `0`, payload: `{"x":0}`},
			ordered{ob: `[1,2]`, payload: `{"x":[1,2]}`}), ""),
		queryengine.NewQueryResult("1", mkOrderByBody(t,
			ordered{ob: `1`, payload: `{"x":1}`},
			ordered{ob: `[1,3]`, payload: `{"x":[1,3]}`}), ""),
	}
	require.Len(t, res.Requests, 2)
	require.NoError(t, p.ProvideData(results))

	var failure error
	for !p.IsComplete() {
		if _, err := p.Run(); err != nil {
			failure = err
			break
		}
	}
	assert.ErrorIs(t, failure, ErrUnsupportedPlanFeature)
}
