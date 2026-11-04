package gonative_test

// Stage 2 live-Azure integration tests for BYT-9239. Guarded by AZURE_COSMOS_KEY;
// skips automatically when unset so the default unit-test run stays hermetic.
//
//   export AZURE_COSMOS_ENDPOINT="https://...documents.azure.com:443/"
//   export AZURE_COSMOS_KEY="$(az cosmosdb keys list --name ... -o tsv)"
//   go test -count=1 -run "^TestIntegration_BYT9239_Stage2" \
//       ./sdk/data/azcosmos/queryengine/gonative/

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db/cosmosdb/queryengine/gonative"
)

func skipIfNoAzureStage2(t *testing.T) (endpoint, key, dbName, collName string) {
	t.Helper()
	endpoint = os.Getenv("AZURE_COSMOS_ENDPOINT")
	key = os.Getenv("AZURE_COSMOS_KEY")
	if endpoint == "" || key == "" {
		t.Skip("set AZURE_COSMOS_ENDPOINT + AZURE_COSMOS_KEY to run live integration tests")
	}
	dbName = envOrDefaultS2("AZURE_COSMOS_DB", "testdb")
	collName = envOrDefaultS2("AZURE_COSMOS_CONTAINER", "WorldCities")
	return endpoint, key, dbName, collName
}

func envOrDefaultS2(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// drainAzure runs a cross-partition query end-to-end against a live Cosmos
// account and returns every row it produces. Engages the default engine
// (engage-on-error path), so this covers the real "gateway fails → engine
// takes over" flow.
func drainAzure(t *testing.T, container *azcosmos.ContainerClient, sql string) [][]byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pager := container.NewQueryItemsPager(sql, azcosmos.NewPartitionKey(), &azcosmos.QueryOptions{QueryEngine: gonative.Default()})
	var items [][]byte
	for pager.More() {
		page, err := pager.NextPage(ctx)
		require.NoError(t, err, "NextPage for %q", sql)
		items = append(items, page.Items...)
	}
	return items
}

func TestIntegration_BYT9239_Stage2_10_1_ObjectDistinct(t *testing.T) {
	endpoint, key, dbName, collName := skipIfNoAzureStage2(t)
	cred, err := azcosmos.NewKeyCredential(key)
	require.NoError(t, err)
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	require.NoError(t, err)
	container, err := client.NewContainer(dbName, collName)
	require.NoError(t, err)

	rows := drainAzure(t, container, `SELECT DISTINCT c.country FROM c`)
	require.NotEmpty(t, rows, "expected at least one distinct country")

	// Each row is {"country": "AE"} etc — no duplicates.
	seen := map[string]bool{}
	for _, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj))
		country, ok := obj["country"].(string)
		require.True(t, ok, "row missing country field: %s", r)
		assert.False(t, seen[country], "duplicate country emitted: %s", country)
		seen[country] = true
	}
	t.Logf("distinct countries: %d", len(seen))
}

func TestIntegration_BYT9239_Stage2_10_2_DistinctValue(t *testing.T) {
	endpoint, key, dbName, collName := skipIfNoAzureStage2(t)
	cred, err := azcosmos.NewKeyCredential(key)
	require.NoError(t, err)
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	require.NoError(t, err)
	container, err := client.NewContainer(dbName, collName)
	require.NoError(t, err)

	rows := drainAzure(t, container, `SELECT DISTINCT VALUE c.countryRegion FROM c`)
	require.NotEmpty(t, rows, "expected at least one distinct countryRegion")

	seen := map[string]bool{}
	for _, r := range rows {
		var v string
		require.NoError(t, json.Unmarshal(r, &v))
		assert.False(t, seen[v], "duplicate region emitted: %s", v)
		seen[v] = true
	}
	t.Logf("distinct countryRegions: %d", len(seen))
}
