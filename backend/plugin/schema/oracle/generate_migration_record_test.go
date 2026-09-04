//go:build oracle_record

// This file is the provenance of testdata/generate_migration/*.json: it applies
// each fixture's DDL to a real Oracle and records what the engine reports, so
// the golden test can run the generator against true synced metadata without a
// container. Refresh the fixtures with:
//
//	go test -tags oracle_record -count=1 -timeout 40m \
//	    ./backend/plugin/schema/oracle/ -run TestRecordGenerateMigrationFixtures
//
// Stats are zeroed because they churn on every run. Everything else is recorded
// as the engine reported it, including system-generated ISEQ$$ sequences and
// SYS_NC virtual-column indexes, which the generator has to cope with.

package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// fixtureSchemaName replaces the throwaway Oracle user this recorder creates, so
// a re-record does not churn every fixture with a new UUID-derived schema name.
const fixtureSchemaName = "TESTSCHEMA"

//nolint:tparallel
func TestRecordGenerateMigrationFixtures(t *testing.T) {
	ctx := context.Background()

	container := testcontainer.GetTestOracleContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	systemDB := container.GetDB()

	require.NoError(t, os.MkdirAll(generateMigrationFixtureDir, 0o755))

	// Oracle Free slows down heavily when too many schema-heavy cases run
	// concurrently against one container.
	const maxConcurrent = 4
	gate := make(chan struct{}, maxConcurrent)

	for _, tc := range generateMigrationCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gate <- struct{}{}
			defer func() { <-gate }()

			testUser := fmt.Sprintf("U_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
			require.NoError(t, createOracleUser(systemDB, testUser))

			driver, err := createOracleDriver(ctx, container.GetHost(), container.GetPort(), testUser)
			require.NoError(t, err)
			defer driver.Close(ctx)

			require.NoError(t, executeStatements(ctx, driver, tc.initialSchema), "failed to execute initial schema")
			schemaA, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err)

			require.NoError(t, executeStatements(ctx, driver, tc.migrationDDL), "failed to execute migration DDL")
			schemaB, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err)

			fixture := generateMigrationFixture{
				SchemaA: recordSchema(t, schemaA, testUser),
				SchemaB: recordSchema(t, schemaB, testUser),
			}
			out, err := json.MarshalIndent(fixture, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(generateMigrationFixturePath(tc.name), append(out, '\n'), 0o644))
		})
	}
}

// recordSchema renders one synced schema for storage, substituting the throwaway
// user name so the fixture is stable across re-records.
func recordSchema(t *testing.T, metadata *storepb.DatabaseSchemaMetadata, testUser string) json.RawMessage {
	t.Helper()

	zeroVolatileStats(metadata)

	raw, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(metadata)
	require.NoError(t, err)

	// The user name reaches the metadata as the schema name and inside view,
	// routine and constraint text, so substitute across the whole document.
	substituted := strings.ReplaceAll(string(raw), strings.ToUpper(testUser), fixtureSchemaName)
	return json.RawMessage(substituted)
}

// zeroVolatileStats clears the row and byte counts, which differ run to run and
// are never read by the migration generator.
func zeroVolatileStats(metadata *storepb.DatabaseSchemaMetadata) {
	for _, schemaMetadata := range metadata.Schemas {
		for _, table := range schemaMetadata.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0
		}
	}
}
