package redshift

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResourcesSeam(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
ALTER TABLE t1 ADD COLUMN c2 INT;
INSERT INTO t1 VALUES (1), (2);
UPDATE t1 SET c2 = 2;`
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_REDSHIFT, true /* caseSensitive */)
	wantResources := model.NewChangedResources(dbMetadata)
	wantResources.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t1"}, true)

	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, statement)
	require.NoError(t, err)
	got, err := base.ExtractChangedResources(storepb.Engine_REDSHIFT, "db", "public", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	require.Equal(t, &base.ChangeSummary{
		ChangedResources: wantResources,
		DMLStatements:    []string{"UPDATE t1 SET c2 = 2;"},
		DMLCount:         1,
		InsertCount:      2,
	}, got)
}
