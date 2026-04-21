package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	DROP TABLE t1;
	ALTER TABLE t1 ADD colb INT;
	INSERT INTO t1 (c1) VALUES (1), (5);
	UPDATE t1 SET c1 = 5;
	`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable(
		"DB",
		"dbo",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"UPDATE t1 SET c1 = 5;",
		},
		DMLCount:    1,
		InsertCount: 2,
	}

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.Len(t, asts, 5)
	got, err := extractChangedResources("DB", "dbo", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResources_DropIndex(t *testing.T) {
	statement := `DROP INDEX idx1 ON t1;`

	want := model.NewChangedResources(nil)
	want.AddTable("DB", "dbo", &storepb.ChangedResourceTable{Name: "t1"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResources_InsertDefaultValues(t *testing.T) {
	statement := `INSERT INTO t1 DEFAULT VALUES;`

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
	require.NoError(t, err)
	require.Equal(t, 1, got.InsertCount)
	require.Equal(t, 0, got.DMLCount)
	require.Empty(t, got.SampleDMLS)
}
