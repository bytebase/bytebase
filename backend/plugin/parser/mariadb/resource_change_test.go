package mariadb

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResourcesSeam(t *testing.T) {
	statement := "CREATE OR REPLACE TABLE other_db.u (a INT);\nALTER TABLE t DROP COLUMN IF EXISTS b;\nINSERT INTO t (a) VALUES (1), (2);\nDELETE FROM d WHERE a = 1 RETURNING a;\nCREATE VIEW other_db.v AS SELECT 1;"

	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
	require.NoError(t, err)

	got, err := base.ExtractChangedResources(storepb.Engine_MARIADB, "db", "", nil, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)

	wantResources := model.NewChangedResources(nil)
	wantResources.AddTable("other_db", "", &storepb.ChangedResourceTable{Name: "u"}, true)
	wantResources.AddTable("db", "", &storepb.ChangedResourceTable{Name: "t"}, true)
	wantResources.AddTable("db", "", &storepb.ChangedResourceTable{Name: "d"}, false)
	wantResources.AddDatabase("other_db")

	require.Equal(t, wantResources, got.ChangedResources)
	require.Equal(t, []string{"DELETE FROM d WHERE a = 1 RETURNING a;"}, got.DMLStatements)
	require.Equal(t, 1, got.DMLCount)
	require.Equal(t, 2, got.InsertCount)

	// AffectsData propagates to model.ChangedResources.CountAffectedTableRows.
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 1000}}}},
	}, nil, nil, storepb.Engine_MARIADB, false)
	replaceStmts, err := base.ParseStatements(storepb.Engine_MARIADB, "CREATE OR REPLACE TABLE t (a INT);")
	require.NoError(t, err)
	replaceGot, err := base.ExtractChangedResources(storepb.Engine_MARIADB, "db", "", dbMetadata, base.ExtractASTs(replaceStmts), "CREATE OR REPLACE TABLE t (a INT);")
	require.NoError(t, err)
	require.Equal(t, int64(1000), replaceGot.ChangedResources.CountAffectedTableRows())

	// Foreign AST rejection.
	_, err = extractChangedResources("db", "", nil, []base.AST{foreignAST{}}, "CREATE TABLE t (a INT);")
	require.ErrorContains(t, err, "expected AST for MariaDB")
}
