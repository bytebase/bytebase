package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func TestIssueCommentListFilter(t *testing.T) {
	const root = "projects/p/issues/101/issueComments/root"
	for _, tc := range []struct {
		filter string
		top    bool
		ids    []string
	}{
		{"", true, nil},
		{"root == null", true, nil},
		{"root == '" + root + "'", false, []string{"root"}},
		{"root in ['" + root + "']", false, []string{"root"}},
		{"root in []", false, []string{}},
	} {
		t.Run(tc.filter, func(t *testing.T) {
			find, err := store.GetIssueCommentListFilter(tc.filter, "p", 101)
			require.NoError(t, err)
			require.Equal(t, "p", find.ProjectID)
			require.Equal(t, int64(101), *find.IssueUID)
			require.Equal(t, tc.top, find.TopLevelOnly)
			if tc.ids == nil {
				require.Nil(t, find.ParentIDs)
			} else {
				require.Equal(t, tc.ids, *find.ParentIDs)
			}
		})
	}
	for _, filter := range []string{
		"true", "root", "root !=", "root != null", "name == 'x'", "null == root",
		"root == 1", "root == ''", "root == other", "root == 'root'",
		"root == 'projects/q/issues/101/issueComments/root'",
		"root == 'projects/p/issues/102/issueComments/root'",
		"root in null", "root in [null]", "root in [1]", "root in [other]",
		"root == null || root == '" + root + "'", "root.contains('x')",
	} {
		t.Run(filter, func(t *testing.T) {
			_, err := store.GetIssueCommentListFilter(filter, "p", 101)
			require.Error(t, err)
		})
	}
}
