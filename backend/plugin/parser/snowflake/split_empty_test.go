package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSplitSQL_EmptyStatementsDropped locks the omni-vs-legacy divergence:
// the legacy ANTLR splitter emitted bare ";" as a (non-empty!) statement, which
// the downstream skip guard `stmt.Empty || TrimSpace(Text)==""` would NOT skip
// (";" is neither empty nor blank) — i.e. it spuriously processed a bare
// semicolon as a real statement. omni's parser.Split produces no empty segments,
// so SplitSQL now drops them. Verified that every consumer skips empty/blank
// statements (trino.go, snowflake.go, redshift, doris, bigquery, tsql, ...), so
// nothing depends on the dropped statements. omni's behavior also matches MySQL
// and the merged Trino omni cutover.
func TestSplitSQL_EmptyStatementsDropped(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"SELECT 1;;SELECT 2;", 2},
		{"SELECT 1;  ;SELECT 2;", 2},
		{";", 0},
		{";;", 0},
		{"SELECT 1;", 1},
	}
	for _, c := range cases {
		got, err := SplitSQL(c.in)
		require.NoError(t, err, c.in)
		require.Len(t, got, c.want, "input %q", c.in)
		for i, st := range got {
			require.NotEmpty(t, st.Text, "stmt %d of %q should be a real statement", i, c.in)
		}
	}
}
