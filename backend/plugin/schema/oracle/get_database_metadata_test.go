package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseMetadataRegistry(t *testing.T) {
	metadata, err := schema.GetDatabaseMetadata(storepb.Engine_ORACLE, `
CREATE TABLE T (
    ID NUMBER PRIMARY KEY
);
`)
	require.NoError(t, err)
	require.Len(t, metadata.Schemas, 1)
	require.Len(t, metadata.Schemas[0].Tables, 1)
	require.Equal(t, "T", metadata.Schemas[0].Tables[0].Name)
	require.Len(t, metadata.Schemas[0].Tables[0].Columns, 1)

	_, err = schema.GetDatabaseMetadata(storepb.Engine_ORACLE, "CREATE TABLE (")
	require.Error(t, err)
}
