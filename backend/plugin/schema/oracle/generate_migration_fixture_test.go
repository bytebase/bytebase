package oracle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const generateMigrationFixtureDir = "testdata/generate_migration"

// generateMigrationFixture is one case's pair of engine-reported schemas: the
// state after initialSchema, and the state after migrationDDL. Both are
// protojson because storepb messages do not round-trip through encoding/json.
type generateMigrationFixture struct {
	SchemaA json.RawMessage `json:"schemaA"`
	SchemaB json.RawMessage `json:"schemaB"`
}

func generateMigrationFixturePath(name string) string {
	return filepath.Join(generateMigrationFixtureDir, name+".json")
}

func generateMigrationGoldenPath(name string) string {
	return filepath.Join(generateMigrationFixtureDir, name+".sql")
}

// loadGenerateMigrationFixture reads the metadata Oracle reported for a case.
func loadGenerateMigrationFixture(t *testing.T, name string) (*storepb.DatabaseSchemaMetadata, *storepb.DatabaseSchemaMetadata) {
	t.Helper()

	path := generateMigrationFixturePath(name)
	raw, err := os.ReadFile(path)
	require.NoErrorf(t, err, "missing fixture %s; regenerate with: go test -tags oracle_record -run TestRecordGenerateMigrationFixtures ./backend/plugin/schema/oracle/", path)

	var fixture generateMigrationFixture
	require.NoError(t, json.Unmarshal(raw, &fixture), "failed to read %s", path)

	schemaA := &storepb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(fixture.SchemaA, schemaA), "failed to decode schemaA in %s", path)
	schemaB := &storepb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal(fixture.SchemaB, schemaB), "failed to decode schemaB in %s", path)

	return schemaA, schemaB
}
