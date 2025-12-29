package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQL(t *testing.T) {
	statement := `SELECT * FROM users;
	SELECT * FROM orders;
	SELECT * FROM products;`
	want := []base.Statement{
		{
			Text:     "SELECT * FROM users;",
			BaseLine: 0,
			Range:    &storepb.Range{Start: 0, End: 20},
			Start:    &storepb.Position{Line: 1, Column: 1},
			End:      &storepb.Position{Line: 1, Column: 21},
			Empty:    false,
		},
		{
			Text:     "\n\tSELECT * FROM orders;",
			BaseLine: 0,
			Range:    &storepb.Range{Start: 20, End: 43},
			Start:    &storepb.Position{Line: 2, Column: 2},
			End:      &storepb.Position{Line: 2, Column: 23},
			Empty:    false,
		},
		{
			Text:     "\n\tSELECT * FROM products;",
			BaseLine: 1,
			Range:    &storepb.Range{Start: 43, End: 68},
			Start:    &storepb.Position{Line: 3, Column: 2},
			End:      &storepb.Position{Line: 3, Column: 25},
			Empty:    false,
		},
	}

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, want, list)
}

func TestSplitSQLSingleStatement(t *testing.T) {
	statement := "SELECT * FROM users;"
	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, 1, len(list))
	require.Equal(t, "SELECT * FROM users;", list[0].Text)
	require.Equal(t, 0, list[0].BaseLine)
	require.False(t, list[0].Empty)
}

func TestSplitSQLEmpty(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantCount int
	}{
		{
			name:      "Empty string",
			statement: "",
			wantCount: 0,
		},
		{
			name:      "Only whitespace",
			statement: "   \n  \t  ",
			wantCount: 0,
		},
		{
			name:      "Only semicolon",
			statement: ";",
			wantCount: 1,
		},
		{
			name:      "Only comments and semicolon",
			statement: "-- comment\n;",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := SplitSQL(tt.statement)
			require.NoError(t, err)
			nonEmpty := 0
			for _, sql := range list {
				if !sql.Empty {
					nonEmpty++
				}
			}
			require.Equal(t, tt.wantCount, nonEmpty)
		})
	}
}

func TestSplitSQLWithComments(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantCount int
	}{
		{
			name:      "Statement with trailing comment",
			statement: "SELECT * FROM users; -- This is a comment\nSELECT * FROM orders;",
			wantCount: 2,
		},
		{
			name:      "Statement with leading comment",
			statement: "-- Comment at start\nSELECT * FROM users;",
			wantCount: 1,
		},
		{
			name:      "Only comment with semicolon",
			statement: "-- comment only\n;",
			wantCount: 1,
		},
		{
			name:      "Multiple statements with comments",
			statement: "-- First query\nSELECT * FROM users;\n-- Second query\nSELECT * FROM orders;",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := SplitSQL(tt.statement)
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, len(list))
		})
	}
}

func TestSplitSQLWithoutSemicolon(t *testing.T) {
	statement := "SELECT * FROM users"
	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, 1, len(list))
	require.Equal(t, "SELECT * FROM users", list[0].Text)
	require.False(t, list[0].Empty)
}
