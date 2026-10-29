package aireview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReply(t *testing.T) {
	t.Parallel()

	const finding = `{"title": "Create the index concurrently", "severity": "P1", "line": 2, "rule": "blocks", "reason": "r", "evidence": ["orders has 52000000 rows"], "fix": "Use CREATE INDEX CONCURRENTLY"}`
	want := []Finding{{
		Title:    "Create the index concurrently",
		Severity: SeverityP1,
		Line:     2,
		Rule:     "blocks",
		Kind:     "blocks",
		Reason:   "r",
		Evidence: []string{"orders has 52000000 rows"},
		Fix:      "Use CREATE INDEX CONCURRENTLY",
	}}

	tests := []struct {
		name         string
		reply        string
		want         []Finding
		wantNotes    []string
		wantProblems []string
		// wantProblemContains matches json/v2 messages by their stable part,
		// because the wording differs between Go patch versions.
		wantProblemContains []string
	}{
		{name: "bare object", reply: `{"findings": [` + finding + `]}`, want: want},
		{name: "code fences", reply: "```json\n{\"findings\": [" + finding + "]}\n```", want: want},
		{name: "sentence around the object", reply: `Here is my review: {"findings": [` + finding + `]} Let me know.`, want: want},
		{name: "empty list passes", reply: `{"findings": []}`},
		{name: "empty list in code fences passes", reply: "```json\n{\"findings\": []}\n```"},
		{
			name:      "notes ride along and are not findings",
			reply:     `{"findings": [], "notes": ["row count of orders is not synced", " ", ""]}`,
			wantNotes: []string{"row count of orders is not synced"},
		},
		{
			// A model that cannot review sometimes invents a key for it. Dropping
			// the key would turn its report into a pass.
			name:                "unknown top level key is rejected",
			reply:               `{"findings": [], "error": "could not review"}`,
			wantProblemContains: []string{`unknown object member name "error"`},
		},
		{
			// json/v1 would let the second key erase the first and pass the change.
			name:                "duplicate findings key is rejected",
			reply:               `{"findings": [` + finding + `], "findings": []}`,
			wantProblemContains: []string{`duplicate object member name "findings"`},
		},
		{
			name:                "unknown key inside a finding is rejected",
			reply:               `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": "fails", "reason": "r", "evidence": ["e"], "fix": "f", "confidence": 0.9}]}`,
			wantProblemContains: []string{`unknown object member name "confidence" within "/findings/0"`},
		},
		{
			name:                "a second object after the first is rejected",
			reply:               `{"findings": []} {"findings": []}`,
			wantProblemContains: []string{"invalid character '{' after top-level value"},
		},
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
			reply: `{"findings": [{"title": "t", "severity": " p0 ", "line": 1, "rule": "fails", "reason": "r", "evidence": ["e"], "fix": "f"}]}`,
			want:  []Finding{{Title: "t", Severity: SeverityP0, Line: 1, Rule: "fails", Kind: "fails", Reason: "r", Evidence: []string{"e"}, Fix: "f"}},
		},
		{
			name:  "a rule ID copied with its backticks is accepted",
			reply: `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": " ` + "`BLOCKS`" + ` ", "reason": "r", "evidence": ["e"], "fix": "f"}]}`,
			want:  []Finding{{Title: "t", Severity: SeverityP1, Line: 1, Rule: "blocks", Kind: "blocks", Reason: "r", Evidence: []string{"e"}, Fix: "f"}},
		},
		{
			// A quoted sentence is not an ID, and findings merge across
			// databases by their rule.
			name:         "a rule that is not an ID is rejected",
			reply:        `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": "a long lock", "reason": "r", "evidence": ["e"], "fix": "f"}]}`,
			wantProblems: []string{`findings[0].rule: "a long lock" is not the ID of a policy line or a check`},
		},
		{
			name:  "blank facts are dropped",
			reply: `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": "fails", "reason": "r", "evidence": ["  invoices has 4,603 rows ", " ", ""], "fix": "f"}]}`,
			want:  []Finding{{Title: "t", Severity: SeverityP1, Line: 1, Rule: "fails", Kind: "fails", Reason: "r", Evidence: []string{"invoices has 4,603 rows"}, Fix: "f"}},
		},
		{
			name:         "a finding without a reason is rejected",
			reply:        `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": "fails", "evidence": ["e"], "fix": "f"}]}`,
			wantProblems: []string{"findings[0].reason: is empty"},
		},
		{
			name:                "evidence as one string is rejected",
			reply:               `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": "fails", "reason": "r", "evidence": "e", "fix": "f"}]}`,
			wantProblemContains: []string{`"/findings/0/evidence"`},
		},
		{name: "empty reply", reply: "  \n", wantProblems: []string{"the reply is empty"}},
		{name: "prose only", reply: "Let me check the orders table first.", wantProblems: []string{"the reply is not a JSON object"}},
		{name: "broken object", reply: `{"findings": [`, wantProblems: []string{"the reply is not a JSON object"}},
		{
			// An object without the key must not read as an empty list, which passes.
			name:         "object without the findings key",
			reply:        `{"notes": ["could not review"]}`,
			wantProblems: []string{`the JSON object has no "findings" key`},
		},
		{
			name:  "every problem is reported at once",
			reply: `{"findings": [` + finding + `, {"title": "", "severity": "critical", "line": 9, "rule": "fails", "reason": "r", "evidence": [""], "fix": "f"}]}`,
			wantProblems: []string{
				`findings[1].severity: "critical" is not one of P0, P1, P2`,
				"findings[1].line: 9 is outside the statements, which have lines 1 to 5",
				"findings[1].title: is empty",
				"findings[1].evidence: is empty; list at least one fact",
			},
		},
		{
			name:         "missing line",
			reply:        `{"findings": [{"title": "t", "severity": "P2", "rule": "fails", "reason": "r", "evidence": ["e"], "fix": "f"}]}`,
			wantProblems: []string{"findings[0].line: 0 is outside the statements, which have lines 1 to 5"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, problems := parseReply(tc.reply, 5, Rules("", ""), false)
			if tc.wantProblemContains != nil {
				require.Nil(t, got)
				require.Len(t, problems, len(tc.wantProblemContains))
				for i, want := range tc.wantProblemContains {
					require.Contains(t, problems[i], want)
				}
				return
			}
			require.Equal(t, tc.wantProblems, problems)
			if tc.wantProblems != nil {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tc.want, got.Findings)
			require.Equal(t, tc.wantNotes, got.Notes)
		})
	}
}

func TestParseReplyAcceptsThePolicyLines(t *testing.T) {
	t.Parallel()

	rules := Rules("Never delete rows from payment.", "- Every new table has an id column.")
	reply := func(rule string, kind string) string {
		return `{"findings": [{"title": "t", "severity": "P1", "line": 1, "rule": "` + rule + `", "kind": "` + kind + `", "reason": "r", "evidence": ["e"], "fix": "f"}]}`
	}
	for _, tc := range []struct{ rule, kind, wantKind string }{
		{rule: "workspace-1", kind: "drops-data", wantKind: "drops-data"},
		{rule: "project-1", kind: " `OTHER` ", wantKind: KindOther},
		{rule: "fails", kind: "", wantKind: "fails"},
		{rule: "fails", kind: "blocks", wantKind: "fails"},
	} {
		got, problems := parseReply(reply(tc.rule, tc.kind), 5, rules, false)
		require.Empty(t, problems, tc)
		require.Equal(t, tc.rule, got.Findings[0].Rule)
		require.Equal(t, tc.wantKind, got.Findings[0].Kind, "a check is its own kind, and a policy rule takes the kind the reply names")
	}
	for _, tc := range []struct{ rule, kind, want string }{
		{rule: "workspace-2", kind: "blocks", want: `findings[0].rule: "workspace-2" is not the ID of a policy line or a check`},
		{rule: "workspace-1", kind: "", want: "findings[0].kind: is empty; a finding that cites a policy rule names the check for its kind of problem, or other"},
		{rule: "workspace-1", kind: "project-1", want: `findings[0].kind: "project-1" is not the ID of a check or other`},
	} {
		_, problems := parseReply(reply(tc.rule, tc.kind), 5, rules, false)
		require.Equal(t, []string{tc.want}, problems)
	}
}

func TestParseReplyPolicyRuleOfTwoKindsOnALine(t *testing.T) {
	t.Parallel()

	rules := Rules("Do not run destructive or blocking DDL.", "")
	finding := func(kind string) string {
		return `{"title": "t", "severity": "P1", "line": 1, "rule": "workspace-1", "kind": "` + kind + `", "reason": "r", "evidence": ["e"], "fix": "f"}`
	}
	got, problems := parseReply(`{"findings": [`+finding("drops-data")+`, `+finding("blocks")+`]}`, 5, rules, false)
	require.Empty(t, problems, "a broad policy rule broken two ways on a line is two findings")
	require.Len(t, got.Findings, 2)

	_, problems = parseReply(`{"findings": [`+finding("blocks")+`, `+finding("blocks")+`]}`, 5, rules, false)
	require.Equal(t, []string{`findings[1].kind: findings[0] already cites "workspace-1" as "blocks" on line 1; report one finding per rule and kind on a line, naming every cause in it`}, problems)
}

func TestParseReplyRepeatedRuleOnALine(t *testing.T) {
	t.Parallel()

	const reply = `{"findings": [
		{"title": "Remove the duplicate index", "severity": "P1", "line": 2, "rule": "fails", "reason": "r", "evidence": ["orders_idx exists"], "fix": "f"},
		{"title": "Drop the duplicate check", "severity": "P1", "line": 2, "rule": "fails", "reason": "r", "evidence": ["orders_check exists"], "fix": "f"},
		{"title": "Create orders_archive first", "severity": "P1", "line": 3, "rule": "fails", "reason": "r", "evidence": ["orders_archive does not exist"], "fix": "f"}
	]}`
	got, problems := parseReply(reply, 5, Rules("", ""), false)
	require.Nil(t, got)
	require.Equal(t, []string{`findings[1].rule: findings[0] already cites "fails" on line 2; report one finding per rule on a line, naming every cause in it`}, problems)

	got, problems = parseReply(reply, 5, Rules("", ""), true)
	require.Empty(t, problems, "a reply that can no longer be corrected keeps the repeat")
	require.Len(t, got.Findings, 3)
}

func TestParseReplyReportsTheErrorInsideTheFence(t *testing.T) {
	t.Parallel()

	// The model must hear about the wrong type of line, not about the fence.
	_, problems := parseReply("```json\n{\"findings\": [{\"line\": \"3\"}]}\n```", 5, Rules("", ""), false)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "unmarshal JSON string into Go int")
	require.Contains(t, problems[0], `"/findings/0/line"`)
}

func TestParseReplyCapsTheProblemList(t *testing.T) {
	t.Parallel()

	reply := `{"findings": [`
	for i := range 6 {
		if i > 0 {
			reply += ","
		}
		reply += `{"severity": "P1", "line": 1}`
	}
	reply += `]}`

	_, problems := parseReply(reply, 5, Rules("", ""), false)
	require.Len(t, problems, maxReportedProblems+1)
	require.Equal(t, "and 20 more problems", problems[maxReportedProblems])
}
