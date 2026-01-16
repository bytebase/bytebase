package base

import "testing"

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
