package gonative

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGroupPlan builds the minimum plan fields a groupPipeline needs for a
// single-expression GROUP BY with the given alias + aggregate-kind mapping.
// aliases order defines output column order.
func newGroupPlan(groupByExpr string, aliases []string, aliasKinds map[string]string) *planDoc {
	plan := &planDoc{}
	plan.QueryInfo.GroupByExpressions = []string{groupByExpr}
	plan.QueryInfo.GroupByAliases = aliases
	plan.QueryInfo.GroupByAliasToAggregateType = aliasKinds
	return plan
}

// mkGroupBody synthesizes a partition response body from a list of per-group
// documents. Each entry is (groupByItemRaw, payloadRaw) — the full JSON of
// groupByItems[0].item and of the payload object respectively.
func mkGroupBody(t *testing.T, entries ...struct {
	key, payload string
}) []byte {
	t.Helper()
	docs := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		var payloadObj map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(e.payload), &payloadObj), "invalid payload fixture: %s", e.payload)
		docs = append(docs, map[string]any{
			"groupByItems": []map[string]any{{"item": json.RawMessage(e.key)}},
			"payload":      payloadObj,
		})
	}
	b, err := json.Marshal(map[string]any{"Documents": docs})
	require.NoError(t, err)
	return b
}

type groupEntry = struct {
	key, payload string
}

// driveGroup feeds the given partition bodies through a group pipeline and
// returns every emitted row.
func driveGroup(t *testing.T, p *groupPipeline, feed map[string][]byte) [][]byte {
	t.Helper()
	res, err := p.Run()
	require.NoError(t, err)
	require.NotEmpty(t, res.Requests)
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

// rowsByKey indexes GROUP BY output rows by a key extracted via extractor.
// Order of emission isn't deterministic in Go's map iteration, so callers
// should assert on this map rather than on the raw slice order.
func rowsByKey(t *testing.T, rows [][]byte, keyAlias string) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		v, ok := obj[keyAlias].(string)
		require.True(t, ok, "row missing %s key: %s", keyAlias, r)
		out[v] = obj
	}
	return out
}

// ---------------------------------------------------------- validation

func TestNewGroupPipeline_RequiresGroupByExpressions(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.GroupByAliases = []string{"a"}
	_, err := newGroupPipeline(plan, []string{"0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "groupByExpressions")
}

func TestNewGroupPipeline_RequiresGroupByAliases(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.GroupByExpressions = []string{"c.a"}
	_, err := newGroupPipeline(plan, []string{"0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "groupByAliases")
}

// ---------------------------------------------------------- semantics

func TestGroupPipeline_SinglePartition_GroupKeyPassthrough(t *testing.T) {
	// SELECT c.country, COUNT(1) AS cityCount FROM c GROUP BY c.country
	plan := newGroupPlan("c.country",
		[]string{"country", "cityCount"},
		map[string]string{"country": "", "cityCount": "Count"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `"AD"`, payload: `{"country":"AD","cityCount":{"item":3}}`},
			groupEntry{key: `"AE"`, payload: `{"country":"AE","cityCount":{"item":5}}`},
		),
	})
	byKey := rowsByKey(t, rows, "country")
	require.Len(t, byKey, 2)
	assert.EqualValues(t, 3, byKey["AD"]["cityCount"])
	assert.EqualValues(t, 5, byKey["AE"]["cityCount"])
}

func TestGroupPipeline_MultiPartition_MergesSameKey(t *testing.T) {
	plan := newGroupPlan("c.country",
		[]string{"country", "cityCount"},
		map[string]string{"country": "", "cityCount": "Count"})
	p, err := newGroupPipeline(plan, []string{"A", "B", "C"})
	require.NoError(t, err)

	// AE appears in all three partitions with counts 2, 3, 1 → expect 6.
	// US only in A (count 4) and C (count 2) → expect 6.
	// GB only in B (count 7) → expect 7.
	rows := driveGroup(t, p, map[string][]byte{
		"A": mkGroupBody(t,
			groupEntry{key: `"AE"`, payload: `{"country":"AE","cityCount":{"item":2}}`},
			groupEntry{key: `"US"`, payload: `{"country":"US","cityCount":{"item":4}}`},
		),
		"B": mkGroupBody(t,
			groupEntry{key: `"AE"`, payload: `{"country":"AE","cityCount":{"item":3}}`},
			groupEntry{key: `"GB"`, payload: `{"country":"GB","cityCount":{"item":7}}`},
		),
		"C": mkGroupBody(t,
			groupEntry{key: `"AE"`, payload: `{"country":"AE","cityCount":{"item":1}}`},
			groupEntry{key: `"US"`, payload: `{"country":"US","cityCount":{"item":2}}`},
		),
	})
	byKey := rowsByKey(t, rows, "country")
	require.Len(t, byKey, 3)
	assert.EqualValues(t, 6, byKey["AE"]["cityCount"])
	assert.EqualValues(t, 6, byKey["US"]["cityCount"])
	assert.EqualValues(t, 7, byKey["GB"]["cityCount"])
}

func TestGroupPipeline_MultipleAggregates(t *testing.T) {
	// SELECT c.cr, COUNT(1) AS count, SUM(...) AS totalPop FROM c GROUP BY c.cr
	plan := newGroupPlan("c.cr",
		[]string{"cr", "count", "totalPop"},
		map[string]string{"cr": "", "count": "Count", "totalPop": "Sum"})
	p, err := newGroupPipeline(plan, []string{"A", "B"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"A": mkGroupBody(t,
			groupEntry{key: `"r1"`, payload: `{"cr":"r1","count":{"item":2},"totalPop":{"item":1000}}`},
			groupEntry{key: `"r2"`, payload: `{"cr":"r2","count":{"item":1},"totalPop":{"item":500}}`},
		),
		"B": mkGroupBody(t,
			groupEntry{key: `"r1"`, payload: `{"cr":"r1","count":{"item":3},"totalPop":{"item":2500}}`},
		),
	})
	byKey := rowsByKey(t, rows, "cr")
	require.Len(t, byKey, 2)
	assert.EqualValues(t, 5, byKey["r1"]["count"])
	assert.EqualValues(t, 3500, byKey["r1"]["totalPop"])
	assert.EqualValues(t, 1, byKey["r2"]["count"])
	assert.EqualValues(t, 500, byKey["r2"]["totalPop"])
}

func TestGroupPipeline_PreservesAliasOrder(t *testing.T) {
	// Rely on the raw emitted bytes to check alias order — map iteration is
	// non-deterministic, but the output JSON is built manually to honor
	// groupByAliases order.
	plan := newGroupPlan("c.country",
		[]string{"country", "cityCount"},
		map[string]string{"country": "", "cityCount": "Count"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `"AD"`, payload: `{"country":"AD","cityCount":{"item":3}}`},
		),
	})
	require.Len(t, rows, 1)
	assert.Equal(t, `{"country":"AD","cityCount":3}`, string(rows[0]))
}

func TestGroupPipeline_EmptyPartitions(t *testing.T) {
	plan := newGroupPlan("c.a",
		[]string{"a", "n"},
		map[string]string{"a": "", "n": "Count"})
	p, err := newGroupPipeline(plan, []string{"A", "B"})
	require.NoError(t, err)
	rows := driveGroup(t, p, map[string][]byte{
		"A": mkGroupBody(t),
		"B": mkGroupBody(t),
	})
	assert.Empty(t, rows)
	assert.True(t, p.IsComplete())
}

func TestGroupPipeline_GroupKeysBeforeAggregatesInAliasOrder(t *testing.T) {
	// Aggregate listed first, group key second — output should still honor that order.
	plan := newGroupPlan("c.country",
		[]string{"cityCount", "country"},
		map[string]string{"country": "", "cityCount": "Count"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)
	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `"AE"`, payload: `{"country":"AE","cityCount":{"item":5}}`},
		),
	})
	require.Len(t, rows, 1)
	assert.Equal(t, `{"cityCount":5,"country":"AE"}`, string(rows[0]))
}

// A group whose rows straddle a page boundary gets one output row, with the
// partials from every page folded into it.
func TestGroupPipeline_FoldsAGroupsPartialsAcrossPages(t *testing.T) {
	plan := newGroupPlan("c.a",
		[]string{"a", "n"},
		map[string]string{"a": "", "n": "Count"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkGroupBody(t, groupEntry{key: `"x"`, payload: `{"a":"x","n":{"item":2}}`}),
				continuation: "0-page-2"},
			{body: mkGroupBody(t,
				groupEntry{key: `"x"`, payload: `{"a":"x","n":{"item":3}}`},
				groupEntry{key: `"y"`, payload: `{"a":"y","n":{"item":1}}`},
			)},
		},
	})
	got := make([]string, 0, len(rows))
	for _, r := range rows {
		got = append(got, string(r))
	}
	assert.ElementsMatch(t, []string{`{"a":"x","n":5}`, `{"a":"y","n":1}`}, got)
	assert.Equal(t, map[string]int{"0": 2}, read)
}

func TestGroupPipeline_RejectsMissingPayloadAlias(t *testing.T) {
	plan := newGroupPlan("c.a",
		[]string{"a", "n"},
		map[string]string{"a": "", "n": "Count"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)
	_, _ = p.Run()

	body := mkGroupBody(t, groupEntry{key: `"x"`, payload: `{"a":"x"}`}) // missing n
	err = p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: body},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `alias "n"`)
}

// ------------------------------------------------------- group-key canonicalization

func TestGroupKey_TreatsKeyReorderedObjectsAsSameGroup(t *testing.T) {
	// Two docs whose groupByItems differ only by object key ordering must hash
	// to the same group key.
	a, err := groupKey([]groupByItem{{Item: json.RawMessage(`{"x":1,"y":2}`)}})
	require.NoError(t, err)
	b, err := groupKey([]groupByItem{{Item: json.RawMessage(`{"y":2,"x":1}`)}})
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestGroupKey_DifferentTypesHashDifferently(t *testing.T) {
	a, err := groupKey([]groupByItem{{Item: json.RawMessage(`"1"`)}})
	require.NoError(t, err)
	b, err := groupKey([]groupByItem{{Item: json.RawMessage(`1`)}})
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

// ------------------------------------------------------- extractAggregatePartial

func TestExtractAggregatePartial_CountUsesItem(t *testing.T) {
	got, err := extractAggregatePartial("Count", json.RawMessage(`{"item":42}`))
	require.NoError(t, err)
	assert.Equal(t, "42", string(got))
}

func TestExtractAggregatePartial_SumUsesItem(t *testing.T) {
	got, err := extractAggregatePartial("Sum", json.RawMessage(`{"item":3500}`))
	require.NoError(t, err)
	assert.Equal(t, "3500", string(got))
}

func TestExtractAggregatePartial_AvgUsesItem(t *testing.T) {
	got, err := extractAggregatePartial("Average", json.RawMessage(`{"item":{"sum":100,"count":4}}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"sum":100,"count":4}`, string(got))
}

func TestExtractAggregatePartial_MinPrefersItem2WhenPresent(t *testing.T) {
	got, err := extractAggregatePartial("Min", json.RawMessage(`{"item":10,"item2":{"min":10,"count":3}}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"min":10,"count":3}`, string(got))
}

func TestExtractAggregatePartial_MinSynthesizesFromItemWhenItem2Absent(t *testing.T) {
	got, err := extractAggregatePartial("Min", json.RawMessage(`{"item":42}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"min":42,"count":1}`, string(got))
}

func TestExtractAggregatePartial_MaxSynthesizesFromItemWhenItem2Absent(t *testing.T) {
	got, err := extractAggregatePartial("Max", json.RawMessage(`{"item":99}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"max":99,"count":1}`, string(got))
}

// ---------------------------------------------------------- output order

// groupKeysOf returns the group-key column of every emitted row, in the order
// the pipeline emitted them.
func groupKeysOf(t *testing.T, rows [][]byte, keyAlias string) []string {
	t.Helper()
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		v, ok := obj[keyAlias].(string)
		require.True(t, ok, "row missing %s key: %s", keyAlias, r)
		out = append(out, v)
	}
	return out
}

func countByCountryPlan() *planDoc {
	return newGroupPlan("c.country",
		[]string{"country", "n"},
		map[string]string{"country": "", "n": "Count"})
}

// orderedCountByCountryPlan is countByCountryPlan for a query the user also
// ordered. Such a plan carries no order fields at all — the clause survives
// only in the rewritten query, spelled as the gateway writes it.
func orderedCountByCountryPlan(clause string) *planDoc {
	plan := countByCountryPlan()
	plan.QueryInfo.RewrittenQuery = `SELECT [{"item": c.country}] AS groupByItems, ` +
		`{"country": c.country, "n": {"item": COUNT(1)}} AS payload` + "\n" +
		`FROM c` + "\n" + `GROUP BY c.country` + "\n" + clause
	return plan
}

// The direction comes from the query, not from the order the ranges happen to
// answer in: these two feed the same rows and differ only in the clause.
func TestGroupPipeline_EmitsAscendingWhenTheQueryOrdersAscending(t *testing.T) {
	p, err := newGroupPipeline(orderedCountByCountryPlan("ORDER BY c.country"), []string{"0", "1"})
	require.NoError(t, err)

	rows := driveGroup(t, p, descendingCountryFeed(t))
	assert.Equal(t, []string{"AE", "BR", "CN", "DE"}, groupKeysOf(t, rows, "country"))
}

func TestGroupPipeline_EmitsDescendingWhenTheQueryOrdersDescending(t *testing.T) {
	p, err := newGroupPipeline(orderedCountByCountryPlan("ORDER BY c.country DESC"), []string{"0", "1"})
	require.NoError(t, err)

	rows := driveGroup(t, p, descendingCountryFeed(t))
	assert.Equal(t, []string{"DE", "CN", "BR", "AE"}, groupKeysOf(t, rows, "country"))
}

func descendingCountryFeed(t *testing.T) map[string][]byte {
	t.Helper()
	return map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `"DE"`, payload: `{"country":"DE","n":{"item":1}}`},
			groupEntry{key: `"AE"`, payload: `{"country":"AE","n":{"item":2}}`},
		),
		"1": mkGroupBody(t,
			groupEntry{key: `"CN"`, payload: `{"country":"CN","n":{"item":3}}`},
			groupEntry{key: `"BR"`, payload: `{"country":"BR","n":{"item":4}}`},
		),
	}
}

// A range that reports a single group says nothing about the direction, which
// is why reading it off the data answered a DESC query ascending.
func TestGroupPipeline_EmitsDescendingWhenEveryRangeHasOneGroup(t *testing.T) {
	p, err := newGroupPipeline(orderedCountByCountryPlan("ORDER BY c.country DESC"), []string{"0", "1"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t, groupEntry{key: `"AE"`, payload: `{"country":"AE","n":{"item":2}}`}),
		"1": mkGroupBody(t, groupEntry{key: `"BR"`, payload: `{"country":"BR","n":{"item":4}}`}),
	})
	assert.Equal(t, []string{"BR", "AE"}, groupKeysOf(t, rows, "country"))
}

// An order the engine cannot produce by comparing group keys has to fail the
// query: ordering by the keys anyway would answer in an order the user did not
// ask for. The service refuses these itself today, so this is the engine
// holding the line.
func TestGroupPipeline_RejectsAnOrderItCannotProduce(t *testing.T) {
	for _, tc := range []struct {
		name   string
		clause string
	}{
		{"by an aggregate", "ORDER BY COUNT(1) DESC"},
		{"by an output alias", "ORDER BY n DESC"},
		{"by a column the query does not group by", "ORDER BY c.city"},
		{"by something that is not a list of keys", "ORDER BY c.country NULLS LAST"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newGroupPipeline(orderedCountByCountryPlan(tc.clause), []string{"0", "1"})
			assert.ErrorIs(t, err, ErrUnsupportedPlanFeature)
		})
	}
}

// Cosmos gives every ORDER BY key its own direction, and accepts a clause that
// names only some of the group keys.
func TestGroupPipeline_OrdersMultipleKeysEachItsOwnWay(t *testing.T) {
	feed := map[string][]byte{
		"0": mkTwoKeyGroupBody(t,
			twoKeyEntry{keys: []string{`"AE"`, `2`}, payload: `{"country":"AE","city":2,"n":{"item":7}}`},
			twoKeyEntry{keys: []string{`"AE"`, `1`}, payload: `{"country":"AE","city":1,"n":{"item":8}}`},
		),
		"1": mkTwoKeyGroupBody(t,
			twoKeyEntry{keys: []string{`"BR"`, `1`}, payload: `{"country":"BR","city":1,"n":{"item":9}}`},
			twoKeyEntry{keys: []string{`"BR"`, `2`}, payload: `{"country":"BR","city":2,"n":{"item":6}}`},
		),
	}

	for _, tc := range []struct {
		name   string
		clause string
		want   []string
	}{
		{"ascending then descending", "ORDER BY c.country, c.city DESC", []string{"AE/2", "AE/1", "BR/2", "BR/1"}},
		{"descending then ascending", "ORDER BY c.country DESC, c.city", []string{"BR/1", "BR/2", "AE/1", "AE/2"}},
		{"only the second key", "ORDER BY c.city DESC", []string{"AE/2", "BR/2", "AE/1", "BR/1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := newGroupPipeline(countryCityPlan(tc.clause), []string{"0", "1"})
			require.NoError(t, err)
			assert.Equal(t, tc.want, countryCityKeysOf(t, driveGroup(t, p, feed)))
		})
	}
}

// A group key the clause leaves out still gets an order, so that two runs of
// one query answer alike however the ranges happened to report their groups.
func TestGroupPipeline_OrdersKeysTheClauseLeavesOut(t *testing.T) {
	p, err := newGroupPipeline(countryCityPlan("ORDER BY c.city DESC"), []string{"0", "1"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkTwoKeyGroupBody(t,
			twoKeyEntry{keys: []string{`"BR"`, `2`}, payload: `{"country":"BR","city":2,"n":{"item":6}}`},
		),
		"1": mkTwoKeyGroupBody(t,
			twoKeyEntry{keys: []string{`"AE"`, `2`}, payload: `{"country":"AE","city":2,"n":{"item":7}}`},
		),
	})
	assert.Equal(t, []string{"AE/2", "BR/2"}, countryCityKeysOf(t, rows))
}

// countryCityPlan groups by two keys, which the gateway reports in
// groupByItems in the order groupByExpressions names them.
func countryCityPlan(clause string) *planDoc {
	plan := newGroupPlan("c.country",
		[]string{"country", "city", "n"},
		map[string]string{"country": "", "city": "", "n": "Count"})
	plan.QueryInfo.GroupByExpressions = []string{"c.country", "c.city"}
	plan.QueryInfo.RewrittenQuery = "SELECT 1\nFROM c\nGROUP BY c.country, c.city\n" + clause
	return plan
}

// mkTwoKeyGroupBody is mkGroupBody for a two-expression GROUP BY: each
// document's groupByItems carries both keys, in groupByExpressions order.
func mkTwoKeyGroupBody(t *testing.T, entries ...struct {
	keys    []string
	payload string
}) []byte {
	t.Helper()
	docs := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		var payloadObj map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(e.payload), &payloadObj), "invalid payload fixture: %s", e.payload)
		items := make([]map[string]any, 0, len(e.keys))
		for _, key := range e.keys {
			items = append(items, map[string]any{"item": json.RawMessage(key)})
		}
		docs = append(docs, map[string]any{"groupByItems": items, "payload": payloadObj})
	}
	b, err := json.Marshal(map[string]any{"Documents": docs})
	require.NoError(t, err)
	return b
}

type twoKeyEntry = struct {
	keys    []string
	payload string
}

func countryCityKeysOf(t *testing.T, rows [][]byte) []string {
	t.Helper()
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var obj struct {
			Country string `json:"country"`
			City    int    `json:"city"`
		}
		require.NoError(t, json.Unmarshal(r, &obj), "row: %s", r)
		out = append(out, fmt.Sprintf("%s/%d", obj.Country, obj.City))
	}
	return out
}

// A key whose name holds a clause keyword is still a key.
func TestGroupPipeline_ReadsAnOrderWhoseKeyNameHoldsAKeyword(t *testing.T) {
	plan := newGroupPlan("c.nowhere", []string{"nowhere", "n"}, map[string]string{"nowhere": "", "n": "Count"})
	plan.QueryInfo.RewrittenQuery = "SELECT 1\nFROM c\nGROUP BY c.nowhere\nORDER BY c.nowhere DESC"

	p, err := newGroupPipeline(plan, []string{"0", "1"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t, groupEntry{key: `"AE"`, payload: `{"nowhere":"AE","n":{"item":2}}`}),
		"1": mkGroupBody(t, groupEntry{key: `"BR"`, payload: `{"nowhere":"BR","n":{"item":4}}`}),
	})
	assert.Equal(t, []string{"BR", "AE"}, groupKeysOf(t, rows, "nowhere"))
}

// The words can appear in the query body — in a string literal the user wrote
// — without the query naming an order.
func TestGroupPipeline_IgnoresOrderByInsideTheQueryBody(t *testing.T) {
	plan := countByCountryPlan()
	plan.QueryInfo.RewrittenQuery = `SELECT [{"item": c.country}] AS groupByItems, ` +
		`{"country": c.country, "n": {"item": COUNT(1)}} AS payload` + "\n" +
		`FROM c` + "\n" + `WHERE c.note != 'order by me'` + "\n" + `GROUP BY c.country`

	p, err := newGroupPipeline(plan, []string{"0", "1"})
	require.NoError(t, err)

	rows := driveGroup(t, p, descendingCountryFeed(t))
	assert.Equal(t, []string{"AE", "BR", "CN", "DE"}, groupKeysOf(t, rows, "country"))
}

// Without an ORDER BY the output order is unspecified, but it still has to be
// the same on every run: before this it followed Go's map iteration and five
// consecutive runs of one query gave five different orders.
func TestGroupPipeline_OutputOrderIsStableAcrossRuns(t *testing.T) {
	feed := map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `"AE"`, payload: `{"country":"AE","n":{"item":2}}`},
			groupEntry{key: `"DE"`, payload: `{"country":"DE","n":{"item":1}}`},
			groupEntry{key: `"BR"`, payload: `{"country":"BR","n":{"item":4}}`},
			groupEntry{key: `"CN"`, payload: `{"country":"CN","n":{"item":3}}`},
			groupEntry{key: `"US"`, payload: `{"country":"US","n":{"item":9}}`},
		),
	}
	var first []string
	for range 5 {
		p, err := newGroupPipeline(countByCountryPlan(), []string{"0"})
		require.NoError(t, err)
		got := groupKeysOf(t, driveGroup(t, p, feed), "country")
		if first == nil {
			first = got
			continue
		}
		assert.Equal(t, first, got)
	}
}

// A group key Cosmos cannot order — an array or an object — leaves the output
// in first-seen order rather than asking the comparator to invent one.
func TestGroupPipeline_UnorderableKeysKeepFirstSeenOrder(t *testing.T) {
	plan := newGroupPlan("c.tags", []string{"tags", "n"}, map[string]string{"tags": "", "n": "Count"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `["b"]`, payload: `{"tags":["b"],"n":{"item":1}}`},
			groupEntry{key: `["a"]`, payload: `{"tags":["a"],"n":{"item":2}}`},
		),
	})
	require.Len(t, rows, 2)
	assert.JSONEq(t, `{"tags":["b"],"n":1}`, string(rows[0]))
	assert.JSONEq(t, `{"tags":["a"],"n":2}`, string(rows[1]))
}

// MIN over a group where the column is undefined in every row has no value,
// and the gateway leaves that alias out of the row rather than writing null.
func TestGroupPipeline_OmitsAnAliasWithNoValue(t *testing.T) {
	plan := newGroupPlan("c.country", []string{"country", "lo"}, map[string]string{"country": "", "lo": "Min"})
	p, err := newGroupPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows := driveGroup(t, p, map[string][]byte{
		"0": mkGroupBody(t,
			groupEntry{key: `"AE"`, payload: `{"country":"AE","lo":{"item2":{"count":0}}}`},
		),
	})
	require.Len(t, rows, 1)
	assert.JSONEq(t, `{"country":"AE"}`, string(rows[0]))
}
