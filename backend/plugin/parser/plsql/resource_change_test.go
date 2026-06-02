package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	CREATE VIEW hello AS SELECT * FROM world;
	INSERT INTO T1 (c1) VALUES (1);
	`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable(
		"DB",
		"",
		&storepb.ChangedResourceTable{
			Name: "T1",
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		InsertCount:      1,
	}

	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.NotEmpty(t, asts)

	// Pass the full asts array to extractChangedResources
	got, err := extractChangedResources("DB", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesOmniSyntax(t *testing.T) {
	statement := `CREATE TABLE IF NOT EXISTS t1 (c1 INT);
DROP TABLE IF EXISTS t2;`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable(
		"DB",
		"",
		&storepb.ChangedResourceTable{
			Name: "T1",
		},
		false,
	)
	changedResources.AddTable(
		"DB",
		"",
		&storepb.ChangedResourceTable{
			Name: "T2",
		},
		true,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
	}

	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.NotEmpty(t, asts)

	got, err := extractChangedResources("DB", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesDMLSamples(t *testing.T) {
	statement := `INSERT INTO t1 SELECT * FROM t2;
UPDATE t1 SET c1 = 5 WHERE c2 = 1;
DELETE FROM t2 WHERE c1 = 1;`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable("DB", "", &storepb.ChangedResourceTable{Name: "T1"}, false)
	changedResources.AddTable("DB", "", &storepb.ChangedResourceTable{Name: "T2"}, false)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"INSERT INTO t1 SELECT * FROM t2;",
			"UPDATE t1 SET c1 = 5 WHERE c2 = 1;",
			"DELETE FROM t2 WHERE c1 = 1;",
		},
		DMLCount: 3,
	}

	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.NotEmpty(t, asts)

	got, err := extractChangedResources("DB", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesDropIndexUsesMetadata(t *testing.T) {
	statement := `DROP INDEX idx_t1_c1;`
	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: "DB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "DB",
				Tables: []*storepb.TableMetadata{
					{
						Name: "T1",
						Indexes: []*storepb.IndexMetadata{
							{
								Name: "IDX_T1_C1",
							},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */)
	changedResources := model.NewChangedResources(dbMetadata)
	changedResources.AddTable("DB", "", &storepb.ChangedResourceTable{Name: "T1"}, false)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
	}

	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.NotEmpty(t, asts)

	got, err := extractChangedResources("DB", "", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesSkipsANTLRFallbackAST(t *testing.T) {
	updateStatement := "UPDATE t1 SET c1 = 5"
	triggerStatement := `CREATE OR REPLACE TRIGGER trg
BEFORE INSERT OR UPDATE OF col1, col2 ON tbl
REFERENCING OLD AS o NEW AS n
FOR EACH ROW
WHEN (n.col1 > 0)
BEGIN
  :n.col2 := :o.col2 + 1;
END;`
	statement := updateStatement + ";\n" + triggerStatement
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable("DB", "", &storepb.ChangedResourceTable{Name: "T1"}, false)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"UPDATE t1 SET c1 = 5;",
		},
		DMLCount: 1,
	}

	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, updateStatement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.Len(t, asts, 1)
	_, ok := asts[0].(*OmniAST)
	require.True(t, ok)

	asts = append(asts, testAST{})

	got, err := extractChangedResources("DB", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

type testAST struct{}

func (testAST) ASTStartPosition() *storepb.Position {
	return nil
}
