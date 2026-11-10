package gonative_test

// Stage 4 live-Azure integration test for BYT-9239. Guarded by AZURE_COSMOS_KEY.
//
//   export AZURE_COSMOS_ENDPOINT="https://...documents.azure.com:443/"
//   export AZURE_COSMOS_KEY="$(az cosmosdb keys list --name ... -o tsv)"
//   go test -count=1 -run "^TestIntegration_BYT9239_Stage4" \
//       ./sdk/data/azcosmos/queryengine/gonative/

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_BYT9239_Stage4_1_3_TopTen(t *testing.T) {
	endpoint, key, dbName, collName := skipIfNoAzureStage2(t)
	cred, err := azcosmos.NewKeyCredential(key)
	require.NoError(t, err)
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	require.NoError(t, err)
	container, err := client.NewContainer(dbName, collName)
	require.NoError(t, err)

	rows := drainAzure(t, container, `SELECT TOP 10 * FROM c`)
	assert.Len(t, rows, 10, "TOP 10 across partitions must emit exactly 10 rows end-to-end")
	for i, r := range rows {
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r, &obj), "row %d failed to parse: %s", i, r)
		_, hasCountry := obj["country"]
		assert.True(t, hasCountry, "row %d should be a full document with a country field: %s", i, r)
	}
	t.Logf("emitted %d rows", len(rows))
}
