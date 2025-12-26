package trino

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type splitTestData struct {
	statement string
	want      resData
}

type resData struct {
	res []base.Statement
	err string
}

func TestTrinoSplitMultiSQL(t *testing.T) {
	// Split SQL functionality is implemented
	tests := []splitTestData{
		{
			statement: "SELECT * FROM users; SELECT * FROM orders;",
			want: resData{
				res: []base.Statement{
					{
						Text:  "SELECT * FROM users;",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 21},
					},
					{
						Text:     "SELECT * FROM orders;",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 1, Column: 22},
						End:      &storepb.Position{Line: 1, Column: 43},
					},
				},
			},
		},
		{
			statement: `
				-- This is a comment
				SELECT
					id,
					name
				FROM users
				WHERE status = 'active';

				/* This is a multi-line
				   comment */
				SELECT * FROM orders;`,
			want: resData{
				res: []base.Statement{
					{
						Text: `SELECT
					id,
					name
				FROM users
				WHERE status = 'active';`,
						BaseLine: 2,
						Start:    &storepb.Position{Line: 3, Column: 5},
						End:      &storepb.Position{Line: 7, Column: 29},
					},
					{
						Text:     `SELECT * FROM orders;`,
						BaseLine: 10,
						Start:    &storepb.Position{Line: 11, Column: 5},
						End:      &storepb.Position{Line: 11, Column: 26},
					},
				},
			},
		},
		{
			statement: `WITH orders_cte AS (
					SELECT * FROM orders
				)
				SELECT u.id, u.name, o.order_id
				FROM users u
				JOIN orders_cte o ON u.id = o.user_id;

				SELECT * FROM products;`,
			want: resData{
				res: []base.Statement{
					{
						Text: `WITH orders_cte AS (
					SELECT * FROM orders
				)
				SELECT u.id, u.name, o.order_id
				FROM users u
				JOIN orders_cte o ON u.id = o.user_id;`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 6, Column: 43},
					},
					{
						Text:     `SELECT * FROM products;`,
						BaseLine: 7,
						Start:    &storepb.Position{Line: 8, Column: 5},
						End:      &storepb.Position{Line: 8, Column: 28},
					},
				},
			},
		},
		{
			// Note: This test falls back to tokenizer path (parser fails with Chinese chars)
			// Tokenizer uses character-based column offsets
			statement: "SELECT * FROM 表名; INSERT INTO 表 VALUES (1);",
			want: resData{
				res: []base.Statement{
					{
						Text:  "SELECT * FROM 表名;",
						Range: &storepb.Range{Start: 0, End: 21}, // Byte offset 0-21
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 18}, // Character-based: 17 chars + 1
					},
					{
						Text:  "INSERT INTO 表 VALUES (1);",
						Range: &storepb.Range{Start: 22, End: 49}, // Byte offset 22-49
						Start: &storepb.Position{Line: 1, Column: 19},
						End:   &storepb.Position{Line: 1, Column: 44}, // Character-based exclusive
					},
				},
			},
		},
	}

	for _, test := range tests {
		// Split the SQL and check results
		res, err := SplitSQL(test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}

		require.Equal(t, len(test.want.res), len(res),
			"Expected %d statements, got %d", len(test.want.res), len(res))
		require.Equal(t, test.want.err, errStr,
			"Expected error '%s', got '%s'", test.want.err, errStr)

		// Check each statement's fields
		for i, want := range test.want.res {
			got := res[i]
			require.Equal(t, want.Text, got.Text, "Text mismatch at index %d", i)
			require.Equal(t, want.Start, got.Start, "Start mismatch at index %d", i)
			require.Equal(t, want.End, got.End, "End mismatch at index %d", i)
			if want.Range != nil {
				require.Equal(t, want.Range, got.Range, "Range mismatch at index %d", i)
			}
		}
	}
}

// TestSplitByParserAndTokenizerConsistency verifies that both splitByParser and
// splitByTokenizer return consistent results for the same valid SQL input.
func TestSplitByParserAndTokenizerConsistency(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "simple single statement",
			statement: "SELECT 1;",
		},
		{
			name:      "multiple statements on one line",
			statement: "SELECT 1; SELECT 2;",
		},
		{
			name:      "multi-line statement",
			statement: "SELECT\n  1;",
		},
		{
			name:      "multiple multi-line statements",
			statement: "SELECT\n  1;\nSELECT\n  2;",
		},
		{
			name:      "statement with trailing whitespace",
			statement: "SELECT 1;  \n  ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parserRes, parserErr := splitByParser(tc.statement)
			tokenizerRes, tokenizerErr := splitByTokenizer(tc.statement)

			// Both should succeed for valid SQL
			require.NoError(t, parserErr, "parser should not error")
			require.NoError(t, tokenizerErr, "tokenizer should not error")

			// Should return same number of statements
			require.Equal(t, len(parserRes), len(tokenizerRes),
				"statement count mismatch: parser=%d, tokenizer=%d", len(parserRes), len(tokenizerRes))

			// Compare each statement's fields
			for i := range parserRes {
				p := parserRes[i]
				tok := tokenizerRes[i]

				require.Equal(t, p.Text, tok.Text, "Text mismatch at index %d", i)
				require.Equal(t, p.BaseLine, tok.BaseLine, "BaseLine mismatch at index %d", i)
				require.Equal(t, p.Start, tok.Start, "Start mismatch at index %d", i)
				require.Equal(t, p.End, tok.End, "End mismatch at index %d", i)
				require.Equal(t, p.Range, tok.Range, "Range mismatch at index %d", i)
				require.Equal(t, p.Empty, tok.Empty, "Empty mismatch at index %d", i)
			}
		})
	}
}
