package gonative_test

// Stage 6 live-Azure integration tests. Guarded by AZURE_COSMOS_KEY.

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newContainerS6(t *testing.T) *azcosmos.ContainerClient {
	t.Helper()
	endpoint, key, dbName, collName := skipIfNoAzureStage2(t)
	cred, err := azcosmos.NewKeyCredential(key)
	require.NoError(t, err)
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	require.NoError(t, err)
	container, err := client.NewContainer(dbName, collName)
	require.NoError(t, err)
	return container
}

// TestIntegration_BYT9239_Stage6_6_1_CountByCountry verifies cross-partition
// GROUP BY with COUNT against the live Cosmos account. The assertion is
// independent of the particular container contents: the grand total of all
// groups' cityCount must equal the total row count of the container (obtained
// via a separate VALUE COUNT(1) query).
func TestIntegration_BYT9239_Stage6_6_1_CountByCountry(t *testing.T) {
	container := newContainerS6(t)

	// Baseline: total row count.
	totalRows := drainAzure(t, container, `SELECT VALUE COUNT(1) FROM c`)
	require.Len(t, totalRows, 1)
	var total float64
	require.NoError(t, json.Unmarshal(totalRows[0], &total))

	// Grouped query.
	rows := drainAzure(t, container, `SELECT c.country, COUNT(1) AS cityCount FROM c GROUP BY c.country`)
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
		require.True(t, ok, "cityCount must be numeric: %s", r)
		summed += cnt
	}
	assert.Equal(t, total, summed, "sum of per-country counts must equal total row count")
	t.Logf("%d country groups covering %.0f rows", len(rows), total)
}

// TestIntegration_BYT9239_Stage6_6_2_MultipleAggregates verifies multi-aggregate
// GROUP BY (COUNT + SUM) cross-partition against the live account. Asserts the
// summed counts match the container total and that totalPop per group is
// non-negative and finite.
func TestIntegration_BYT9239_Stage6_6_2_MultipleAggregates(t *testing.T) {
	container := newContainerS6(t)

	totalRows := drainAzure(t, container, `SELECT VALUE COUNT(1) FROM c`)
	require.Len(t, totalRows, 1)
	var total float64
	require.NoError(t, json.Unmarshal(totalRows[0], &total))

	rows := drainAzure(t, container, `SELECT c.countryRegion, COUNT(1) AS count, SUM(StringToNumber(c.population)) AS totalPop FROM c GROUP BY c.countryRegion`)
	require.NotEmpty(t, rows)

	var sumCounts float64
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
		sumCounts += cnt
		pop, ok := obj["totalPop"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, pop, float64(0), "totalPop must be non-negative for region %s", region)
	}
	assert.Equal(t, total, sumCounts, "sum of per-region counts must equal total row count")
	t.Logf("%d region groups covering %.0f rows", len(rows), total)
}
