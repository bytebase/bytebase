package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQL(t *testing.T) {
	statement := `SELECT * FROM users;
	SELECT * FROM orders;
	SELECT * FROM products;`
	want := []base.Statement{
		{
			Text:  "SELECT * FROM users;",
			Range: &storepb.Range{Start: 0, End: 20},
			Start: &storepb.Position{Line: 1, Column: 1},
			End:   &storepb.Position{Line: 1, Column: 20},
			Empty: false,
		},
		{
			Text:  "\n\tSELECT * FROM orders;",
			Range: &storepb.Range{Start: 20, End: 43},
			Start: &storepb.Position{Line: 2, Column: 2},
			End:   &storepb.Position{Line: 2, Column: 22},
			Empty: false,
		},
		{
			Text:  "\n\tSELECT * FROM products;",
			Range: &storepb.Range{Start: 43, End: 68},
			Start: &storepb.Position{Line: 3, Column: 2},
			End:   &storepb.Position{Line: 3, Column: 24},
			Empty: false,
		},
	}

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, want, list)
}

func TestSplitSQLSingleStatement(t *testing.T) {
	statement := "SELECT * FROM users;"
	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, 1, len(list))
	require.Equal(t, "SELECT * FROM users;", list[0].Text)
	require.Equal(t, 0, list[0].GetBaseLine())
	require.False(t, list[0].Empty)
}
