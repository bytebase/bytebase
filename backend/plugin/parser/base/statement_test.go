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

func TestExtractASTs(t *testing.T) {
	ast1 := &ANTLRAST{StartPosition: &storepb.Position{Line: 1}}
	ast2 := &ANTLRAST{StartPosition: &storepb.Position{Line: 2}}

	statements := []ParsedStatement{
		{Statement: Statement{Text: "SELECT 1"}, AST: ast1},
		{Statement: Statement{Text: "SELECT 2"}, AST: ast2},
	}

	asts := ExtractASTs(statements)

	require.Len(t, asts, 2)
	require.Equal(t, ast1, asts[0])
	require.Equal(t, ast2, asts[1])
}

func TestParsedStatementEmbedding(t *testing.T) {
	// Test that ParsedStatement embeds Statement correctly
	// Fields should be accessible directly
	ps := ParsedStatement{
		Statement: Statement{
			Text:  "SELECT 1",
			Start: &storepb.Position{Line: 6, Column: 1},
		},
		AST: &ANTLRAST{StartPosition: &storepb.Position{Line: 6}},
	}

	// Direct access to embedded fields
	require.Equal(t, "SELECT 1", ps.Text)
	require.Equal(t, 5, ps.BaseLine()) // BaseLine() = Start.Line - 1 = 6 - 1 = 5
	require.Equal(t, int32(6), ps.Start.Line)
	require.NotNil(t, ps.AST)
}
