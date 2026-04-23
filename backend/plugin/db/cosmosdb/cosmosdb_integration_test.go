// Copyright (c) Bytebase (Hong Kong) Limited.
package cosmosdb

// End-to-end integration tests for BYT-9239 — every query class unlocked by
// Stages 1 through 6 of the pure-Go Cosmos query engine, exercised through
// the Bytebase driver's QueryConn path against a live Cosmos account.
//
// Guarded by AZURE_COSMOS_KEY — skips automatically when unset, so default
// unit-test runs stay hermetic.
//
// Running locally against the bytebase-cosmostest account:
//
//   export AZURE_COSMOS_ENDPOINT="https://bytebase-cosmostest.documents.azure.com:443/"
//   export AZURE_COSMOS_KEY="$(az cosmosdb keys list \
//       --name bytebase-cosmostest \
//       --resource-group rg-bytebase-cosmostest \
//       --type keys --query primaryMasterKey -o tsv)"
//   go test -count=1 -run "^TestIntegration_BYT9239" \
//       -timeout 300s ./backend/plugin/db/cosmosdb/

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func skipIfNoAzure(t *testing.T) (endpoint, key, dbName, container string) {
	t.Helper()
	endpoint = os.Getenv("AZURE_COSMOS_ENDPOINT")
	key = os.Getenv("AZURE_COSMOS_KEY")
	if endpoint == "" || key == "" {
		t.Skip("set AZURE_COSMOS_ENDPOINT + AZURE_COSMOS_KEY to run live integration tests")
	}
	dbName = envOrDefault("AZURE_COSMOS_DB", "testdb")
	container = envOrDefault("AZURE_COSMOS_CONTAINER", "WorldCities")
	return endpoint, key, dbName, container
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// newTestDriver constructs a Driver directly with a master-key-authenticated
// client, bypassing Open's Azure-AD path (which requires OAuth credentials).
// Exercises QueryConn — the exact entry point the Bytebase SQL viewer uses.
func newTestDriver(t *testing.T, endpoint, key, dbName string) *Driver {
	t.Helper()
	cred, err := azcosmos.NewKeyCredential(key)
	require.NoError(t, err, "NewKeyCredential")
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	require.NoError(t, err, "NewClientWithKey")
	return &Driver{
		client:       client,
		databaseName: dbName,
		connCfg: db.ConnectionConfig{
			ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
		},
	}
}

// runQuery runs a single query through the driver and returns its emitted
// rows as a slice of JSON documents (one per Row in the QueryResult).
func runQuery(t *testing.T, driver *Driver, container, sql string) [][]byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	results, err := driver.QueryConn(ctx, nil, sql, db.QueryContext{Container: container})
	require.NoError(t, err, "QueryConn: %s", sql)
	require.Len(t, results, 1, "driver returns exactly one QueryResult per query")

	out := make([][]byte, 0, len(results[0].Rows))
	for _, row := range results[0].Rows {
		values := row.GetValues()
		require.Len(t, values, 1, "each row has exactly one column of JSON text")
		out = append(out, []byte(values[0].GetStringValue()))
	}
	return out
}

// ---------------------------------------------------------- Stage 1: aggregates

func TestIntegration_BYT9239_Stage1Aggregates(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	cases := []struct {
		id    string
		sql   string
		check func(t *testing.T, rows [][]byte)
	}{
		{"5.1", `SELECT COUNT(1) AS totalRecords FROM c`,
			func(t *testing.T, r [][]byte) { assertAliasNumeric(t, r, "totalRecords") }},
		{"5.1b", `SELECT VALUE COUNT(1) FROM c`,
			func(t *testing.T, r [][]byte) { assertScalarNumeric(t, r) }},
		{"5.2", `SELECT COUNT(1) AS aeCount FROM c WHERE c.country = "AE"`,
			func(t *testing.T, r [][]byte) { assertAliasNumeric(t, r, "aeCount") }},
		{"5.2b", `SELECT VALUE COUNT(1) FROM c WHERE c.country = "AE"`,
			func(t *testing.T, r [][]byte) { assertScalarNumeric(t, r) }},
		{"5.3", `SELECT SUM(StringToNumber(c.population)) AS totalPop FROM c`,
			func(t *testing.T, r [][]byte) { assertAliasNumeric(t, r, "totalPop") }},
		{"5.3b", `SELECT VALUE SUM(StringToNumber(c.population)) FROM c`,
			func(t *testing.T, r [][]byte) { assertScalarNumeric(t, r) }},
		{"5.4", `SELECT AVG(StringToNumber(c.population)) AS avgPop FROM c`,
			func(t *testing.T, r [][]byte) { assertAliasNumeric(t, r, "avgPop") }},
		{"5.4b", `SELECT VALUE AVG(StringToNumber(c.population)) FROM c`,
			func(t *testing.T, r [][]byte) { assertScalarNumeric(t, r) }},
		{"5.5", `SELECT MIN(StringToNumber(c.population)) AS minPop, MAX(StringToNumber(c.population)) AS maxPop FROM c WHERE StringToNumber(c.population) > 0`,
			func(t *testing.T, r [][]byte) {
				require.Len(t, r, 1)
				var obj map[string]any
				require.NoError(t, json.Unmarshal(r[0], &obj))
				minV, ok := obj["minPop"].(float64)
				require.True(t, ok)
				maxV, ok := obj["maxPop"].(float64)
				require.True(t, ok)
				assert.LessOrEqual(t, minV, maxV)
			}},
		{"11.2", `SELECT VALUE COUNT(1) FROM c`,
			func(t *testing.T, r [][]byte) { assertScalarNumeric(t, r) }},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			rows := runQuery(t, driver, container, tc.sql)
			tc.check(t, rows)
		})
	}
}

// ---------------------------------------------------------- Stage 2: DISTINCT

func TestIntegration_BYT9239_Stage2Distinct(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	t.Run("10.1_ObjectDistinct", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT DISTINCT c.country FROM c`)
		require.NotEmpty(t, rows)
		seen := map[string]bool{}
		for _, r := range rows {
			var obj map[string]any
			require.NoError(t, json.Unmarshal(r, &obj))
			country, ok := obj["country"].(string)
			require.True(t, ok, "row missing country: %s", r)
			assert.False(t, seen[country], "duplicate country: %s", country)
			seen[country] = true
		}
	})

	t.Run("10.2_DistinctValue", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT DISTINCT VALUE c.countryRegion FROM c`)
		require.NotEmpty(t, rows)
		seen := map[string]bool{}
		for _, r := range rows {
			var v string
			require.NoError(t, json.Unmarshal(r, &v))
			assert.False(t, seen[v], "duplicate: %s", v)
			seen[v] = true
		}
	})
}

// ---------------------------------------------------------- Stage 3: ORDER BY

func TestIntegration_BYT9239_Stage3OrderBy(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	t.Run("4.1_Ascending", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT c.name, c.population FROM c ORDER BY c.population ASC`)
		require.NotEmpty(t, rows)
		assertOrderedByPopulation(t, rows, true)
	})

	t.Run("4.2_Descending", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT c.name, c.population FROM c ORDER BY c.population DESC`)
		require.NotEmpty(t, rows)
		assertOrderedByPopulation(t, rows, false)
	})
}

// ---------------------------------------------------------- Stage 4: TOP

func TestIntegration_BYT9239_Stage4Top(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	rows := runQuery(t, driver, container, `SELECT TOP 10 * FROM c`)
	assert.Len(t, rows, 10, "TOP 10 must return exactly 10 rows")
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj), "row %d failed to parse: %s", i, r)
		_, hasCountry := obj["country"]
		assert.True(t, hasCountry, "row %d should be a full document: %s", i, r)
	}
}

// ---------------------------------------------------------- Stage 5: OFFSET/LIMIT

func TestIntegration_BYT9239_Stage5OffsetLimit(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	t.Run("12.1_OffsetZeroLimitTen", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT * FROM c ORDER BY c.name OFFSET 0 LIMIT 10`)
		require.Len(t, rows, 10)
		assertAscendingByName(t, rows)
	})

	t.Run("12.2_OffsetTenLimitTen", func(t *testing.T) {
		// Baseline: full ordered list.
		all := runQuery(t, driver, container, `SELECT * FROM c ORDER BY c.name`)
		require.NotEmpty(t, all)
		allNames := assertAscendingByName(t, all)

		rows := runQuery(t, driver, container, `SELECT * FROM c ORDER BY c.name OFFSET 10 LIMIT 10`)
		names := assertAscendingByName(t, rows)

		start := 10
		end := start + len(rows)
		require.LessOrEqual(t, end, len(allNames), "pagination window extends past available rows")
		assert.Equal(t, allNames[start:end], names,
			"OFFSET 10 LIMIT 10 must equal allNames[10:%d]", end)
	})
}

// ---------------------------------------------------------- Stage 6: GROUP BY

func TestIntegration_BYT9239_Stage6GroupBy(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	totalRows := runQuery(t, driver, container, `SELECT VALUE COUNT(1) FROM c`)
	require.Len(t, totalRows, 1)
	var total float64
	require.NoError(t, json.Unmarshal(totalRows[0], &total))

	t.Run("6.1_CountByCountry", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT c.country, COUNT(1) AS cityCount FROM c GROUP BY c.country`)
		require.NotEmpty(t, rows)
		var summed float64
		seen := map[string]bool{}
		for _, r := range rows {
			var obj map[string]any
			require.NoError(t, json.Unmarshal(r, &obj))
			country, ok := obj["country"].(string)
			require.True(t, ok, "country must be a string: %s", r)
			assert.False(t, seen[country], "duplicate country group: %s", country)
			seen[country] = true
			cnt, ok := obj["cityCount"].(float64)
			require.True(t, ok)
			summed += cnt
		}
		assert.Equal(t, total, summed, "sum of per-country counts must equal total")
	})

	t.Run("6.2_MultipleAggregates", func(t *testing.T) {
		rows := runQuery(t, driver, container, `SELECT c.countryRegion, COUNT(1) AS count, SUM(StringToNumber(c.population)) AS totalPop FROM c GROUP BY c.countryRegion`)
		require.NotEmpty(t, rows)
		var summed float64
		seen := map[string]bool{}
		for _, r := range rows {
			var obj map[string]any
			require.NoError(t, json.Unmarshal(r, &obj))
			region, ok := obj["countryRegion"].(string)
			require.True(t, ok, "countryRegion must be a string: %s", r)
			assert.False(t, seen[region], "duplicate region group: %s", region)
			seen[region] = true
			cnt, ok := obj["count"].(float64)
			require.True(t, ok)
			summed += cnt
			pop, ok := obj["totalPop"].(float64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, pop, float64(0))
		}
		assert.Equal(t, total, summed, "sum of per-region counts must equal total")
	})
}

// ---------------------------------------------------------- assertion helpers

func assertAliasNumeric(t *testing.T, rows [][]byte, alias string) {
	t.Helper()
	require.Len(t, rows, 1)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(rows[0], &obj))
	v, ok := obj[alias].(float64)
	require.True(t, ok, "alias %q must be numeric: %s", alias, rows[0])
	assert.NotZero(t, v, "alias %q must be non-zero", alias)
}

func assertScalarNumeric(t *testing.T, rows [][]byte) {
	t.Helper()
	require.Len(t, rows, 1)
	var n float64
	require.NoError(t, json.Unmarshal(rows[0], &n))
	assert.NotZero(t, n)
}

// assertOrderedByPopulation walks `rows` and asserts they are sorted by
// c.population under Cosmos item ordering (undefined < null < bool < number < string).
// ascending=true checks nondecreasing; ascending=false checks nonincreasing.
func assertOrderedByPopulation(t *testing.T, rows [][]byte, ascending bool) {
	t.Helper()
	var prevRank int
	var prevVal any
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		v, hasV := obj["population"]
		require.True(t, hasV, "row missing population: %s", r)
		rank, val := cosmosTypeRank(v)
		if i == 0 {
			prevRank, prevVal = rank, val
			continue
		}
		if rank != prevRank {
			if ascending {
				assert.GreaterOrEqual(t, rank, prevRank, "type-rank regressed at row %d: %s", i, r)
			} else {
				assert.LessOrEqual(t, rank, prevRank, "type-rank ascended at row %d: %s", i, r)
			}
			prevRank, prevVal = rank, val
			continue
		}
		switch pv := prevVal.(type) {
		case float64:
			cv, ok := val.(float64)
			require.True(t, ok, "row %d: expected float64 same-rank value, got %T", i, val)
			if ascending {
				assert.LessOrEqual(t, pv, cv, "ascending number order broken at row %d", i)
			} else {
				assert.GreaterOrEqual(t, pv, cv, "descending number order broken at row %d", i)
			}
		case string:
			cv, ok := val.(string)
			require.True(t, ok, "row %d: expected string same-rank value, got %T", i, val)
			if ascending {
				assert.LessOrEqual(t, pv, cv, "ascending string order broken at row %d", i)
			} else {
				assert.GreaterOrEqual(t, pv, cv, "descending string order broken at row %d", i)
			}
		default:
			// Other Cosmos types (bool, null, undefined) don't occur in our
			// WorldCities fixture for the population field — skip.
		}
		prevVal = val
	}
}

// assertAscendingByName is a shorthand for checking that rows are ordered by
// the `name` field in ascending order. Returns the extracted names.
func assertAscendingByName(t *testing.T, rows [][]byte) []string {
	t.Helper()
	names := make([]string, 0, len(rows))
	var prev string
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj), "row %d failed to parse: %s", i, r)
		name, ok := obj["name"].(string)
		require.True(t, ok, "row %d missing name: %s", i, r)
		names = append(names, name)
		if i > 0 {
			assert.LessOrEqual(t, prev, name, "ascending broken at row %d: prev=%q cur=%q", i, prev, name)
		}
		prev = name
	}
	return names
}

// cosmosTypeRank tags a JSON value with its Cosmos-ordering type class. Used
// to compare rows that straddle the number→string boundary: Cosmos orders
// undefined < null < bool < number < string.
func cosmosTypeRank(v any) (int, any) {
	switch x := v.(type) {
	case float64:
		return 3, x
	case string:
		return 4, x
	case bool:
		return 2, x
	case nil:
		return 1, x
	default:
		return 0, x
	}
}
