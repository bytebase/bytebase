package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithSearchPath(t *testing.T) {
	statement := WithSearchPath("DELETE FROM t;", []string{"app", `we"ird`})
	require.Equal(t, "SET LOCAL search_path TO \"app\", \"we\"\"ird\";\nDELETE FROM t;", statement)

	setup, rest := SplitSearchPath(statement)
	require.Equal(t, `SET LOCAL search_path TO "app", "we""ird"`, setup)
	require.Equal(t, "DELETE FROM t;", rest)

	setup, rest = SplitSearchPath("DELETE FROM t;")
	require.Empty(t, setup)
	require.Equal(t, "DELETE FROM t;", rest)
}

func TestTSQLRecognizeExplainType(t *testing.T) {
	testCases := []struct {
		spans     []*QuerySpan
		stmts     []Statement
		wantSpans []*QuerySpan
	}{
		{
			spans: []*QuerySpan{
				{
					Type: Select,
				},
				{
					Type: Select,
				},
				{
					Type: Select,
				},
				{
					Type: Select,
				},
				{
					Type: Select,
				},
			},
			stmts: []Statement{
				{Text: "SELECT 1"},
				{Text: "SET   SHOWPLAN_TEXT ON"},
				{Text: "SELECT 2"},
				{Text: "SET SHOWPLAN_TEXT        OFF"},
				{Text: "SELECT 3"},
			},
			wantSpans: []*QuerySpan{
				{
					Type: Select,
				},
				{
					Type: Explain,
				},
				{
					Type: Explain,
				},
				{
					Type: Explain,
				},
				{
					Type: Select,
				},
			},
		},
	}

	for _, tc := range testCases {
		tsqlRecognizeExplainType(tc.spans, tc.stmts)
		for i := range tc.spans {
			if tc.spans[i].Type != tc.wantSpans[i].Type {
				t.Errorf("tsqlRecognizeExplainType() = %v, want %v", tc.spans[i].Type, tc.wantSpans[i].Type)
			}
		}
	}
}
