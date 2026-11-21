package cockroachdb

import (
	"slices"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"

	// Registers the statement splitter that CockroachDB parsing uses.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

type foreignAST struct{}

func (foreignAST) ASTStartPosition() *storepb.Position { return &storepb.Position{Line: 1} }

func TestExtractChangedResourcesSeam(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:       "db",
		SearchPath: `"$user", app, public`,
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:   "app",
				Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 100, Indexes: []*metadatapb.IndexMetadata{{Name: "t_c_idx"}}}},
			},
			{
				Name:   "public",
				Tables: []*metadatapb.TableMetadata{{Name: "t2", RowCount: 50}},
			},
		},
	}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_COCKROACHDB, true /* caseSensitive */)

	// Registry dispatch, synced search_path fallback when currentSchema is empty,
	// changeMetadata callbacks (RelationSchema + IndexTable), AddTable/AddDatabase,
	// WithSearchPath formatting on DMLStatements, and row count aggregation.
	const script = `SET search_path = custom, app, public;
ALTER TABLE t ADD COLUMN c2 INT;
DELETE FROM t;
UPSERT INTO t2 VALUES (1), (2);
DROP INDEX t_c_idx;
DROP INDEX otherdb.public.idx_other;`
	got, err := base.ExtractChangedResources(storepb.Engine_COCKROACHDB, "db", "", dbMetadata, parseASTs(t, script), script)
	require.NoError(t, err)
	require.Equal(t, []string{"db.app.t", "db.public.t2"}, getTableNames(got.ChangedResources))
	require.Equal(t, []string{"otherdb"}, got.ChangedResources.GetDatabaseOnlyTargets())
	require.Equal(t, int64(100), got.ChangedResources.CountAffectedTableRows())
	require.Equal(t, 1, got.DMLCount)
	require.Equal(t, 2, got.InsertCount)
	require.Equal(t, []string{"SET LOCAL search_path TO \"custom\", \"app\", \"public\";\nDELETE FROM t"}, got.DMLStatements)

	// Foreign AST rejection.
	_, err = base.ExtractChangedResources(storepb.Engine_COCKROACHDB, "db", "public", nil, []base.AST{foreignAST{}}, "")
	require.ErrorContains(t, err, "expected CockroachDB AST")
}

func parseASTs(t *testing.T, statement string) []base.AST {
	t.Helper()
	stmts, err := base.ParseStatements(storepb.Engine_COCKROACHDB, statement)
	require.NoError(t, err)
	return base.ExtractASTs(stmts)
}

func getTableNames(changedResources *model.ChangedResources) []string {
	var names []string
	for _, database := range changedResources.Build().GetDatabases() {
		for _, schema := range database.GetSchemas() {
			for _, table := range schema.GetTables() {
				names = append(names, database.GetName()+"."+schema.GetName()+"."+table.GetName())
			}
		}
	}
	slices.Sort(names)
	return names
}
