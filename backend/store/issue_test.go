package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalizeIssueLabels(t *testing.T) {
	got := CanonicalizeIssueLabels([]string{
		" release ",
		"bug",
		"",
		"bug ",
		"  feature",
		"\t",
		"release",
	})

	require.Equal(t, []string{"bug", "feature", "release"}, got)
}

func TestCanonicalizeIssueLabelsEmpty(t *testing.T) {
	require.Empty(t, CanonicalizeIssueLabels(nil))
	require.Empty(t, CanonicalizeIssueLabels([]string{"", " ", "\t"}))
}
