package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetViewDependencies(t *testing.T) {
	cases := []struct {
		name     string
		viewDef  string
		schema   string
		expected []string
	}{
		{
			name:     "simple",
			viewDef:  "CREATE VIEW [dbo].[v] AS SELECT id FROM [dbo].[a]",
			schema:   "dbo",
			expected: []string{"dbo.a"},
		},
		{
			name: "union two tables",
			viewDef: "CREATE VIEW [dbo].[v] AS " +
				"SELECT id FROM [dbo].[a] UNION SELECT id FROM [dbo].[b]",
			schema:   "dbo",
			expected: []string{"dbo.a", "dbo.b"},
		},
		{
			name: "union all three tables",
			viewDef: "CREATE VIEW [dbo].[v] AS " +
				"SELECT id FROM [dbo].[a] UNION ALL " +
				"SELECT id FROM [dbo].[b] UNION ALL " +
				"SELECT id FROM [dbo].[c]",
			schema:   "dbo",
			expected: []string{"dbo.a", "dbo.b", "dbo.c"},
		},
		{
			// GetQuerySpan with empty mock metadata cannot distinguish CTE
			// references from real table references, so cte names appear in
			// the dependency set. This is pre-existing behavior carried over
			// from the ANTLR implementation; the test pins it down.
			name: "cte",
			viewDef: "CREATE VIEW [dbo].[v] AS " +
				"WITH cte AS (SELECT id FROM [dbo].[a]) " +
				"SELECT id FROM cte",
			schema:   "dbo",
			expected: []string{"dbo.a", "dbo.cte"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getViewDependencies(tc.viewDef, tc.schema)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, got)
		})
	}
}
