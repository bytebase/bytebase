package trino

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTrinoSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
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
