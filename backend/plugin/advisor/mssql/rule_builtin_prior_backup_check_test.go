package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func TestPrepareTransformationUsesOmniAST(t *testing.T) {
	sql := `UPDATE p SET c1 = 1 FROM dbo.pokes AS p WHERE p.c1 = 1;
DELETE p FROM dbo.pokes AS p WHERE p.c1 = 1;`
	omniStmts, err := tsqlparser.ParseTSQLOmni(sql)
	require.NoError(t, err)

	var parsedStatements []base.ParsedStatement
	for _, stmt := range omniStmts {
		start := &storepb.Position{
			Line:   int32(stmt.Start.Line),
			Column: int32(stmt.Start.Column),
		}
		parsedStatements = append(parsedStatements, base.ParsedStatement{
			Statement: base.Statement{
				Text:  stmt.Text,
				Start: start,
			},
			AST: &tsqlparser.OmniAST{
				Node:          stmt.AST,
				StartPosition: start,
			},
		})
	}

	statementInfoList := prepareTransformation("master", parsedStatements)
	require.Len(t, statementInfoList, 2)
	require.Equal(t, StatementTypeUpdate, statementInfoList[0].table.StatementType)
	require.Equal(t, StatementTypeDelete, statementInfoList[1].table.StatementType)
	for _, info := range statementInfoList {
		require.Equal(t, "master", info.table.Database)
		require.Equal(t, "dbo", info.table.Schema)
		require.Equal(t, "pokes", info.table.Table)
		require.Equal(t, "p", info.table.Alias)
	}
}
