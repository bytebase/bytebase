package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResourcesSeam(t *testing.T) {
	parsed, err := base.ParseStatements(storepb.Engine_MSSQL, "UPDATE dbo.users SET id = 2; INSERT INTO dbo.users VALUES (1);")
	require.NoError(t, err)

	asts := []base.AST{&AST{Node: nil}} // nil Node is skipped
	for _, stmt := range parsed {
		asts = append(asts, stmt.AST)
	}

	summary, err := base.ExtractChangedResources(storepb.Engine_MSSQL, "app", "dbo", nil, asts, "")
	require.NoError(t, err)
	require.Equal(t, 1, summary.DMLCount)
	require.Equal(t, 1, summary.InsertCount)
	require.Len(t, summary.DMLStatements, 1)

	want := model.NewChangedResources(nil)
	want.AddTable("app", "dbo", &storepb.ChangedResourceTable{Name: "users"}, false)
	require.Equal(t, want, summary.ChangedResources)

	// DatabaseOnly statement (explicitly qualified non-table DDL) calls AddDatabase.
	viewParsed, err := base.ParseStatements(storepb.Engine_MSSQL, "DROP VIEW app2.dbo.v1;")
	require.NoError(t, err)
	viewSummary, err := base.ExtractChangedResources(storepb.Engine_MSSQL, "app", "dbo", nil, base.ExtractASTs(viewParsed), "")
	require.NoError(t, err)
	wantDB := model.NewChangedResources(nil)
	wantDB.AddDatabase("app2")
	require.Equal(t, wantDB, viewSummary.ChangedResources)

	_, err = base.ExtractChangedResources(storepb.Engine_MSSQL, "app", "dbo", nil, []base.AST{&fakeAST{}}, "")
	require.Error(t, err)
}
