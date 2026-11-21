package starrocks

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type foreignAST struct{}

func (*foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestExtractChangedResourcesSeam(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 42}}}},
	}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_STARROCKS, false /* caseSensitive */)

	statement := "ALTER TABLE t ADD COLUMN b INT;\nINSERT INTO t VALUES (1), (2);\nUPDATE t SET a = 1;\nCREATE VIEW other.v AS SELECT 1;"
	stmts, err := base.ParseStatements(storepb.Engine_STARROCKS, statement)
	require.NoError(t, err)
	summary, err := base.ExtractChangedResources(storepb.Engine_STARROCKS, "db", "", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	require.Equal(t, int64(42), summary.ChangedResources.CountAffectedTableRows())
	require.Equal(t, 2, summary.InsertCount)
	require.Equal(t, 1, summary.DMLCount)
	require.Equal(t, []string{"UPDATE t SET a = 1;"}, summary.DMLStatements)
	require.Equal(t, []string{"other"}, summary.ChangedResources.GetDatabaseOnlyTargets())

	_, err = extractChangedResources("db", "", nil, []base.AST{&foreignAST{}}, "")
	require.ErrorContains(t, err, "expected astNode for StarRocks")
}
