// Copyright (c) Bytebase (Hong Kong) Limited.
package cosmosdb

// Two suites cover the cross-partition query path: the gateway cannot serve an
// aggregate, DISTINCT, ORDER BY, TOP, OFFSET/LIMIT or GROUP BY that spans
// partitions, so QueryEveryPartition pivots to the pure-Go query engine in
// queryengine/gonative, which merges the per-partition streams itself.
//
// TestCrossPartition* run on the emulator SharedCosmosContainer starts, against
// a fixture whose contents are known, and so assert exact answers. Both suites
// are manual: this one starts a 4.9 GB emulator image that takes most of a
// minute to boot and publishes a fixed port, so only one can run on a machine.
//
//   AZURE_COSMOS_EMULATOR=1 go test -count=1 -run "^TestCrossPartition" \
//       -timeout 600s ./backend/plugin/db/cosmosdb/
//
// TestIntegration_BYT9239_* run the same query classes against a live Cosmos
// account, which the emulator cannot stand in for entirely, and assert only
// what holds for data they do not control. They are guarded by
// AZURE_COSMOS_KEY and skip when it is unset:
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
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"

	// Registers the CosmosDB query validator that QueryConn calls.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cosmosdb"
)

func TestMain(m *testing.M) { testcontainer.Main(m) }

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
	require.Equal(t, sql, results[0].Statement)

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

// ------------------------------------------- cross-partition, local emulator

const (
	crossPartitionDatabase  = "crosspartition"
	crossPartitionContainer = "cities"
	// The emulator hands a container one physical partition per 10k RU, so
	// throughput — not document count — is what makes these queries span
	// partitions and reach the merge the query engine exists for.
	crossPartitionThroughput       = 30000
	crossPartitionCitiesPerCountry = 5
	crossPartitionPopulationGap    = 100000
)

// crossPartitionCountries are the fixture's partition key values. Each holds
// crossPartitionCitiesPerCountry cities whose populations step by
// crossPartitionPopulationGap, so every aggregate has one correct answer.
var crossPartitionCountries = []string{"AE", "BR", "CN", "DE", "FR", "IN", "JP", "US"}

func crossPartitionItems() int { return len(crossPartitionCountries) * crossPartitionCitiesPerCountry }

// crossPartitionPopulationSum is the total the fixture's SUM must return.
func crossPartitionPopulationSum() int {
	perCountry := 0
	for i := 1; i <= crossPartitionCitiesPerCountry; i++ {
		perCountry += i * crossPartitionPopulationGap
	}
	return perCountry * len(crossPartitionCountries)
}

type crossPartitionCity struct {
	ID         string `json:"id"`
	Country    string `json:"country"`
	Name       string `json:"name"`
	Population int    `json:"population"`
}

var (
	crossPartitionOnce   sync.Once
	crossPartitionDriver *Driver
	crossPartitionRanges int
	crossPartitionErr    error
)

// crossPartitionEnv opts into the cross-partition suite. Gating it here rather
// than in each test is what keeps the emulator from starting at all on a run
// that did not ask for it, since SharedCosmosContainer starts lazily.
const crossPartitionEnv = "AZURE_COSMOS_EMULATOR"

// emulatorFixture returns a driver pointed at the seeded fixture, building it
// on first use. It refuses a fixture confined to one physical partition: the
// gateway serves those queries on its own, so the tests below would pass
// without the query engine ever running.
func emulatorFixture(t *testing.T) *Driver {
	t.Helper()
	if os.Getenv(crossPartitionEnv) != "1" {
		t.Skipf("set %s=1 to run the cross-partition tests against a local Cosmos emulator", crossPartitionEnv)
	}
	container := testcontainer.SharedCosmosContainer(t)
	endpoint := fmt.Sprintf("https://%s:%s", container.GetHost(), container.GetPort())
	crossPartitionOnce.Do(func() {
		crossPartitionDriver, crossPartitionRanges, crossPartitionErr = seedCrossPartition(endpoint)
	})
	require.NoError(t, crossPartitionErr)
	require.Greater(t, crossPartitionRanges, 1,
		"fixture spans %d physical partition(s); cross-partition queries need more than one", crossPartitionRanges)
	return crossPartitionDriver
}

func seedCrossPartition(endpoint string) (*Driver, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	opened, err := (&Driver{}).Open(ctx, storepb.Engine_COSMOSDB, db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Host: endpoint},
		ConnectionContext: db.ConnectionContext{DatabaseName: crossPartitionDatabase},
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "open the driver against the emulator")
	}
	driver, ok := opened.(*Driver)
	if !ok {
		return nil, 0, errors.Errorf("Open returned %T, want *Driver", opened)
	}
	if err := driver.Ping(ctx); err != nil {
		return nil, 0, errors.Wrap(err, "ping the emulator")
	}

	if _, err := driver.client.CreateDatabase(ctx, azcosmos.DatabaseProperties{ID: crossPartitionDatabase}, nil); err != nil && !alreadyExists(err) {
		return nil, 0, errors.Wrap(err, "create the fixture database")
	}
	database, err := driver.client.NewDatabase(crossPartitionDatabase)
	if err != nil {
		return nil, 0, errors.Wrap(err, "open the fixture database")
	}
	throughput := azcosmos.NewManualThroughputProperties(crossPartitionThroughput)
	if _, err := database.CreateContainer(ctx, azcosmos.ContainerProperties{
		ID:                     crossPartitionContainer,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{Paths: []string{"/country"}},
	}, &azcosmos.CreateContainerOptions{ThroughputProperties: &throughput}); err != nil && !alreadyExists(err) {
		return nil, 0, errors.Wrap(err, "create the fixture container")
	}
	container, err := database.NewContainer(crossPartitionContainer)
	if err != nil {
		return nil, 0, errors.Wrap(err, "open the fixture container")
	}

	for _, country := range crossPartitionCountries {
		for i := range crossPartitionCitiesPerCountry {
			item, err := json.Marshal(crossPartitionCity{
				ID:         fmt.Sprintf("%s-%d", country, i),
				Country:    country,
				Name:       fmt.Sprintf("city-%s-%d", country, i),
				Population: (i + 1) * crossPartitionPopulationGap,
			})
			if err != nil {
				return nil, 0, errors.Wrap(err, "marshal a fixture city")
			}
			if _, err := container.CreateItem(ctx, azcosmos.NewPartitionKeyString(country), item, nil); err != nil && !alreadyExists(err) {
				return nil, 0, errors.Wrapf(err, "seed %s-%d", country, i)
			}
		}
	}

	ranges, err := countPartitionKeyRanges(endpoint)
	if err != nil {
		return nil, 0, err
	}
	return driver, ranges, nil
}

// countPartitionKeyRanges reports how many physical partitions the fixture
// container spans. It reads the REST endpoint because the SDK keeps the range
// list to itself, inside the query path under test.
func countPartitionKeyRanges(endpoint string) (int, error) {
	client, err := newEmulatorRESTClient(endpoint)
	if err != nil {
		return 0, errors.Wrap(err, "build the emulator REST client")
	}
	link := fmt.Sprintf("dbs/%s/colls/%s", crossPartitionDatabase, crossPartitionContainer)
	body, err := client.get(link+"/pkranges", "pkranges", link)
	if err != nil {
		return 0, errors.Wrap(err, "read the container's partition key ranges")
	}
	var resp struct {
		PartitionKeyRanges []struct {
			ID string `json:"id"`
		} `json:"PartitionKeyRanges"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, errors.Wrap(err, "parse the partition key range list")
	}
	return len(resp.PartitionKeyRanges), nil
}

func alreadyExists(err error) bool {
	var responseErr *azcore.ResponseError
	return errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusConflict
}

func TestCrossPartitionAggregates(t *testing.T) {
	driver := emulatorFixture(t)
	items := crossPartitionItems()
	sum := crossPartitionPopulationSum()

	for _, tc := range []struct {
		name string
		sql  string
		want string
	}{
		{"count_value", `SELECT VALUE COUNT(1) FROM c`, fmt.Sprint(items)},
		{"count_aliased", `SELECT COUNT(1) AS total FROM c`, fmt.Sprintf(`{"total":%d}`, items)},
		{"sum_value", `SELECT VALUE SUM(c.population) FROM c`, fmt.Sprint(sum)},
		{"avg_value", `SELECT VALUE AVG(c.population) FROM c`, fmt.Sprint(sum / items)},
		{"min_max_aliased", `SELECT MIN(c.population) AS lo, MAX(c.population) AS hi FROM c`,
			fmt.Sprintf(`{"lo":%d,"hi":%d}`, crossPartitionPopulationGap, crossPartitionCitiesPerCountry*crossPartitionPopulationGap)},
		{"count_and_sum_aliased", `SELECT COUNT(1) AS total, SUM(c.population) AS pop FROM c`,
			fmt.Sprintf(`{"total":%d,"pop":%d}`, items, sum)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := runQuery(t, driver, crossPartitionContainer, tc.sql)
			require.Len(t, rows, 1)
			assert.JSONEq(t, tc.want, string(rows[0]))
		})
	}
}

func TestCrossPartitionDistinct(t *testing.T) {
	driver := emulatorFixture(t)

	t.Run("value_form", func(t *testing.T) {
		rows := runQuery(t, driver, crossPartitionContainer, `SELECT DISTINCT VALUE c.country FROM c`)
		got := make([]string, 0, len(rows))
		for _, row := range rows {
			var country string
			require.NoError(t, json.Unmarshal(row, &country))
			got = append(got, country)
		}
		assert.ElementsMatch(t, crossPartitionCountries, got)
	})

	t.Run("object_form", func(t *testing.T) {
		rows := runQuery(t, driver, crossPartitionContainer, `SELECT DISTINCT c.country FROM c`)
		assert.Len(t, rows, len(crossPartitionCountries))
	})
}

func TestCrossPartitionOrderBy(t *testing.T) {
	driver := emulatorFixture(t)

	t.Run("ascending", func(t *testing.T) {
		rows := runQuery(t, driver, crossPartitionContainer, `SELECT c.name, c.population FROM c ORDER BY c.population ASC`)
		require.Len(t, rows, crossPartitionItems())
		assertOrderedByPopulation(t, rows, true)
	})

	t.Run("descending", func(t *testing.T) {
		rows := runQuery(t, driver, crossPartitionContainer, `SELECT c.name, c.population FROM c ORDER BY c.population DESC`)
		require.Len(t, rows, crossPartitionItems())
		assertOrderedByPopulation(t, rows, false)
	})
}

func TestCrossPartitionTop(t *testing.T) {
	driver := emulatorFixture(t)
	rows := runQuery(t, driver, crossPartitionContainer, `SELECT TOP 3 * FROM c`)
	assert.Len(t, rows, 3)
}

// TestCrossPartitionOffsetLimit pins the page against the globally ordered
// result: a merge that drops or reorders a partition's stream still returns
// five rows, so only comparing the slice catches it.
func TestCrossPartitionOffsetLimit(t *testing.T) {
	driver := emulatorFixture(t)

	all := assertAscendingByName(t, runQuery(t, driver, crossPartitionContainer, `SELECT c.name FROM c ORDER BY c.name`))
	require.Len(t, all, crossPartitionItems())

	page := assertAscendingByName(t, runQuery(t, driver, crossPartitionContainer, `SELECT c.name FROM c ORDER BY c.name OFFSET 5 LIMIT 5`))
	assert.Equal(t, all[5:10], page)
}

func TestCrossPartitionGroupBy(t *testing.T) {
	driver := emulatorFixture(t)
	rows := runQuery(t, driver, crossPartitionContainer,
		`SELECT c.country, COUNT(1) AS cities, SUM(c.population) AS pop FROM c GROUP BY c.country`)
	require.Len(t, rows, len(crossPartitionCountries))

	wantPop := float64(crossPartitionPopulationSum() / len(crossPartitionCountries))
	seen := map[string]bool{}
	for _, row := range rows {
		var group struct {
			Country string  `json:"country"`
			Cities  float64 `json:"cities"`
			Pop     float64 `json:"pop"`
		}
		require.NoError(t, json.Unmarshal(row, &group))
		assert.False(t, seen[group.Country], "country %q grouped twice", group.Country)
		seen[group.Country] = true
		assert.Equal(t, float64(crossPartitionCitiesPerCountry), group.Cities, "country %q", group.Country)
		assert.Equal(t, wantPop, group.Pop, "country %q", group.Country)
	}
	assert.Len(t, seen, len(crossPartitionCountries))
}

// TestCrossPartitionGatewayServable covers the queries that never leave the
// gateway path, which the engine must not take over: a projection of every
// partition, and a filter the gateway can route to a single one.
func TestCrossPartitionGatewayServable(t *testing.T) {
	driver := emulatorFixture(t)

	t.Run("projection", func(t *testing.T) {
		rows := runQuery(t, driver, crossPartitionContainer, `SELECT * FROM c`)
		assert.Len(t, rows, crossPartitionItems())
	})

	t.Run("single_partition_filter", func(t *testing.T) {
		rows := runQuery(t, driver, crossPartitionContainer, `SELECT * FROM c WHERE c.country = "US"`)
		assert.Len(t, rows, crossPartitionCitiesPerCountry)
	})
}

// TestCrossPartitionQueryLimit covers QueryConn's own row cap, which stops
// paging mid-result and so decides how much of a cross-partition query the
// driver pulls back.
func TestCrossPartitionQueryLimit(t *testing.T) {
	driver := emulatorFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results, err := driver.QueryConn(ctx, nil, `SELECT * FROM c`, db.QueryContext{
		Container: crossPartitionContainer,
		Limit:     7,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Rows, 7)
	assert.EqualValues(t, 7, results[0].RowsCount)
}
