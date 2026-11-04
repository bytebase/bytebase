package gonative

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValuePartition_CountIsNumeric(t *testing.T) {
	// 5.1b: SELECT VALUE COUNT(1) FROM c — partition doc is [ {"item": <number>} ].
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_1b_partitions", "range_0.json"))
	require.NoError(t, err)
	p, err := parseValuePartitionResponse(raw)
	require.NoError(t, err)
	require.NotEmpty(t, p.Documents)
	require.Len(t, p.Documents[0], 1, "single-aggregate partition has one item per document")
	var n float64
	require.NoError(t, json.Unmarshal(p.Documents[0][0].Item, &n))
	assert.Positive(t, n)
}

func TestParseValuePartition_SumIsNumeric(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_3b_partitions", "range_0.json"))
	require.NoError(t, err)
	p, err := parseValuePartitionResponse(raw)
	require.NoError(t, err)
	require.NotEmpty(t, p.Documents)
	require.Len(t, p.Documents[0], 1)
	var n float64
	require.NoError(t, json.Unmarshal(p.Documents[0][0].Item, &n))
	assert.Positive(t, n)
}

func TestParseValuePartition_AvgHasSumAndCount(t *testing.T) {
	// 5.4b: SELECT VALUE AVG(...) FROM c — item is {"sum": s, "count": n}.
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_4b_partitions", "range_0.json"))
	require.NoError(t, err)
	p, err := parseValuePartitionResponse(raw)
	require.NoError(t, err)
	require.NotEmpty(t, p.Documents)
	require.Len(t, p.Documents[0], 1)
	var partial map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(p.Documents[0][0].Item, &partial))
	assert.Contains(t, partial, "sum")
	assert.Contains(t, partial, "count")
}

func TestParseAliasedPartition_Count(t *testing.T) {
	// 5.1: SELECT COUNT(1) AS totalRecords FROM c — payload.totalRecords.item.
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_1_partitions", "range_0.json"))
	require.NoError(t, err)
	p, err := parseAliasedPartitionResponse(raw)
	require.NoError(t, err)
	require.NotEmpty(t, p.Documents)
	alias, ok := p.Documents[0].Payload["totalRecords"]
	require.True(t, ok, "expected totalRecords alias in payload")
	var n float64
	require.NoError(t, json.Unmarshal(alias.Item, &n))
	assert.Positive(t, n)
	assert.Empty(t, alias.Item2, "Count aliased carries only item, no item2")
}

func TestParseAliasedPartition_SumAndAvg(t *testing.T) {
	// 5.3: SELECT SUM(...) AS totalPop FROM c
	sumRaw, err := os.ReadFile(filepath.Join("testdata", "query_5_3_partitions", "range_0.json"))
	require.NoError(t, err)
	sumP, err := parseAliasedPartitionResponse(sumRaw)
	require.NoError(t, err)
	require.NotEmpty(t, sumP.Documents)
	_, ok := sumP.Documents[0].Payload["totalPop"]
	require.True(t, ok)

	// 5.4: SELECT AVG(...) AS avgPop FROM c — item contains {"sum": s, "count": n}.
	avgRaw, err := os.ReadFile(filepath.Join("testdata", "query_5_4_partitions", "range_0.json"))
	require.NoError(t, err)
	avgP, err := parseAliasedPartitionResponse(avgRaw)
	require.NoError(t, err)
	avgAlias, ok := avgP.Documents[0].Payload["avgPop"]
	require.True(t, ok)
	var partial map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(avgAlias.Item, &partial))
	assert.Contains(t, partial, "sum")
	assert.Contains(t, partial, "count")
}

func TestParseAliasedPartition_MinMaxHasItem2(t *testing.T) {
	// 5.5: multi-agg MIN/MAX aliased — item2 carries the reliable cross-partition partial.
	raw, err := os.ReadFile(filepath.Join("testdata", "query_5_5_partitions", "range_0.json"))
	require.NoError(t, err)
	p, err := parseAliasedPartitionResponse(raw)
	require.NoError(t, err)
	require.NotEmpty(t, p.Documents)

	minPop, ok := p.Documents[0].Payload["minPop"]
	require.True(t, ok)
	require.NotEmpty(t, minPop.Item2, "Min aliased MUST carry item2 for cross-partition correctness")
	var reliable struct {
		Min   json.RawMessage `json:"min"`
		Count int             `json:"count"`
	}
	require.NoError(t, json.Unmarshal(minPop.Item2, &reliable))
	assert.NotZero(t, reliable.Count)

	maxPop, ok := p.Documents[0].Payload["maxPop"]
	require.True(t, ok)
	require.NotEmpty(t, maxPop.Item2, "Max aliased MUST carry item2")
}

func TestParseValuePartition_RejectsInvalidJSON(t *testing.T) {
	_, err := parseValuePartitionResponse([]byte(`not json`))
	require.Error(t, err)
}

func TestParseAliasedPartition_RejectsInvalidJSON(t *testing.T) {
	_, err := parseAliasedPartitionResponse([]byte(`not json`))
	require.Error(t, err)
}
