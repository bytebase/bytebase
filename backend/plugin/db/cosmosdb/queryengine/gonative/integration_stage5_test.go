package gonative_test

// Stage 5 live-Azure integration tests. Guarded by AZURE_COSMOS_KEY.

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newContainerS5(t *testing.T) *azcosmos.ContainerClient {
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

// assertAscendingByName walks `rows` and fails if `rows[i]["name"]` is not in
// nondecreasing string order. Empty input is treated as trivially ordered.
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

func TestIntegration_BYT9239_Stage5_12_1_OffsetZeroLimitTen(t *testing.T) {
	container := newContainerS5(t)
	rows := drainAzure(t, container, `SELECT * FROM c ORDER BY c.name OFFSET 0 LIMIT 10`)
	require.Len(t, rows, 10, "OFFSET 0 LIMIT 10 must return exactly 10 rows")
	names := assertAscendingByName(t, rows)
	t.Logf("first page: %v", names)
}

func TestIntegration_BYT9239_Stage5_12_2_OffsetTenLimitTen(t *testing.T) {
	container := newContainerS5(t)
	// First grab the full ordered list so we know what rows 11 onward should be.
	allRows := drainAzure(t, container, `SELECT * FROM c ORDER BY c.name`)
	require.NotEmpty(t, allRows, "expected baseline rows")
	allNames := assertAscendingByName(t, allRows)

	rows := drainAzure(t, container, `SELECT * FROM c ORDER BY c.name OFFSET 10 LIMIT 10`)
	names := assertAscendingByName(t, rows)

	// The OFFSET 10 LIMIT 10 result must be a contiguous slice of the sorted
	// name list starting at index 10. If the container has < 20 rows the
	// returned set is allNames[10:]; otherwise allNames[10:20].
	start := 10
	end := start + len(rows)
	require.LessOrEqual(t, end, len(allNames), "pagination window extends past available rows")
	assert.Equal(t, allNames[start:end], names,
		"OFFSET 10 LIMIT 10 must equal the [10:%d] slice of the globally sorted list", end)
	t.Logf("second page (%d rows): %v", len(names), names)
}
