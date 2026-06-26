package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseMySQLStatements(t *testing.T) {
	statement := "SELECT 1; SELECT 2;"

	statements, err := base.ParseStatements(storepb.Engine_MYSQL, statement)
	require.NoError(t, err)

	require.Len(t, statements, 2)

	// Check first statement
	require.Equal(t, "SELECT 1;", statements[0].Text)
	require.False(t, statements[0].Empty)
	require.NotNil(t, statements[0].AST)
	require.NotNil(t, statements[0].Start)

	// Check second statement
	require.Contains(t, statements[1].Text, "SELECT 2")
	require.False(t, statements[1].Empty)
	require.NotNil(t, statements[1].AST)
}
