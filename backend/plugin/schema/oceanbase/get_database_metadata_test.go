package oceanbase

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestGetDatabaseMetadata pins that OceanBase reaches this package's metadata
// extractor through the schema registry.
func TestGetDatabaseMetadata(t *testing.T) {
	metadata, err := schema.GetDatabaseMetadata(storepb.Engine_OCEANBASE, "CREATE TABLE `t` (`a` int NOT NULL, PRIMARY KEY (`a`));")
	require.NoError(t, err)

	tables := metadata.GetSchemas()[0].GetTables()
	require.Len(t, tables, 1)
	require.Equal(t, "t", tables[0].GetName())
	require.Len(t, tables[0].GetColumns(), 1)
}
