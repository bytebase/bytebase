package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestFilterEmptyStatements(t *testing.T) {
	statements := []Statement{
		{Text: "SELECT 1", Empty: false},
		{Text: "", Empty: true},
		{Text: "SELECT 2", Empty: false},
		{Text: "-- comment", Empty: true},
	}

	result := FilterEmptyStatements(statements)

	require.Len(t, result, 2)
	require.Equal(t, "SELECT 1", result[0].Text)
	require.Equal(t, "SELECT 2", result[1].Text)
}

func TestFilterEmptyStatementsWithIndexes(t *testing.T) {
	statements := []Statement{
		{Text: "SELECT 1", Empty: false},
		{Text: "", Empty: true},
		{Text: "SELECT 2", Empty: false},
	}

	result, indexes := FilterEmptyStatementsWithIndexes(statements)

	require.Len(t, result, 2)
	require.Equal(t, []int32{0, 2}, indexes)
}

func TestSingleSQLToStatement(t *testing.T) {
	sql := SingleSQL{
		Text:            "SELECT 1",
		Empty:           false,
		Start:           &storepb.Position{Line: 1, Column: 1},
		End:             &storepb.Position{Line: 1, Column: 9},
		ByteOffsetStart: 0,
		ByteOffsetEnd:   8,
	}

	stmt := SingleSQLToStatement(sql)

	require.Equal(t, sql.Text, stmt.Text)
	require.Equal(t, sql.Empty, stmt.Empty)
	require.Equal(t, sql.Start, stmt.StartPosition)
	require.Equal(t, sql.End, stmt.EndPosition)
	require.Equal(t, sql.ByteOffsetStart, stmt.ByteOffsetStart)
	require.Equal(t, sql.ByteOffsetEnd, stmt.ByteOffsetEnd)
	require.Nil(t, stmt.AST)
}

func TestStatementToSingleSQL(t *testing.T) {
	stmt := Statement{
		Text:            "SELECT 1",
		Empty:           false,
		StartPosition:   &storepb.Position{Line: 1, Column: 1},
		EndPosition:     &storepb.Position{Line: 1, Column: 9},
		ByteOffsetStart: 0,
		ByteOffsetEnd:   8,
	}

	sql := StatementToSingleSQL(stmt)

	require.Equal(t, stmt.Text, sql.Text)
	require.Equal(t, stmt.Empty, sql.Empty)
	require.Equal(t, stmt.StartPosition, sql.Start)
	require.Equal(t, stmt.EndPosition, sql.End)
	require.Equal(t, stmt.ByteOffsetStart, sql.ByteOffsetStart)
	require.Equal(t, stmt.ByteOffsetEnd, sql.ByteOffsetEnd)
}
