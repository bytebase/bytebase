package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExtractOracleResourceList(t *testing.T) {
	tests := []struct {
		statement string
		expected  []base.SchemaResource
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T1",
				},
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1;",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Schema:   "SCHEMA1",
					Table:    "T1",
				},
				{
					Database: "DB",
					Schema:   "SCHEMA2",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			expected: []base.SchemaResource{
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T1",
				},
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T2",
				},
			},
		},
	}

	for _, test := range tests {
		resources, err := ExtractResourceList("DB", "ROOT", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
