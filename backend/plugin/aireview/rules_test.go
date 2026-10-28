package aireview

import (
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
