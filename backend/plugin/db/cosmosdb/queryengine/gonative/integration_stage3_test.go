package gonative_test

// Stage 3 live-Azure integration tests for BYT-9239. Guarded by AZURE_COSMOS_KEY.
//
//   export AZURE_COSMOS_ENDPOINT="https://...documents.azure.com:443/"
//   export AZURE_COSMOS_KEY="$(az cosmosdb keys list --name ... -o tsv)"
//   go test -count=1 -run "^TestIntegration_BYT9239_Stage3" \
//       ./sdk/data/azcosmos/queryengine/gonative/

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newContainerS3(t *testing.T) *azcosmos.ContainerClient {
	t.Helper()
	endpoint, key, dbName, collName := skipIfNoAzureStage2(t) // same env guard as Stage 2
	cred, err := azcosmos.NewKeyCredential(key)
	require.NoError(t, err)
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	require.NoError(t, err)
	container, err := client.NewContainer(dbName, collName)
	require.NoError(t, err)
	return container
}

// sortKey extracts a comparable representation of a row's `population` field
// that honors Cosmos item ordering: numbers sort before strings, so we tag
// the value with the type class first. Returned tuple is (typeRank, raw).
func sortKey(t *testing.T, row []byte) (int, any) {
	t.Helper()
	var obj map[string]any
	require.NoError(t, json.Unmarshal(row, &obj))
	v, ok := obj["population"]
	require.True(t, ok, "row missing population: %s", row)
	switch x := v.(type) {
	case float64:
		return 3, x // numbers
	case string:
		return 4, x // strings
	case bool:
		return 2, x
	case nil:
		return 1, x
	default:
		return 0, x
	}
}

func TestIntegration_BYT9239_Stage3_4_1_Ascending(t *testing.T) {
	container := newContainerS3(t)
	rows := drainAzure(t, container, `SELECT c.name, c.population FROM c ORDER BY c.population ASC`)
	require.NotEmpty(t, rows, "expected rows back")

	// Verify the result is in ascending order under Cosmos item ordering.
	var prevRank int
	var prevVal any
	for i, r := range rows {
		rank, val := sortKey(t, r)
		if i == 0 {
			prevRank, prevVal = rank, val
			continue
		}
		if rank != prevRank {
			assert.GreaterOrEqual(t, rank, prevRank, "type-rank must not regress at row %d: %s", i, r)
			prevRank, prevVal = rank, val
			continue
		}
		switch pv := prevVal.(type) {
		case float64:
			assert.LessOrEqual(t, pv, val.(float64), "ascending number order broken at row %d", i)
		case string:
			assert.LessOrEqual(t, pv, val.(string), "ascending string order broken at row %d", i)
		default:
			t.Fatalf("unexpected sort key type %T at row %d", pv, i)
		}
		prevVal = val
	}
	t.Logf("ASC verified across %d rows", len(rows))
}

func TestIntegration_BYT9239_Stage3_4_2_Descending(t *testing.T) {
	container := newContainerS3(t)
	rows := drainAzure(t, container, `SELECT c.name, c.population FROM c ORDER BY c.population DESC`)
	require.NotEmpty(t, rows)

	var prevRank int
	var prevVal any
	for i, r := range rows {
		rank, val := sortKey(t, r)
		if i == 0 {
			prevRank, prevVal = rank, val
			continue
		}
		if rank != prevRank {
			assert.LessOrEqual(t, rank, prevRank, "type-rank must not ascend under DESC at row %d: %s", i, r)
			prevRank, prevVal = rank, val
			continue
		}
		switch pv := prevVal.(type) {
		case float64:
			assert.GreaterOrEqual(t, pv, val.(float64), "descending number order broken at row %d", i)
		case string:
			assert.GreaterOrEqual(t, pv, val.(string), "descending string order broken at row %d", i)
		default:
			t.Fatalf("unexpected sort key type %T at row %d", pv, i)
		}
		prevVal = val
	}
	t.Logf("DESC verified across %d rows", len(rows))
}
