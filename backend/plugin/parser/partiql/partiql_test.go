package partiql

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseAndValidateQuerySeam(t *testing.T) {
	// ParseStatements populates AST and StartPosition, and shifts multi-statement syntax errors.
	stmts, err := base.ParseStatements(storepb.Engine_DYNAMODB, "SELECT * FROM Music;\nSELECT Artist FROM Music;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.NotNil(t, stmts[0].AST)
	require.Equal(t, int32(1), stmts[0].AST.ASTStartPosition().Line)
	require.NotNil(t, stmts[1].AST)

	_, err = base.ParseStatements(storepb.Engine_DYNAMODB, "SELECT * FROM Music;\nSELECT FROM WHERE")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(2), syntaxErr.Position.Line)

	// ValidateSQLForEditor distinguishes DQL, DML/DDL, and syntax errors.
	readOnly, returnsData, err := base.ValidateSQLForEditor(storepb.Engine_DYNAMODB, "SELECT * FROM Music")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, returnsData)

	readOnly, returnsData, err = base.ValidateSQLForEditor(storepb.Engine_DYNAMODB, "DELETE FROM Music WHERE Artist = 'x'")
	require.NoError(t, err)
	require.False(t, readOnly)
	require.False(t, returnsData)

	readOnly, returnsData, err = base.ValidateSQLForEditor(storepb.Engine_DYNAMODB, "SELECT FROM WHERE")
	require.True(t, errors.As(err, &syntaxErr))
	require.False(t, readOnly)
	require.False(t, returnsData)
	require.Equal(t, int32(1), syntaxErr.Position.Line)
}
