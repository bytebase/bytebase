package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTiDBSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}

func TestTiDBSplitSQLDelimiterDirectives(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []string
	}{
		{
			name:      "common custom delimiter",
			statement: "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT delimiter FROM t;\nEND//\nDELIMITER ;\nCALL p();",
			want: []string{
				"CREATE PROCEDURE p()\nBEGIN\n  SELECT delimiter FROM t;\nEND",
				"CALL p();",
			},
		},
		{
			name:      "label is not directive",
			statement: "DELIMITER: LOOP\n  SELECT 1;\n  LEAVE DELIMITER;\nEND LOOP DELIMITER;",
			want:      []string{"DELIMITER: LOOP\n  SELECT 1;\n  LEAVE DELIMITER;\nEND LOOP DELIMITER;"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list, err := SplitSQL(test.statement)
			require.NoError(t, err)
			list = base.FilterEmptyStatements(list)

			require.Len(t, list, len(test.want))
			for i, stmt := range list {
				require.Equal(t, test.want[i], stmt.Text)
				require.Equal(t, test.statement[stmt.Range.Start:stmt.Range.End], stmt.Text)
			}
		})
	}
}
