//go:build oracle_record

// This file is the provenance of testdata/generate_migration/*.json: it applies
// each fixture's DDL to a real Oracle, verifies the generated rollback against
// that Oracle, and records what the engine reports, so the golden test can run
// the generator against true synced metadata without a container. Refresh the
// fixtures with:
//
//	go test -tags oracle_record -count=1 -timeout 40m \
//	    ./backend/plugin/schema/oracle/ -run TestRecordGenerateMigrationFixtures
//
// The live round trip the golden test cannot do lives here: generate the
// rollback, execute it, and require that it restores the initial schema. A
// generator that emits invalid or incomplete DDL fails the re-record instead of
// silently blessing a new golden.
//
// Recorded metadata is what the engine reported, minus two adjustments that
// would otherwise churn every fixture on re-record: table stats are zeroed, and
// Oracle-allocated identifiers are canonicalized (see canonicalizeGeneratedNames).

package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// fixtureSchemaName replaces the throwaway Oracle user this recorder creates, so
// a re-record does not churn every fixture with a new UUID-derived schema name.
const fixtureSchemaName = "TESTSCHEMA"

// oracleGeneratedName matches the identifiers Oracle allocates for itself:
// identity-column sequences and the names it invents for unnamed constraints.
// Both embed a database object number that depends on allocation order across
// concurrently recording subtests, so the same schema recorded twice yields
// different numbers.
var oracleGeneratedName = regexp.MustCompile(`ISEQ\$\$_[0-9]+|SYS_C[0-9]+`)

//nolint:tparallel
func TestRecordGenerateMigrationFixtures(t *testing.T) {
	ctx := context.Background()

	container := sharedOracleContainer(t)
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

			// Render the fixture before the round trip below mutates these
			// messages, but write it only once Oracle has accepted the rollback.
			fixture := renderFixture(t, schemaA, schemaB, testUser)

			verifyRollbackAgainstOracle(ctx, t, driver, schemaA, schemaB)

			out, err := json.MarshalIndent(fixture, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(generateMigrationFixturePath(tc.name), append(out, '\n'), 0o644))
		})
	}
}

// verifyRollbackAgainstOracle is the live check the offline golden test cannot
// make: generate the rollback from B back to A, execute it, and require that the
// engine ends up where it started.
func verifyRollbackAgainstOracle(ctx context.Context, t *testing.T, driver db.Driver, schemaA, schemaB *storepb.DatabaseSchemaMetadata) {
	t.Helper()

	dbSchemaA := model.NewDatabaseMetadata(schemaA, nil, nil, storepb.Engine_ORACLE, false)
	dbSchemaB := model.NewDatabaseMetadata(schemaB, nil, nil, storepb.Engine_ORACLE, false)

	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_ORACLE, dbSchemaB, dbSchemaA)
	require.NoError(t, err)

	rollbackDDL, err := schema.GenerateMigration(storepb.Engine_ORACLE, diff)
	require.NoError(t, err)

	require.NoError(t, executeStatements(ctx, driver, rollbackDDL), "Oracle rejected the generated rollback:\n%s", rollbackDDL)

	schemaC, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	normalizeMetadataForComparison(schemaA)
	normalizeMetadataForComparison(schemaC)
	normalizeColumnPositions(schemaA)
	normalizeColumnPositions(schemaC)

	if mismatch := cmp.Diff(schemaA, schemaC, protocmp.Transform()); mismatch != "" {
		t.Errorf("rollback did not restore the initial schema (-want +got):\n%s\ngenerated DDL:\n%s", mismatch, rollbackDDL)
	}
}

// renderFixture marshals the pair for storage. Both schemas go through one
// canonicalization pass so an identifier Oracle allocated once carries the same
// stable name on both sides.
func renderFixture(t *testing.T, schemaA, schemaB *storepb.DatabaseSchemaMetadata, testUser string) generateMigrationFixture {
	t.Helper()

	rawA := marshalSchema(t, schemaA, testUser)
	rawB := marshalSchema(t, schemaB, testUser)

	canonical := canonicalizeGeneratedNames(rawA, rawB)
	return generateMigrationFixture{
		SchemaA: json.RawMessage(canonical[0]),
		SchemaB: json.RawMessage(canonical[1]),
	}
}

// marshalSchema renders one synced schema, substituting the throwaway user name.
func marshalSchema(t *testing.T, metadata *storepb.DatabaseSchemaMetadata, testUser string) string {
	t.Helper()

	zeroVolatileStats(metadata)

	raw, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(metadata)
	require.NoError(t, err)

	// The user name reaches the metadata as the database name and inside view,
	// routine and constraint text, so substitute across the whole document.
	return strings.ReplaceAll(string(raw), strings.ToUpper(testUser), fixtureSchemaName)
}

// canonicalizeGeneratedNames renumbers Oracle-allocated identifiers by order of
// first appearance across the documents, so re-recording an unchanged schema
// produces an unchanged fixture. Prefixes are preserved because the generator
// keys on them: get_database_definition.go and generate_migration.go both test
// for "ISEQ$$_" to recognize identity-column sequences.
func canonicalizeGeneratedNames(docs ...string) []string {
	canonical := make(map[string]string)
	seen := make(map[string]int)

	rename := func(match string) string {
		if replacement, ok := canonical[match]; ok {
			return replacement
		}
		prefix := "SYS_C"
		if strings.HasPrefix(match, "ISEQ") {
			prefix = "ISEQ$$_"
		}
		seen[prefix]++
		replacement := fmt.Sprintf("%s%07d", prefix, seen[prefix])
		canonical[match] = replacement
		return replacement
	}

	out := make([]string, 0, len(docs))
	for _, doc := range docs {
		out = append(out, oracleGeneratedName.ReplaceAllStringFunc(doc, rename))
	}
	return out
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
