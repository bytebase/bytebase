package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseMetadataSeam(t *testing.T) {
	t.Run("empty schema returns empty metadata", func(t *testing.T) {
		meta, err := schema.GetDatabaseMetadata(storepb.Engine_TIDB, "  \n")
		require.NoError(t, err)
		require.Empty(t, meta.Schemas)
	})

	t.Run("strips scratch database name and view qualifier when DDL selects no database", func(t *testing.T) {
		meta, err := schema.GetDatabaseMetadata(storepb.Engine_TIDB, `
CREATE TABLE t (id INT PRIMARY KEY);
CREATE VIEW v AS SELECT id FROM t;
`)
		require.NoError(t, err)
		require.Empty(t, meta.Name)
		require.Len(t, meta.Schemas, 1)
		require.Len(t, meta.Schemas[0].Views, 1)
		require.NotContains(t, meta.Schemas[0].Views[0].Definition, catalogDBName)
	})

	t.Run("preserves explicit database name and view qualifier when DDL selects one", func(t *testing.T) {
		meta, err := schema.GetDatabaseMetadata(storepb.Engine_TIDB, `
CREATE DATABASE app;
USE app;
CREATE TABLE t (id INT PRIMARY KEY);
CREATE VIEW v AS SELECT id FROM t;
`)
		require.NoError(t, err)
		require.Equal(t, "app", meta.Name)
		require.Len(t, meta.Schemas, 1)
		require.Len(t, meta.Schemas[0].Views, 1)
		require.Contains(t, meta.Schemas[0].Views[0].Definition, "`app`.")
	})

	t.Run("invalid DDL returns error", func(t *testing.T) {
		_, err := schema.GetDatabaseMetadata(storepb.Engine_TIDB, "CREATE TABLE (")
		require.Error(t, err)
	})
}
