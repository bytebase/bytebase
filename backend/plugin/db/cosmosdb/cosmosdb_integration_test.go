// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cosmosdb

// End-to-end integration tests for BYT-9239 Stage 1. Live Azure-backed; these
// tests skip automatically when AZURE_COSMOS_KEY is not set so the default
// unit-test run stays hermetic.
//
// To run against the bytebase-cosmostest account:
//   export AZURE_COSMOS_ENDPOINT="https://bytebase-cosmostest.documents.azure.com:443/"
//   export AZURE_COSMOS_KEY="$(az cosmosdb keys list \
//       --name bytebase-cosmostest \
//       --resource-group rg-bytebase-cosmostest \
//       --type keys --query primaryMasterKey -o tsv)"
//   go test -count=1 -run "^TestIntegration_BYT9239" ./backend/plugin/db/cosmosdb/

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

func envOrDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// newTestDriver constructs a Driver directly with a master-key-authenticated
// client, bypassing Open's Azure-AD path (which requires OAuth credentials).
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

// TestIntegration_BYT9239_Stage1Queries runs every Stage-1 target query
// end-to-end through the Bytebase driver against a live Azure Cosmos account.
// Each query must succeed (no error) and produce at least one row of output.
func TestIntegration_BYT9239_Stage1Queries(t *testing.T) {
	endpoint, key, dbName, container := skipIfNoAzure(t)
	driver := newTestDriver(t, endpoint, key, dbName)

	cases := []struct {
		id     string
		sql    string
		minRow int // minimum expected row count (>=1 for every Stage-1 query)
	}{
		{"5.1", `SELECT COUNT(1) AS totalRecords FROM c`, 1},
		{"5.1b", `SELECT VALUE COUNT(1) FROM c`, 1},
		{"5.2", `SELECT COUNT(1) AS aeCount FROM c WHERE c.country = "AE"`, 1},
		{"5.2b", `SELECT VALUE COUNT(1) FROM c WHERE c.country = "AE"`, 1},
		{"5.3", `SELECT SUM(StringToNumber(c.population)) AS totalPop FROM c`, 1},
		{"5.3b", `SELECT VALUE SUM(StringToNumber(c.population)) FROM c`, 1},
		{"5.4", `SELECT AVG(StringToNumber(c.population)) AS avgPop FROM c`, 1},
		{"5.4b", `SELECT VALUE AVG(StringToNumber(c.population)) FROM c`, 1},
		{"5.5", `SELECT MIN(StringToNumber(c.population)) AS minPop, MAX(StringToNumber(c.population)) AS maxPop FROM c WHERE StringToNumber(c.population) > 0`, 1},
		{"11.2", `SELECT VALUE COUNT(1) FROM c`, 1},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			results, err := driver.QueryConn(ctx, nil, tc.sql, db.QueryContext{Container: container})
			require.NoError(t, err, "QueryConn for %s", tc.id)
			require.Len(t, results, 1)
			require.GreaterOrEqual(t, len(results[0].Rows), tc.minRow,
				"query %s produced no rows; expected >= %d", tc.id, tc.minRow)

			// Log the first row for visual inspection during CI debugging.
			if len(results[0].Rows) > 0 {
				v := results[0].Rows[0].GetValues()[0]
				t.Logf("%s -> %s", tc.id, v.GetStringValue())
			}
		})
	}
}

// ensureImports keeps the storepb + json imports used transitively when the
// build tags include emulator tests. If this test file grows to include an
// emulator-only path (currently not exercised in Stage 1), these will be
// needed; placeholder avoids 'unused import' errors in mixed builds.
var _ = json.Valid
var _ = storepb.Engine_COSMOSDB
