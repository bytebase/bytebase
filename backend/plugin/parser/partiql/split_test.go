package partiql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQL(t *testing.T) {
	statement := `
SELECT * FROM my_table;
SELECT * FROM my_table;
foobar();`
	want := []base.SingleSQL{
		{
			Text:                 "\nSELECT * FROM my_table;",
			BaseLine:             0,
			FirstStatementLine:   1,
			FirstStatementColumn: 0,
			LastLine:             1,
			LastColumn:           22,
			Empty:                false,
		},
		{
			Text:                 "\nSELECT * FROM my_table;",
			BaseLine:             1,
			FirstStatementLine:   2,
			FirstStatementColumn: 0,
			LastLine:             2,
			LastColumn:           22,
			Empty:                false,
		},
		{
			Text:                 "\nfoobar();",
			BaseLine:             2,
			FirstStatementLine:   3,
			FirstStatementColumn: 0,
			LastLine:             3,
			LastColumn:           8,
			Empty:                false,
		},
	}

	list, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Equal(t, want, list)
}
