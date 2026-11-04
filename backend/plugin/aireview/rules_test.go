package aireview

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRules(t *testing.T) {
	t.Parallel()

	workspace := "Never delete rows from payment.\r\n\n  - Every foreign key needs an index.\n2) Flag a rewrite of a table over 1 million rows.\n"
	rules := Rules(workspace, "* Each new table requires an `id` column.")
	require.Equal(t, []Rule{
		{ID: "workspace-1", Text: "Never delete rows from payment."},
		{ID: "workspace-2", Text: "Every foreign key needs an index."},
		{ID: "workspace-3", Text: "Flag a rewrite of a table over 1 million rows."},
		{ID: "project-1", Text: "Each new table requires an `id` column."},
	}, rules[:4])
	require.Equal(t, checks, rules[4:], "the checks follow the policy lines")

	require.Equal(t, checks, Rules("", " \n"))
}

func TestParsePolicy(t *testing.T) {
	t.Parallel()

	rule := func(n int, text string) policyPart {
		return policyPart{Rule: Rule{ID: fmt.Sprintf("workspace-%d", n), Text: text}}
	}
	tests := []struct {
		name   string
		policy string
		want   []policyPart
	}{
		{
			name:   "a hard-wrapped sentence is one rule",
			policy: "Never delete payment rows unless\nthey have been archived.\nEvery foreign key needs an index.",
			want:   []policyPart{rule(1, "Never delete payment rows unless they have been archived."), rule(2, "Every foreign key needs an index.")},
		},
		{
			name:   "lines without a period are rules of their own when they start a sentence",
			policy: "No TRUNCATE in production\nEvery table needs a primary key",
			want:   []policyPart{rule(1, "No TRUNCATE in production"), rule(2, "Every table needs a primary key")},
		},
		{
			name:   "a lower-case line after an ended sentence starts a rule",
			policy: "Archive before you delete.\nnever drop a column that a view reads.",
			want:   []policyPart{rule(1, "Archive before you delete."), rule(2, "never drop a column that a view reads.")},
		},
		{
			name:   "a list item takes its wrapped lines",
			policy: "- Flag a rewrite of a table\n  over 1 million rows (or `events`)\n- Every new table has an id column",
			want:   []policyPart{rule(1, "Flag a rewrite of a table over 1 million rows (or `events`)"), rule(2, "Every new table has an id column")},
		},
		{
			name:   "a closing parenthesis after the period still ends the sentence",
			policy: "Keep invoices (seven years.)\nfor audits, archive them.",
			want:   []policyPart{rule(1, "Keep invoices (seven years.)"), rule(2, "for audits, archive them.")},
		},
		{
			name:   "a blank line and a heading end a rule",
			policy: "Never truncate a table\n\nwhose rows are kept.\n## Production only\nlocks are short.",
			want:   []policyPart{rule(1, "Never truncate a table"), rule(2, "whose rows are kept."), {Heading: "Production only"}, rule(3, "locks are short.")},
		},
		{
			name:   "text without case starts a rule on every line",
			policy: "不要删除支付记录\n每个外键都需要索引",
			want:   []policyPart{rule(1, "不要删除支付记录"), rule(2, "每个外键都需要索引")},
		},
		{name: "an empty heading is dropped", policy: "#  \n# \n", want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, parsePolicy("workspace", tc.policy))
		})
	}
}

func TestChecksHaveDistinctIDs(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for _, check := range checks {
		require.NotEmpty(t, check.Text)
		require.False(t, seen[check.ID], check.ID)
		seen[check.ID] = true
		require.NotContains(t, check.ID, "workspace-", "a check ID cannot pose as a policy line")
		require.NotContains(t, check.ID, "project-", "a check ID cannot pose as a policy line")
	}
}
