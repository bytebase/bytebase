package base

import "testing"

func TestTSQLRecognizeExplainType(t *testing.T) {
	testCases := []struct {
		spans     []*QuerySpan
		stmts     []string
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
			stmts: []string{
				"SELECT 1",
				"SET   SHOWPLAN_TEXT ON",
				"SELECT 2",
				"SET SHOWPLAN_TEXT        OFF",
				"SELECT 3",
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
		TSQLRecognizeExplainType(tc.spans, tc.stmts)
		for i := range tc.spans {
			if tc.spans[i].Type != tc.wantSpans[i].Type {
				t.Errorf("TSQLRecognizeExplainType() = %v, want %v", tc.spans[i].Type, tc.wantSpans[i].Type)
			}
		}
	}
}
