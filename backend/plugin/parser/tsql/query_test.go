package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExtractMSSQLNormalizedResourceListFromSelectStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      []base.SchemaResource
	}{
		{
			statement: `SELECT * FROM t1;SELECT * FROM t2;`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "t1",
				},
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "t2",
				},
			},
		},
		{
			statement: `SELECT * FROM myTable;SELECT * FROM MyTable;`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "MyTable",
				},
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "myTable",
				},
			},
		},
		{
			statement: `SELECT * FROM TableOne JOIN TableTwo ON TableOne.ID = TableTwo.ID;`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "TableOne",
				},
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "TableTwo",
				},
			},
		},
		{
			statement: `SELECT * FROM DatabaseA.dbo.[TableOne] JOIN DatabaseB.dbo.TableTwo ON DatabaseA.dbo.TableOne.ID = DatabaseB.dbo.TableTwo.ID;`,
			want: []base.SchemaResource{
				{
					Database: "DatabaseA",
					Schema:   "dbo",
					Table:    "TableOne",
				},
				{
					Database: "DatabaseB",
					Schema:   "dbo",
					Table:    "TableTwo",
				},
			},
		},
		{
			statement: `SELECT (SELECT MAX(b) FROM t1 WHERE c > 0);`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "t1",
				},
			},
		},
	}

	for _, test := range tests {
		res, err := ExtractResourceList("MyDB", "dbo", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, res, "for statement: %v", test.statement)
	}
}
