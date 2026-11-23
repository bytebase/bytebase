package mariadb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseMetadata(t *testing.T) {
	ddl := "CREATE TABLE `t` (`a` int NOT NULL, PRIMARY KEY (`a`)) WITH SYSTEM VERSIONING;\n" +
		"CREATE SEQUENCE `seq` START WITH 10 INCREMENT BY 2;\n"
	metadata, err := schema.GetDatabaseMetadata(storepb.Engine_MARIADB, ddl)
	require.NoError(t, err)

	tables := metadata.GetSchemas()[0].GetTables()
	require.Len(t, tables, 1)
	require.Equal(t, "t", tables[0].GetName())
	require.Contains(t, tables[0].GetCreateOptions(), "WITH SYSTEM VERSIONING")

	sequences := metadata.GetSchemas()[0].GetSequences()
	require.Len(t, sequences, 1)
	require.Equal(t, "seq", sequences[0].GetName())
	require.Equal(t, "10", sequences[0].GetStart())
	require.Equal(t, "2", sequences[0].GetIncrement())
}
