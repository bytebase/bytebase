package gonative

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlan_5_1_AliasedIsGroupByForm(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_1_plan.json"))
	require.NoError(t, err)
	p, err := parsePlan(raw)
	require.NoError(t, err)
	// 5.1 is `SELECT COUNT(1) AS totalRecords FROM c`.
	// IMPORTANT: aliased aggregates use the groupBy-alias wire format (single-group),
	// NOT the aggregates array. See testdata/query_5_1_plan.json.
	assert.Empty(t, p.QueryInfo.Aggregates, "aliased form leaves aggregates[] empty")
	assert.Equal(t, []string{"totalRecords"}, p.QueryInfo.GroupByAliases)
	assert.Equal(t, map[string]string{"totalRecords": "Count"}, p.QueryInfo.GroupByAliasToAggregateType)
	assert.False(t, p.QueryInfo.HasSelectValue)
	assert.Contains(t, p.QueryInfo.RewrittenQuery, "AS payload",
		"aliased form rewrites to {...} AS payload")
}

func TestParsePlan_5_1b_ValueForm(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_1b_plan.json"))
	require.NoError(t, err)
	p, err := parsePlan(raw)
	require.NoError(t, err)
	// 5.1b is `SELECT VALUE COUNT(1) FROM c` — one aggregate, VALUE form.
	assert.Equal(t, []string{"Count"}, p.QueryInfo.Aggregates)
	assert.True(t, p.QueryInfo.HasSelectValue)
	assert.Empty(t, p.QueryInfo.GroupByAliasToAggregateType, "VALUE form has no aliases")
	assert.Contains(t, p.QueryInfo.RewrittenQuery, `SELECT VALUE [{"item":`,
		"VALUE form rewrites to an array of {item: ...} objects")
}

func TestParsePlan_5_5_MultiAgg_Aliased(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_5_plan.json"))
	require.NoError(t, err)
	p, err := parsePlan(raw)
	require.NoError(t, err)
	// 5.5 has two aliased aggregates — same groupBy form, just two aliases.
	assert.Empty(t, p.QueryInfo.Aggregates)
	assert.ElementsMatch(t, []string{"minPop", "maxPop"}, p.QueryInfo.GroupByAliases)
	assert.Equal(t, map[string]string{"minPop": "Min", "maxPop": "Max"}, p.QueryInfo.GroupByAliasToAggregateType)
	// MIN/MAX aliased include "item2" partial for cross-partition correctness.
	assert.Contains(t, p.QueryInfo.RewrittenQuery, `"item2"`,
		"MIN/MAX aliased include item2: {min|max, count} for cross-partition correctness")
}

func TestParsePKRangeIDs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_1_pkranges.json"))
	require.NoError(t, err)
	ids, err := parsePKRangeIDs(raw)
	require.NoError(t, err)
	assert.Equal(t, []string{"0"}, ids, "test container has a single partition with id=0")
}

func TestParsePlan_RejectsInvalidJSON(t *testing.T) {
	_, err := parsePlan([]byte(`not json`))
	require.Error(t, err)
}

func TestParsePKRangeIDs_RejectsInvalidJSON(t *testing.T) {
	_, err := parsePKRangeIDs([]byte(`not json`))
	require.Error(t, err)
}
