package plsql

import (
	"testing"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementTypesSeam(t *testing.T) {
	parsed, err := base.ParseStatements(storepb.Engine_ORACLE, "CREATE TABLE t (id NUMBER); INSERT INTO t VALUES (1); INSERT INTO t VALUES (2);")
	require.NoError(t, err)

	var asts []base.AST
	for _, stmt := range parsed {
		if stmt.AST != nil {
			asts = append(asts, stmt.AST)
		}
	}

	types, err := base.GetStatementTypes(storepb.Engine_ORACLE, asts)
	require.NoError(t, err)
	require.ElementsMatch(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_INSERT,
	}, types)

	_, err = base.GetStatementTypes(storepb.Engine_ORACLE, []base.AST{&fakeAST{}})
	require.Error(t, err)

	require.Equal(t, storepb.StatementType_CREATE_DATABASE, base.OmniStatementType(analysis.CreateDatabase))
	require.Equal(t, storepb.StatementType_DROP_INDEX, base.OmniStatementType(analysis.DropIndex))
	require.Equal(t, storepb.StatementType_ALTER_TABLE, base.OmniStatementType(analysis.AlterTable))
	require.Equal(t, storepb.StatementType_EXECUTE, base.OmniStatementType(analysis.Execute))
	require.Equal(t, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, base.OmniStatementType(analysis.StatementType(999)))
}
