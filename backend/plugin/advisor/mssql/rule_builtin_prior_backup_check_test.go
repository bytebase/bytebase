package mssql

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func TestPrepareTransformationUsesAST(t *testing.T) {
	sql := `UPDATE p SET c1 = 1 FROM dbo.pokes AS p WHERE p.c1 = 1;
DELETE p FROM dbo.pokes AS p WHERE p.c1 = 1;`
	stmts, err := tsqlparser.ParseTSQL(sql)
	require.NoError(t, err)

	var parsedStatements []base.ParsedStatement
	for _, stmt := range stmts {
		start := &storepb.Position{
			Line:   int32(stmt.Start.Line),
			Column: int32(stmt.Start.Column),
		}
		parsedStatements = append(parsedStatements, base.ParsedStatement{
			Statement: base.Statement{
				Text:  stmt.Text,
				Start: start,
			},
			AST: &tsqlparser.AST{
				Node:          stmt.AST,
				StartPosition: start,
			},
		})
	}

	advices, err := advisor.Check(context.Background(), storepb.Engine_MSSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, advisor.Context{
		DBType:            storepb.Engine_MSSQL,
		DBSchema:          &metadatapb.DatabaseSchemaMetadata{Name: "master"},
		EnablePriorBackup: true,
		Rule: &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
			Level: storepb.SQLReviewRule_WARNING,
		},
		ParsedStatements: parsedStatements,
		ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
			return []string{"bbdataarchive"}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, advices, 1)
	require.Equal(t, code.BuiltinPriorBackupCheck.Int32(), advices[0].Code)
	require.Equal(t, `The statement type is not the same for all statements on the same table "master.dbo.pokes"`, advices[0].Content)
}
