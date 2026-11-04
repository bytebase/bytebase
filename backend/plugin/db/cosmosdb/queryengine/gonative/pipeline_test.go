package gonative_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db/cosmosdb/queryengine/gonative"
)

// mustReadFile loads a fixture path and fails the test on error.
func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)
	return b
}

// partitionBodies returns a map of pk-range-id -> response body for each file
// found under `dir`. Expects filenames of the form `range_<id>.json`.
func partitionBodies(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "read dir %s", dir)
	out := make(map[string][]byte, len(entries))
	for _, e := range entries {
		name := e.Name()
		// file is `range_<id>.json` — strip prefix/suffix
		id := name[len("range_") : len(name)-len(".json")]
		out[id] = mustReadFile(t, filepath.Join(dir, name))
	}
	return out
}

// driveAggPipeline runs a pipeline end-to-end against captured fixtures and
// returns the single emitted row.
func driveAggPipeline(t *testing.T, queryID, originalQuery string) json.RawMessage {
	t.Helper()
	base := filepath.Join("testdata", "query_"+queryID)
	plan := mustReadFile(t, base+"_plan.json")
	pkranges := mustReadFile(t, base+"_pkranges.json")
	bodies := partitionBodies(t, base+"_partitions")

	engine := gonative.Default()
	pipe, err := engine.CreateQueryPipeline(originalQuery, string(plan), string(pkranges))
	require.NoError(t, err, "CreateQueryPipeline")
	defer pipe.Close()

	var emitted json.RawMessage
	for i := 0; i < 10 && !pipe.IsComplete(); i++ { // bounded loop — 10 turns is a safe upper bound for single-partition aggregates
		res, err := pipe.Run()
		require.NoError(t, err, "Run turn %d", i)
		if len(res.Requests) > 0 {
			results := make([]queryengine.QueryResult, 0, len(res.Requests))
			for _, req := range res.Requests {
				body, ok := bodies[req.PartitionKeyRangeID]
				require.True(t, ok, "missing fixture for pk-range %s", req.PartitionKeyRangeID)
				results = append(results, queryengine.NewQueryResult(req.PartitionKeyRangeID, body, ""))
			}
			require.NoError(t, pipe.ProvideData(results))
		}
		if len(res.Items) > 0 {
			require.Len(t, res.Items, 1, "aggregate query emits exactly one row")
			emitted = append(json.RawMessage(nil), res.Items[0]...)
		}
	}
	require.True(t, pipe.IsComplete(), "pipeline did not converge")
	require.NotNil(t, emitted, "no row emitted")
	return emitted
}

func TestAggPipeline_5_1_AliasedCount(t *testing.T) {
	row := driveAggPipeline(t, "5_1", `SELECT COUNT(1) AS totalRecords FROM c`)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(row, &obj))
	assert.EqualValues(t, 18, obj["totalRecords"])
}

func TestAggPipeline_5_1b_ValueCount(t *testing.T) {
	row := driveAggPipeline(t, "5_1b", `SELECT VALUE COUNT(1) FROM c`)
	// VALUE form emits the scalar directly.
	var n float64
	require.NoError(t, json.Unmarshal(row, &n))
	assert.EqualValues(t, 18, n)
}

func TestAggPipeline_5_2_AliasedCountWithFilter(t *testing.T) {
	row := driveAggPipeline(t, "5_2", `SELECT COUNT(1) AS aeCount FROM c WHERE c.country = "AE"`)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(row, &obj))
	assert.EqualValues(t, 5, obj["aeCount"])
}

func TestAggPipeline_5_2b_ValueCountWithFilter(t *testing.T) {
	row := driveAggPipeline(t, "5_2b", `SELECT VALUE COUNT(1) FROM c WHERE c.country = "AE"`)
	var n float64
	require.NoError(t, json.Unmarshal(row, &n))
	assert.EqualValues(t, 5, n)
}

func TestAggPipeline_5_3_AliasedSum(t *testing.T) {
	row := driveAggPipeline(t, "5_3", `SELECT SUM(StringToNumber(c.population)) AS totalPop FROM c`)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(row, &obj))
	assert.Positive(t, obj["totalPop"])
}

func TestAggPipeline_5_3b_ValueSum(t *testing.T) {
	row := driveAggPipeline(t, "5_3b", `SELECT VALUE SUM(StringToNumber(c.population)) FROM c`)
	var n float64
	require.NoError(t, json.Unmarshal(row, &n))
	assert.Positive(t, n)
}

func TestAggPipeline_5_4_AliasedAvg(t *testing.T) {
	row := driveAggPipeline(t, "5_4", `SELECT AVG(StringToNumber(c.population)) AS avgPop FROM c`)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(row, &obj))
	assert.Positive(t, obj["avgPop"])
}

func TestAggPipeline_5_4b_ValueAvg(t *testing.T) {
	row := driveAggPipeline(t, "5_4b", `SELECT VALUE AVG(StringToNumber(c.population)) FROM c`)
	var n float64
	require.NoError(t, json.Unmarshal(row, &n))
	assert.Positive(t, n)
}

func TestAggPipeline_5_5_MultiAggMinMax(t *testing.T) {
	row := driveAggPipeline(t, "5_5", `SELECT MIN(StringToNumber(c.population)) AS minPop, MAX(StringToNumber(c.population)) AS maxPop FROM c WHERE StringToNumber(c.population) > 0`)
	// Aliased multi-agg preserves alias order from the plan.
	assert.Contains(t, string(row), `"minPop"`)
	assert.Contains(t, string(row), `"maxPop"`)
	// Order assertion: minPop appears before maxPop because the plan lists it first.
	idxMin := indexOf(row, []byte(`"minPop"`))
	idxMax := indexOf(row, []byte(`"maxPop"`))
	assert.Less(t, idxMin, idxMax, "aliases emitted in plan order")

	var obj map[string]any
	require.NoError(t, json.Unmarshal(row, &obj))
	minV, ok := obj["minPop"].(float64)
	require.True(t, ok)
	maxV, ok := obj["maxPop"].(float64)
	require.True(t, ok)
	assert.LessOrEqual(t, minV, maxV)
}

// drivePipelineAll runs a pipeline end-to-end against captured fixtures and
// returns every emitted row. Used for operators that emit more than one row
// (DISTINCT, later Stage 3+ ORDER BY/TOP/LIMIT/GROUP BY).
func drivePipelineAll(t *testing.T, queryID, originalQuery string) []json.RawMessage {
	t.Helper()
	base := filepath.Join("testdata", "query_"+queryID)
	plan := mustReadFile(t, base+"_plan.json")
	pkranges := mustReadFile(t, base+"_pkranges.json")
	bodies := partitionBodies(t, base+"_partitions")

	engine := gonative.Default()
	pipe, err := engine.CreateQueryPipeline(originalQuery, string(plan), string(pkranges))
	require.NoError(t, err, "CreateQueryPipeline")
	defer pipe.Close()

	var rows []json.RawMessage
	for i := 0; i < 50 && !pipe.IsComplete(); i++ {
		res, err := pipe.Run()
		require.NoError(t, err, "Run turn %d", i)
		if len(res.Requests) > 0 {
			results := make([]queryengine.QueryResult, 0, len(res.Requests))
			for _, req := range res.Requests {
				body, ok := bodies[req.PartitionKeyRangeID]
				require.True(t, ok, "missing fixture for pk-range %s", req.PartitionKeyRangeID)
				results = append(results, queryengine.NewQueryResult(req.PartitionKeyRangeID, body, ""))
			}
			require.NoError(t, pipe.ProvideData(results))
		}
		for _, it := range res.Items {
			rows = append(rows, append(json.RawMessage(nil), it...))
		}
	}
	require.True(t, pipe.IsComplete(), "pipeline did not converge")
	return rows
}

func TestDistinctPipeline_10_1_ObjectDistinct(t *testing.T) {
	rows := drivePipelineAll(t, "10_1", `SELECT DISTINCT c.country FROM c`)
	// Captured fixture has 6 unique countries in a single partition.
	assert.Len(t, rows, 6)
	seen := map[string]bool{}
	for _, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		country, ok := obj["country"].(string)
		require.True(t, ok, "each row must carry a country field")
		assert.False(t, seen[country], "duplicate country emitted: %s", country)
		seen[country] = true
	}
}

func TestGroupPipeline_6_1_CountByCountry(t *testing.T) {
	rows := drivePipelineAll(t, "6_1", `SELECT c.country, COUNT(1) AS cityCount FROM c GROUP BY c.country`)
	// Captured fixture covers 6 countries.
	assert.Len(t, rows, 6)
	byCountry := map[string]float64{}
	for _, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		country, ok := obj["country"].(string)
		require.True(t, ok, "row missing country: %s", r)
		cnt, ok := obj["cityCount"].(float64)
		require.True(t, ok, "row missing cityCount: %s", r)
		byCountry[country] = cnt
	}
	// AE has 5 cities in the test container (from Stage 1 captures).
	assert.EqualValues(t, 5, byCountry["AE"])
	// Verify sum of group counts equals the total row count (18).
	var total float64
	for _, n := range byCountry {
		total += n
	}
	assert.Equal(t, float64(18), total)
}

func TestGroupPipeline_6_2_MultipleAggregates(t *testing.T) {
	rows := drivePipelineAll(t, "6_2", `SELECT c.countryRegion, COUNT(1) AS count, SUM(StringToNumber(c.population)) AS totalPop FROM c GROUP BY c.countryRegion`)
	assert.NotEmpty(t, rows)
	// Each row must carry all three aliases and their values must be numeric
	// (count) / numeric (totalPop) / string (countryRegion).
	var totalCount float64
	for _, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		_, ok := obj["countryRegion"].(string)
		assert.True(t, ok, "countryRegion should be a string: %s", r)
		cnt, ok := obj["count"].(float64)
		require.True(t, ok, "count should be numeric: %s", r)
		_, ok = obj["totalPop"].(float64)
		assert.True(t, ok, "totalPop should be numeric: %s", r)
		totalCount += cnt
	}
	assert.Equal(t, float64(18), totalCount, "sum of per-region counts must equal total row count")
}

func TestOrderByPipeline_12_1_OffsetZeroLimitTen(t *testing.T) {
	rows := drivePipelineAll(t, "12_1", `SELECT * FROM c ORDER BY c.name OFFSET 0 LIMIT 10`)
	assert.Len(t, rows, 10, "OFFSET 0 LIMIT 10 must emit 10 rows")
	// Rows should be in ascending order by c.name.
	var prevName string
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj), "row %d: %s", i, r)
		name, ok := obj["name"].(string)
		require.True(t, ok, "name must be a string: %s", r)
		if i > 0 {
			assert.LessOrEqual(t, prevName, name, "ascending order broken at row %d", i)
		}
		prevName = name
	}
}

func TestOrderByPipeline_12_2_OffsetTenLimitTen(t *testing.T) {
	rows := drivePipelineAll(t, "12_2", `SELECT * FROM c ORDER BY c.name OFFSET 10 LIMIT 10`)
	// WorldCities has 18 rows total; OFFSET 10 LIMIT 10 returns rows 11–18 (8 rows).
	assert.Len(t, rows, 8)
	var prevName string
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		name, ok := obj["name"].(string)
		require.True(t, ok, "name must be a string: %s", r)
		if i > 0 {
			assert.LessOrEqual(t, prevName, name)
		}
		prevName = name
	}
}

func TestTopPipeline_1_3_TopTen(t *testing.T) {
	rows := drivePipelineAll(t, "1_3", `SELECT TOP 10 * FROM c`)
	assert.Len(t, rows, 10, "TOP 10 must emit exactly 10 rows")
	// Every row should be a valid object (not the payload-wrapped form).
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj), "row %d: %s", i, r)
		assert.NotEmpty(t, obj, "row %d should be a non-empty object", i)
	}
}

func TestOrderByPipeline_4_1_Ascending(t *testing.T) {
	rows := drivePipelineAll(t, "4_1", `SELECT c.name, c.population FROM c ORDER BY c.population ASC`)
	// Captured fixture: 18 cities in the WorldCities test container, mixed
	// numeric + string populations (the gateway delivered them pre-sorted).
	assert.Len(t, rows, 18, "expected one row per city")
	// First row should be the smallest population (number type comes before
	// string under Cosmos ordering, so this is a numeric).
	var first map[string]any
	require.NoError(t, json.Unmarshal(rows[0], &first))
	_, ok := first["population"].(float64)
	assert.True(t, ok, "first row under ASC ordering must be numeric population (numbers sort before strings)")
}

func TestOrderByPipeline_4_2_Descending(t *testing.T) {
	rows := drivePipelineAll(t, "4_2", `SELECT c.name, c.population FROM c ORDER BY c.population DESC`)
	assert.Len(t, rows, 18)
	// DESC: strings come first (strings > numbers under Cosmos ordering).
	var first map[string]any
	require.NoError(t, json.Unmarshal(rows[0], &first))
	_, ok := first["population"].(string)
	assert.True(t, ok, "first row under DESC must be a string-valued population (strings sort after numbers)")
}

func TestDistinctPipeline_10_2_DistinctValue(t *testing.T) {
	rows := drivePipelineAll(t, "10_2", `SELECT DISTINCT VALUE c.countryRegion FROM c`)
	// Captured fixture has 6 unique countryRegion scalars.
	assert.Len(t, rows, 6)
	seen := map[string]bool{}
	for _, r := range rows {
		var v string
		require.NoError(t, json.Unmarshal(r, &v))
		assert.False(t, seen[v], "duplicate scalar emitted: %s", v)
		seen[v] = true
	}
}

func TestAggPipeline_11_2_ValueCount(t *testing.T) {
	row := driveAggPipeline(t, "11_2", `SELECT VALUE COUNT(1) FROM c`)
	var n float64
	require.NoError(t, json.Unmarshal(row, &n))
	assert.EqualValues(t, 18, n)
}

// indexOf is a tiny helper around bytes.Index to keep the test file imports minimal.
func indexOf(haystack, needle []byte) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
