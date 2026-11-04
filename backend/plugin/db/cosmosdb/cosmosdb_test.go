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
	"slices"
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
			if err := seedItem(ctx, container, country, item); err != nil {
				return nil, 0, errors.Wrapf(err, "seed %s-%d", country, i)
			}
		}
	}

	ranges, err := countContainerRanges(endpoint, crossPartitionContainer)
	if err != nil {
		return nil, 0, err
	}
	return driver, ranges, nil
}

// countContainerRanges reports how many physical partitions a fixture
// container spans. It reads the REST endpoint because the SDK keeps the range
// list to itself, inside the query path under test.
func countContainerRanges(endpoint, container string) (int, error) {
	client, err := newEmulatorRESTClient(endpoint)
	if err != nil {
		return 0, errors.Wrap(err, "build the emulator REST client")
	}
	link := fmt.Sprintf("dbs/%s/colls/%s", crossPartitionDatabase, container)
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

// seedItem writes one fixture item, retrying what a loaded emulator does to a
// write it cannot serve in time: a 408, a 429, a 503, or a dropped connection
// that surfaces as an EOF. The emulator is slow enough under this repo's own
// test load that an unretried seeder fails the whole suite on one dropped
// write, with an error that reads like a query-engine bug. An error that is
// not transient costs the attempts and then surfaces unchanged.
func seedItem(ctx context.Context, container *azcosmos.ContainerClient, partitionKey string, item []byte) error {
	var err error
	for attempt := range 6 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
		if _, err = container.CreateItem(ctx, azcosmos.NewPartitionKeyString(partitionKey), item, nil); err == nil || alreadyExists(err) {
			return nil
		}
	}
	return err
}

func alreadyExists(err error) bool {
	var responseErr *azcore.ResponseError
	return errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusConflict
}

// ------------------------------------------ multi-page fixture, local emulator

const (
	// pagedContainer is a second fixture whose ranges answer over several
	// pages. The gateway caps a page at pagedPageSize items when the request
	// names no page size, which is what the SDK's engine requests do: it
	// forwards a continuation token and the range's partition key range id,
	// and nothing else. A query engine that stops at the first page therefore
	// loses every row past the cap, which is what this fixture catches.
	pagedContainer = "paged"
	// The emulator hands a container one physical partition per 6k RU or so,
	// and the cities fixture already spends most of its budget, so this one
	// asks for the least that still spans more than one.
	pagedThroughput = 12000
	// pagedPageSize is the gateway's own per-page item cap, measured both on
	// the emulator and on a real Cosmos account.
	pagedPageSize = 1000
	// pagedBigRows puts one partition key value past the cap, so the range
	// holding it must be read over several pages.
	pagedBigRows = pagedPageSize + 50
	// pagedSmallRows keeps the other partition key values short, so a merge
	// has ranges that drain while the big ones still have pages to come.
	pagedSmallRows = 5
)

// pagedBuckets are the fixture's partition key values: a partition key value
// lives in exactly one physical partition, so the row count per value is what
// decides which ranges have to page.
var (
	pagedBigBuckets   = []string{"big-1", "big-2"}
	pagedSmallBuckets = []string{"small-1", "small-2", "small-3", "small-4", "small-5", "small-6"}
)

// pagedRows is the fixture's row count. Every row carries a distinct seq, from
// 0 to pagedRows-1, so an ordered result has exactly one correct answer and
// DISTINCT and GROUP BY over seq have exactly pagedRows outputs.
func pagedRows() int {
	return len(pagedBigBuckets)*pagedBigRows + len(pagedSmallBuckets)*pagedSmallRows
}

// pagedShapes gives eleven of the fixture's rows a `v`, one per shape a
// document can hold, chosen so that the values span the container's ranges:
// two rows in each big bucket and one in each small one. Every other row
// leaves `v` undefined, so a query grouping or ordering on it meets both.
//
// Ordering them is what the service does by type — undefined < null < bool <
// number < string < array < object — and the two seqs inside each of the array
// and object classes are interchangeable, since the service orders those by
// its index's own encoding.
var pagedShapes = []struct {
	seq   int
	value string
}{
	{seq: 0, value: `null`},
	{seq: 1, value: `false`},
	{seq: pagedBigRows, value: `true`},
	{seq: pagedBigRows + 1, value: `1`},
	{seq: 2 * pagedBigRows, value: `20`},
	{seq: 2*pagedBigRows + 1, value: `"a"`},
	{seq: 2*pagedBigRows + pagedSmallRows, value: `"b"`},
	{seq: 2*pagedBigRows + 2*pagedSmallRows, value: `[1,2]`},
	{seq: 2*pagedBigRows + 3*pagedSmallRows, value: `[2]`},
	{seq: 2*pagedBigRows + 4*pagedSmallRows, value: `{"a":1}`},
	{seq: 2*pagedBigRows + 5*pagedSmallRows, value: `{"b":1}`},
}

// pagedShapeOf returns the shape attached to a row, or "" when the row leaves
// `v` undefined.
func pagedShapeOf(seq int) string {
	for _, shape := range pagedShapes {
		if shape.seq == seq {
			return shape.value
		}
	}
	return ""
}

var (
	pagedOnce            sync.Once
	pagedContainerClient *azcosmos.ContainerClient
	pagedRanges          int
	pagedErr             error
)

// pagedFixture returns the seeded multi-page container, building it on first
// use. It refuses a fixture confined to one physical partition for the same
// reason emulatorFixture does: the gateway serves those queries itself.
func pagedFixture(t *testing.T) *azcosmos.ContainerClient {
	t.Helper()
	driver := emulatorFixture(t)
	pagedOnce.Do(func() {
		pagedContainerClient, pagedRanges, pagedErr = seedPaged(driver)
	})
	require.NoError(t, pagedErr)
	require.Greater(t, pagedRanges, 1,
		"fixture spans %d physical partition(s); cross-partition queries need more than one", pagedRanges)
	return pagedContainerClient
}

func seedPaged(driver *Driver) (*azcosmos.ContainerClient, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	database, err := driver.client.NewDatabase(crossPartitionDatabase)
	if err != nil {
		return nil, 0, errors.Wrap(err, "open the fixture database")
	}
	throughput := azcosmos.NewManualThroughputProperties(pagedThroughput)
	if _, err := database.CreateContainer(ctx, azcosmos.ContainerProperties{
		ID:                     pagedContainer,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{Paths: []string{"/bucket"}},
	}, &azcosmos.CreateContainerOptions{ThroughputProperties: &throughput}); err != nil && !alreadyExists(err) {
		return nil, 0, errors.Wrap(err, "create the multi-page container")
	}
	container, err := database.NewContainer(pagedContainer)
	if err != nil {
		return nil, 0, errors.Wrap(err, "open the multi-page container")
	}

	// A few thousand single writes is most of this suite's runtime, so they go
	// out concurrently. The worker count is what the emulator answered without
	// timing out under the load of the rest of this machine.
	type row struct {
		bucket string
		seq    int
	}
	rows := make(chan row, pagedRows())
	seq := 0
	for _, bucket := range pagedBigBuckets {
		for range pagedBigRows {
			rows <- row{bucket: bucket, seq: seq}
			seq++
		}
	}
	for _, bucket := range pagedSmallBuckets {
		for range pagedSmallRows {
			rows <- row{bucket: bucket, seq: seq}
			seq++
		}
	}
	close(rows)

	var wg sync.WaitGroup
	errs := make(chan error, 1)
	for range 12 {
		wg.Go(func() {
			for r := range rows {
				row := pagedRow{
					ID:     fmt.Sprintf("row-%d", r.seq),
					Bucket: r.bucket,
					Seq:    r.seq,
				}
				if shape := pagedShapeOf(r.seq); shape != "" {
					row.V = json.RawMessage(shape)
				}
				item, err := json.Marshal(row)
				if err == nil {
					err = seedItem(ctx, container, r.bucket, item)
				}
				if err != nil {
					select {
					case errs <- errors.Wrapf(err, "seed row %d", r.seq):
					default:
					}
					return
				}
			}
		})
	}
	wg.Wait()
	select {
	case err := <-errs:
		return nil, 0, err
	default:
	}

	ranges, err := countContainerRanges(driver.connCfg.DataSource.GetHost(), pagedContainer)
	if err != nil {
		return nil, 0, err
	}
	return container, ranges, nil
}

type pagedRow struct {
	ID     string `json:"id"`
	Bucket string `json:"bucket"`
	Seq    int    `json:"seq"`
	// V carries one of pagedShapes on the rows that have one, and is absent
	// from every other document — which is what makes the fixture answer for
	// a column a schemaless store defines on only some of its rows.
	V json.RawMessage `json:"v,omitempty"`
}

// pagedSeqs runs a query against the multi-page fixture and returns the seq of
// every row it emitted, in the order the engine emitted them.
func pagedSeqs(t *testing.T, container *azcosmos.ContainerClient, sql string) []int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	items, err := QueryEveryPartition(ctx, container, sql, nil, 0)
	require.NoError(t, err, "QueryEveryPartition: %s", sql)
	seqs := make([]int, 0, len(items))
	for _, item := range items {
		var row struct {
			Seq *int `json:"seq"`
		}
		require.NoError(t, json.Unmarshal(item, &row), "row: %s", item)
		require.NotNil(t, row.Seq, "row missing seq: %s", item)
		seqs = append(seqs, *row.Seq)
	}
	return seqs
}

// ascendingSeqs is the fixture's seq values in order: every query below either
// returns this sequence or a slice of it.
func ascendingSeqs(from, to int) []int {
	seqs := make([]int, 0, to-from)
	for seq := from; seq < to; seq++ {
		seqs = append(seqs, seq)
	}
	return seqs
}

// TestCrossPartitionMultiPage covers the query classes whose results a range
// can only deliver over several pages. Each one reads past the gateway's
// per-page cap, which an engine that treats a continuation token as the end of
// a range cannot do: before the engine followed them, an unbounded ORDER BY
// failed outright on any container holding more than pagedPageSize rows in one
// range, and so did DISTINCT and GROUP BY over that many values and a TOP
// above the cap.
func TestCrossPartitionMultiPage(t *testing.T) {
	container := pagedFixture(t)
	rows := pagedRows()

	t.Run("order_by_unbounded", func(t *testing.T) {
		assert.Equal(t, ascendingSeqs(0, rows),
			pagedSeqs(t, container, `SELECT c.seq FROM c ORDER BY c.seq ASC`))
	})

	t.Run("order_by_descending", func(t *testing.T) {
		want := ascendingSeqs(0, rows)
		slices.Reverse(want)
		assert.Equal(t, want, pagedSeqs(t, container, `SELECT c.seq FROM c ORDER BY c.seq DESC`))
	})

	// A page of a range ends mid-result, so the rows either side of the cap
	// are the ones a resumed range can misplace.
	t.Run("offset_limit_across_a_page_boundary", func(t *testing.T) {
		assert.Equal(t, ascendingSeqs(pagedPageSize-5, pagedPageSize+5),
			pagedSeqs(t, container, fmt.Sprintf(
				`SELECT c.seq FROM c ORDER BY c.seq ASC OFFSET %d LIMIT 10`, pagedPageSize-5)))
	})

	t.Run("distinct_high_cardinality", func(t *testing.T) {
		seqs := pagedSeqs(t, container, `SELECT DISTINCT c.seq FROM c`)
		slices.Sort(seqs)
		assert.Equal(t, ascendingSeqs(0, rows), seqs)
	})

	t.Run("group_by_many_groups", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		items, err := QueryEveryPartition(ctx, container,
			`SELECT c.seq, COUNT(1) AS n FROM c GROUP BY c.seq`, nil, 0)
		require.NoError(t, err)
		seqs := make([]int, 0, len(items))
		for _, item := range items {
			var group struct {
				Seq int `json:"seq"`
				N   int `json:"n"`
			}
			require.NoError(t, json.Unmarshal(item, &group))
			assert.Equal(t, 1, group.N, "seq %d", group.Seq)
			seqs = append(seqs, group.Seq)
		}
		slices.Sort(seqs)
		assert.Equal(t, ascendingSeqs(0, rows), seqs)
	})

	t.Run("top_above_the_page_cap", func(t *testing.T) {
		assert.Len(t, pagedSeqs(t, container,
			fmt.Sprintf(`SELECT TOP %d c.seq FROM c`, pagedPageSize+25)), pagedPageSize+25)
	})

	// An aggregate answers with one partial document per range however many
	// rows that range holds, which is why aggregates were the one class this
	// fixture's size never broke. Kept as the control case.
	t.Run("aggregate_over_the_whole_fixture", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		items, err := QueryEveryPartition(ctx, container,
			`SELECT COUNT(1) AS n, SUM(c.seq) AS total FROM c`, nil, 0)
		require.NoError(t, err)
		require.Len(t, items, 1)
		want := 0
		for _, seq := range ascendingSeqs(0, rows) {
			want += seq
		}
		assert.JSONEq(t, fmt.Sprintf(`{"n":%d,"total":%d}`, rows, want), string(items[0]))
	})
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

// ------------------------------------ composed operators, local emulator
//
// The query classes in BYT-10397: each composes two operators the gateway
// reports in one plan, and the engine used to honour only one of them. Every
// case here runs on the multi-page fixture, because two of them (TOP with
// ORDER BY, and DISTINCT with a LIMIT) answer correctly on a single physical
// partition however the engine is broken.

// pagedQuery runs a query against the multi-page fixture and returns its rows
// as raw JSON, in the order the engine emitted them.
func pagedQuery(t *testing.T, container *azcosmos.ContainerClient, sql string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	items, err := QueryEveryPartition(ctx, container, sql, nil, 0)
	require.NoError(t, err, "QueryEveryPartition: %s", sql)
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, string(item))
	}
	return rows
}

// pagedBucketValues reads the bucket out of every row, for both spellings of
// a projection: `{"bucket":"big-1"}` and the VALUE form's bare `"big-1"`.
func pagedBucketValues(t *testing.T, rows []string) []string {
	t.Helper()
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		var object struct {
			Bucket *string `json:"bucket"`
		}
		if err := json.Unmarshal([]byte(row), &object); err == nil && object.Bucket != nil {
			out = append(out, *object.Bucket)
			continue
		}
		var scalar string
		require.NoError(t, json.Unmarshal([]byte(row), &scalar), "row: %s", row)
		out = append(out, scalar)
	}
	return out
}

// sortedPagedBuckets is the fixture's partition key values in ascending order.
func sortedPagedBuckets() []string {
	buckets := append(append([]string(nil), pagedBigBuckets...), pagedSmallBuckets...)
	slices.Sort(buckets)
	return buckets
}

// pagedBucketRows is how many rows each partition key value holds.
func pagedBucketRows(bucket string) int {
	if slices.Contains(pagedBigBuckets, bucket) {
		return pagedBigRows
	}
	return pagedSmallRows
}

// TestCrossPartitionDistinctComposed covers DISTINCT alongside an ORDER BY or
// a page. The service reports distinctType "Unordered" for the object form
// even under ORDER BY, so the engine dropped the DISTINCT and returned every
// row in the container; a paged DISTINCT ran the user's own query per range,
// so each range applied the LIMIT to itself before anything was deduplicated.
func TestCrossPartitionDistinctComposed(t *testing.T) {
	container := pagedFixture(t)
	buckets := sortedPagedBuckets()

	t.Run("order_by_object_form", func(t *testing.T) {
		assert.Equal(t, buckets, pagedBucketValues(t,
			pagedQuery(t, container, `SELECT DISTINCT c.bucket FROM c ORDER BY c.bucket`)))
	})

	t.Run("order_by_value_form", func(t *testing.T) {
		assert.Equal(t, buckets, pagedBucketValues(t,
			pagedQuery(t, container, `SELECT DISTINCT VALUE c.bucket FROM c ORDER BY c.bucket`)))
	})

	t.Run("order_by_descending", func(t *testing.T) {
		want := sortedPagedBuckets()
		slices.Reverse(want)
		assert.Equal(t, want, pagedBucketValues(t,
			pagedQuery(t, container, `SELECT DISTINCT c.bucket FROM c ORDER BY c.bucket DESC`)))
	})

	// An ordered DISTINCT pins the page against exact values; the unordered
	// forms below can only be held to the count, since Cosmos defines no
	// order for them.
	t.Run("order_by_with_a_page", func(t *testing.T) {
		assert.Equal(t, buckets[2:5], pagedBucketValues(t,
			pagedQuery(t, container, `SELECT DISTINCT c.bucket FROM c ORDER BY c.bucket OFFSET 2 LIMIT 3`)))
	})

	t.Run("limit_value_form", func(t *testing.T) {
		got := pagedBucketValues(t,
			pagedQuery(t, container, `SELECT DISTINCT VALUE c.bucket FROM c OFFSET 0 LIMIT 3`))
		assert.Len(t, got, 3)
		assert.Subset(t, buckets, got)
		assert.Len(t, slices.Compact(slices.Sorted(slices.Values(got))), 3, "values must be distinct")
	})

	t.Run("offset_limit_object_form", func(t *testing.T) {
		got := pagedBucketValues(t,
			pagedQuery(t, container, `SELECT DISTINCT c.bucket FROM c OFFSET 2 LIMIT 3`))
		assert.Len(t, got, 3)
		assert.Subset(t, buckets, got)
		assert.Len(t, slices.Compact(slices.Sorted(slices.Values(got))), 3, "values must be distinct")
	})
}

// TestCrossPartitionTopWithOrderBy covers TOP alongside an ORDER BY. The
// rewritten per-range query carries the TOP, so the merge saw n rows from
// every range and emitted all of them: n x partitions rows, in order.
func TestCrossPartitionTopWithOrderBy(t *testing.T) {
	container := pagedFixture(t)
	rows := pagedRows()

	t.Run("ascending", func(t *testing.T) {
		assert.Equal(t, ascendingSeqs(0, 5),
			pagedSeqs(t, container, `SELECT TOP 5 c.seq FROM c ORDER BY c.seq ASC`))
	})

	t.Run("descending", func(t *testing.T) {
		want := ascendingSeqs(rows-5, rows)
		slices.Reverse(want)
		assert.Equal(t, want, pagedSeqs(t, container, `SELECT TOP 5 c.seq FROM c ORDER BY c.seq DESC`))
	})

	// A TOP past the gateway's page cap has to keep reading the ranges after
	// the cap, and still stop at n.
	t.Run("above_the_page_cap", func(t *testing.T) {
		assert.Equal(t, ascendingSeqs(0, pagedPageSize+25),
			pagedSeqs(t, container, fmt.Sprintf(
				`SELECT TOP %d c.seq FROM c ORDER BY c.seq ASC`, pagedPageSize+25)))
	})
}

// TestCrossPartitionGroupByOrder covers the order GROUP BY rows come back in.
// The pipeline emitted them straight out of a Go map, so the same query
// answered in a different order every run, and a GROUP BY the user ordered
// reaches the engine as a plan with no order fields at all — the ORDER BY
// survives only inside the rewritten query.
func TestCrossPartitionGroupByOrder(t *testing.T) {
	container := pagedFixture(t)

	type group struct {
		Bucket string `json:"bucket"`
		N      int    `json:"n"`
	}
	groupsOf := func(t *testing.T, rows []string) []group {
		t.Helper()
		out := make([]group, 0, len(rows))
		for _, row := range rows {
			var g group
			require.NoError(t, json.Unmarshal([]byte(row), &g), "row: %s", row)
			out = append(out, g)
		}
		return out
	}
	want := make([]group, 0, len(sortedPagedBuckets()))
	for _, bucket := range sortedPagedBuckets() {
		want = append(want, group{Bucket: bucket, N: pagedBucketRows(bucket)})
	}

	t.Run("order_by_the_group_key", func(t *testing.T) {
		assert.Equal(t, want, groupsOf(t, pagedQuery(t, container,
			`SELECT c.bucket, COUNT(1) AS n FROM c GROUP BY c.bucket ORDER BY c.bucket`)))
	})

	t.Run("order_by_descending", func(t *testing.T) {
		reversed := slices.Clone(want)
		slices.Reverse(reversed)
		assert.Equal(t, reversed, groupsOf(t, pagedQuery(t, container,
			`SELECT c.bucket, COUNT(1) AS n FROM c GROUP BY c.bucket ORDER BY c.bucket DESC`)))
	})

	// Without an ORDER BY the order is Cosmos's to choose, but it must be the
	// same on every run.
	t.Run("unordered_is_stable_across_runs", func(t *testing.T) {
		const sql = `SELECT c.bucket, COUNT(1) AS n FROM c GROUP BY c.bucket`
		first := groupsOf(t, pagedQuery(t, container, sql))
		assert.ElementsMatch(t, want, first)
		for range 2 {
			assert.Equal(t, first, groupsOf(t, pagedQuery(t, container, sql)))
		}
	})
}

// TestCrossPartitionEmptyAggregates covers an aggregate whose filter matches
// no row. MIN and MAX failed the query outright, because an empty partition's
// partial is an absent one; the aliased form answered `{"m":null}` where the
// gateway omits the alias. COUNT and SUM really are 0 over an empty set and
// are here as controls.
func TestCrossPartitionEmptyAggregates(t *testing.T) {
	container := pagedFixture(t)
	const noRows = `WHERE c.seq > 99999999`

	for _, tc := range []struct {
		name string
		sql  string
		want []string
	}{
		{"min_value", `SELECT VALUE MIN(c.seq) FROM c ` + noRows, nil},
		{"max_value", `SELECT VALUE MAX(c.seq) FROM c ` + noRows, nil},
		{"avg_value", `SELECT VALUE AVG(c.seq) FROM c ` + noRows, nil},
		{"min_aliased", `SELECT MIN(c.seq) AS m FROM c ` + noRows, []string{`{}`}},
		{"min_max_aliased", `SELECT MIN(c.seq) AS lo, MAX(c.seq) AS hi FROM c ` + noRows, []string{`{}`}},
		{"count_value", `SELECT VALUE COUNT(1) FROM c ` + noRows, []string{`0`}},
		{"count_aliased", `SELECT COUNT(1) AS n FROM c ` + noRows, []string{`{"n":0}`}},
		{"sum_value", `SELECT VALUE SUM(c.seq) FROM c ` + noRows, []string{`0`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := pagedQuery(t, container, tc.sql)
			require.Len(t, got, len(tc.want))
			for i := range tc.want {
				assert.JSONEq(t, tc.want[i], got[i])
			}
		})
	}

	// The same MIN and MAX over rows that do match: the VALUE form reads its
	// partial from item2, which is where the count lives.
	t.Run("min_max_over_rows_that_match", func(t *testing.T) {
		assert.Equal(t, []string{`0`}, pagedQuery(t, container, `SELECT VALUE MIN(c.seq) FROM c`))
		assert.Equal(t, []string{fmt.Sprint(pagedRows() - 1)},
			pagedQuery(t, container, `SELECT VALUE MAX(c.seq) FROM c`))
	})
}

// TestCrossPartitionOffsetLimitWithoutOrderBy covers ordinary result paging,
// which the engine rejected although it advertises OffsetAndLimit. Cosmos
// defines no order for it, so only the count and the rows' provenance hold.
func TestCrossPartitionOffsetLimitWithoutOrderBy(t *testing.T) {
	container := pagedFixture(t)

	seqs := pagedSeqs(t, container, `SELECT c.seq FROM c OFFSET 2 LIMIT 3`)
	require.Len(t, seqs, 3)
	assert.Len(t, slices.Compact(slices.Sorted(slices.Values(seqs))), 3, "rows must not repeat")
	for _, seq := range seqs {
		assert.Less(t, seq, pagedRows())
		assert.GreaterOrEqual(t, seq, 0)
	}

	// A window wider than the gateway's page cap reads several pages of a
	// range before it is full.
	wide := pagedSeqs(t, container, fmt.Sprintf(`SELECT c.seq FROM c OFFSET 0 LIMIT %d`, pagedPageSize+25))
	assert.Len(t, wide, pagedPageSize+25)
}

// TestCrossPartitionAggregateWithOffsetLimit covers a page applied to an
// aggregate. The query produces exactly one row, so OFFSET 1 skips past it and
// the gateway answers with nothing; the engine returned the row regardless.
func TestCrossPartitionAggregateWithOffsetLimit(t *testing.T) {
	container := pagedFixture(t)

	assert.Empty(t, pagedQuery(t, container, `SELECT VALUE COUNT(1) FROM c OFFSET 1 LIMIT 1`))
	assert.Equal(t, []string{fmt.Sprint(pagedRows())},
		pagedQuery(t, container, `SELECT VALUE COUNT(1) FROM c OFFSET 0 LIMIT 1`))
	assert.Empty(t, pagedQuery(t, container, `SELECT COUNT(1) AS n FROM c OFFSET 1 LIMIT 1`))
}

// TestCrossPartitionDistinctWithOrderByThroughTheDriver runs the worst of
// these through QueryConn, the entry point the SQL editor uses, so the SQL
// validator is in the path too.
func TestCrossPartitionDistinctWithOrderByThroughTheDriver(t *testing.T) {
	driver := emulatorFixture(t)
	rows := runQuery(t, driver, crossPartitionContainer, `SELECT DISTINCT c.country FROM c ORDER BY c.country`)
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		var object struct {
			Country string `json:"country"`
		}
		require.NoError(t, json.Unmarshal(row, &object))
		got = append(got, object.Country)
	}
	assert.Equal(t, slices.Sorted(slices.Values(crossPartitionCountries)), got)
}

// TestCrossPartitionSparseGroupBy groups on a column the documents do not all
// define, which is ordinary in a schemaless store. The service answers the
// undefined rows as a group of their own, counted apart from an explicit null
// and spelled with the alias left out of the row.
func TestCrossPartitionSparseGroupBy(t *testing.T) {
	container := pagedFixture(t)

	want := []string{fmt.Sprintf(`{"n":%d}`, pagedRows()-len(pagedShapes))}
	for _, shape := range pagedShapes {
		want = append(want, fmt.Sprintf(`{"k":%s,"n":1}`, shape.value))
	}
	rows := pagedQuery(t, container, `SELECT c.v AS k, COUNT(1) AS n FROM c GROUP BY c.v`)
	assert.ElementsMatch(t, want, rows)

	// The groups also come back in key order, the undefined one first, which
	// is what makes two runs of one query agree.
	require.NotEmpty(t, rows)
	assert.Equal(t, want[0], rows[0])
	assert.ElementsMatch(t, []string{`{"k":{"a":1},"n":1}`, `{"k":{"b":1},"n":1}`}, rows[len(rows)-2:],
		"objects sort last")
}

// TestCrossPartitionMixedTypeOrder checks ORDER BY against the order the
// service itself returns: undefined < null < bool < number < string < array <
// object.
//
// Two different arrays, or two different objects, it cannot order, because the
// service orders those by its index's own encoding — [1,2] < [2] < [1,3] on a
// real container — and a merge comparator that disagreed with the streams it
// merges would answer in an order that is neither. Those it fails instead.
func TestCrossPartitionMixedTypeOrder(t *testing.T) {
	container := pagedFixture(t)

	scalars := make([]int, 0, len(pagedShapes))
	for _, shape := range pagedShapes {
		if shape.value[0] != '[' && shape.value[0] != '{' {
			scalars = append(scalars, shape.seq)
		}
	}
	got := pagedSeqs(t, container,
		`SELECT c.seq FROM c WHERE IS_DEFINED(c.v) AND NOT IS_ARRAY(c.v) AND NOT IS_OBJECT(c.v) ORDER BY c.v`)
	assert.Equal(t, scalars, got, "the scalar classes are ordered as the service orders them")

	// One composite among them still orders, since the class ranking is the
	// service's own: an array sorts after every string.
	withOneArray := append(slices.Clone(scalars), pagedShapes[7].seq)
	got = pagedSeqs(t, container, fmt.Sprintf(
		`SELECT c.seq FROM c WHERE IS_DEFINED(c.v) AND NOT IS_OBJECT(c.v) AND (NOT IS_ARRAY(c.v) OR c.seq = %d) ORDER BY c.v`,
		pagedShapes[7].seq))
	assert.Equal(t, withOneArray, got, "an array sorts after the scalars")

	// Two of them is the case no comparator can serve.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	_, err := QueryEveryPartition(ctx, container, `SELECT c.seq FROM c WHERE IS_DEFINED(c.v) ORDER BY c.v`, nil, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot order two different")
}
