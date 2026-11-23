package mariadb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestWalkThroughMariaDBSimulatesMariaDBOnlyStatements(t *testing.T) {
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MARIADB, true)
	statement := `
CREATE OR REPLACE TABLE t (
	a INT PRIMARY KEY,
	k INT,
	KEY idx_k (k) IGNORED
);
ALTER TABLE t ADD COLUMN b INT;
CREATE TABLE hist (id INT PRIMARY KEY) WITH SYSTEM VERSIONING;
CREATE SEQUENCE seq START WITH 10 INCREMENT BY 2;
CREATE OR REPLACE VIEW v AS SELECT a, b FROM t;
`
	sm := sheet.NewManager()
	stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
	require.Empty(t, advices)

	advice := schema.WalkThroughWithContext(context.Background(), storepb.Engine_MARIADB, schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
	require.Nil(t, advice)

	s := state.GetSchemaMetadata("")
	require.NotNil(t, s)
	tbl := s.GetTable("t")
	require.NotNil(t, tbl)
	require.NotNil(t, tbl.GetColumn("a"))
	require.NotNil(t, tbl.GetColumn("b"))
	require.NotNil(t, s.GetTable("hist"))
	require.Contains(t, s.GetTable("hist").GetProto().GetCreateOptions(), "WITH SYSTEM VERSIONING")
	require.NotNil(t, s.GetView("v"))
	require.Len(t, state.GetProto().GetSchemas()[0].GetSequences(), 1)
	require.Equal(t, "seq", state.GetProto().GetSchemas()[0].GetSequences()[0].GetName())
}

func TestWalkThroughMariaDBSimulatesSharedStatements(t *testing.T) {
	state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MARIADB, true)
	statement := "CREATE TABLE t (a INT);\nALTER TABLE t ADD COLUMN b INT;"
	sm := sheet.NewManager()
	stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
	require.Empty(t, advices)

	advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
	require.Nil(t, advice)
	require.NotNil(t, state.GetSchemaMetadata("").GetTable("t").GetColumn("b"))
}

func TestWalkThroughMariaDBErrorAndPrecheck(t *testing.T) {
	t.Run("duplicate table error", func(t *testing.T) {
		state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "test",
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{
					Name:    "t",
					Columns: []*metadatapb.ColumnMetadata{{Name: "a", Type: "int"}},
				}},
			}},
		}, nil, nil, storepb.Engine_MARIADB, true)
		statement := "CREATE TABLE t (a INT);"
		sm := sheet.NewManager()
		stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
		require.Empty(t, advices)

		advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
		require.NotNil(t, advice)
		require.Equal(t, code.TableExists.Int32(), advice.Code)
	})

	t.Run("cross-database sequence precheck", func(t *testing.T) {
		state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MARIADB, true)
		statement := "CREATE SEQUENCE other_db.seq START WITH 1;"
		sm := sheet.NewManager()
		stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
		require.Empty(t, advices)

		advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
		require.NotNil(t, advice)
		require.Equal(t, code.NotCurrentDatabase.Int32(), advice.Code)
	})

	t.Run("cross-database rename succeeds", func(t *testing.T) {
		state := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "test",
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{
					Name:    "t",
					Columns: []*metadatapb.ColumnMetadata{{Name: "a", Type: "int"}},
				}},
			}},
		}, nil, nil, storepb.Engine_MARIADB, true)
		statement := "RENAME TABLE t TO archive_db.t_old;"
		sm := sheet.NewManager()
		stmts, advices := sm.GetStatementsForChecks(storepb.Engine_MARIADB, statement)
		require.Empty(t, advices)

		advice := WalkThroughWithContext(context.Background(), schema.WalkThroughContext{RawSQL: statement}, state, base.ExtractASTs(stmts))
		require.Nil(t, advice)
		require.Nil(t, state.GetSchemaMetadata("").GetTable("t"))
	})
}
