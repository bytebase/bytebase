package aireview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFindings(t *testing.T) {
	t.Parallel()

	const finding = `{"title": "Create the index concurrently", "severity": "P1", "line": 2, "rule": "a long lock", "evidence": "orders has 52000000 rows", "fix": "Use CREATE INDEX CONCURRENTLY"}`
	want := []Finding{{
		Title:    "Create the index concurrently",
		Severity: SeverityP1,
		Line:     2,
		Rule:     "a long lock",
		Evidence: "orders has 52000000 rows",
		Fix:      "Use CREATE INDEX CONCURRENTLY",
	}}

	tests := []struct {
		name         string
		reply        string
		want         []Finding
		wantProblems []string
	}{
		{name: "bare object", reply: `{"findings": [` + finding + `]}`, want: want},
		{name: "code fences", reply: "```json\n{\"findings\": [" + finding + "]}\n```", want: want},
		{name: "sentence around the object", reply: `Here is my review: {"findings": [` + finding + `]} Let me know.`, want: want},
		{name: "empty list passes", reply: `{"findings": []}`},
		{name: "empty list in code fences passes", reply: "```json\n{\"findings\": []}\n```"},
		{
			name:         "a single word on the fence line does not pass",
			reply:        "```REJECTED\n{\"findings\": []}\n```",
			wantProblems: []string{"the reply has text around the JSON object; reply with only the JSON object"},
		},
		{
			// Chinese needs no spaces, so a sentence is one run of letters.
			name:         "a sentence without spaces on the fence line does not pass",
			reply:        "```发现严重问题删除语句没有条件\n{\"findings\": []}\n```",
			wantProblems: []string{"the reply has text around the JSON object; reply with only the JSON object"},
		},
		{
			name:         "prose on the fence line does not pass",
			reply:        "```I found a P0: the DELETE has no WHERE clause\n{\"findings\": []}\n```",
			wantProblems: []string{"the reply has text around the JSON object; reply with only the JSON object"},
		},
		{
			// Only the bare or fenced object is a verdict. Prose that mentions
			// the empty list must not pass the change.
			name:         "empty list inside prose does not pass",
			reply:        `I would answer {"findings": []} only if the table were small, but it has 52M rows.`,
			wantProblems: []string{"the reply has text around the JSON object; reply with only the JSON object"},
		},
		{
			name:  "lower case severity is accepted",
			reply: `{"findings": [{"title": "t", "severity": " p0 ", "line": 1, "rule": "r", "evidence": "e", "fix": "f"}]}`,
			want:  []Finding{{Title: "t", Severity: SeverityP0, Line: 1, Rule: "r", Evidence: "e", Fix: "f"}},
		},
		{name: "empty reply", reply: "  \n", wantProblems: []string{"the reply is empty"}},
		{name: "prose only", reply: "Let me check the orders table first.", wantProblems: []string{"the reply is not a JSON object"}},
		{name: "broken object", reply: `{"findings": [`, wantProblems: []string{"the reply is not a JSON object"}},
		{
			// An object without the key must not read as an empty list, which passes.
			name:         "object without the findings key",
			reply:        `{"error": "could not review"}`,
			wantProblems: []string{`the JSON object has no "findings" key`},
		},
		{
			name:  "every problem is reported at once",
			reply: `{"findings": [` + finding + `, {"title": "", "severity": "critical", "line": 9, "rule": "r", "evidence": "", "fix": "f"}]}`,
			wantProblems: []string{
				`findings[1].severity: "critical" is not one of P0, P1, P2`,
				"findings[1].line: 9 is outside the statements, which have lines 1 to 5",
				"findings[1].title: is empty",
				"findings[1].evidence: is empty",
			},
		},
		{
			name:         "missing line",
			reply:        `{"findings": [{"title": "t", "severity": "P2", "rule": "r", "evidence": "e", "fix": "f"}]}`,
			wantProblems: []string{"findings[0].line: 0 is outside the statements, which have lines 1 to 5"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, problems := parseFindings(tc.reply, 5)
			require.Equal(t, tc.wantProblems, problems)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestParseFindingsReportsTheErrorInsideTheFence(t *testing.T) {
	t.Parallel()

	// The model must hear about the wrong type of line, not about the fence.
	_, problems := parseFindings("```json\n{\"findings\": [{\"line\": \"3\"}]}\n```", 5)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "cannot unmarshal string")
	require.Contains(t, problems[0], "line")
}

func TestParseFindingsCapsTheProblemList(t *testing.T) {
	t.Parallel()

	reply := `{"findings": [`
	for i := range 6 {
		if i > 0 {
			reply += ","
		}
		reply += `{"severity": "P1", "line": 1}`
	}
	reply += `]}`

	_, problems := parseFindings(reply, 5)
	require.Len(t, problems, maxReportedProblems+1)
	require.Equal(t, "and 14 more problems", problems[maxReportedProblems])
}
