package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateResourceFilter(t *testing.T) {
	tests := []struct {
		filter   string
		expected string
	}{
		{
			filter:   `tableExists("db1", "schema1", "table1")`,
			expected: `(instance_change_history.payload @> '{"changedResources":{"databases":[{"name":"db1", "schemas":[{"name":"schema1", "tables":[{"name":"table1"}]}]}]}}'::jsonb)`,
		},
		{
			filter: `
			tableExists("db1", "schema1", "table1") ||
			tableExists("db1", "schema2", "table2")
			`,
			expected: `((instance_change_history.payload @> '{"changedResources":{"databases":[{"name":"db1", "schemas":[{"name":"schema1", "tables":[{"name":"table1"}]}]}]}}'::jsonb) OR (instance_change_history.payload @> '{"changedResources":{"databases":[{"name":"db1", "schemas":[{"name":"schema2", "tables":[{"name":"table2"}]}]}]}}'::jsonb))`,
		},
		{
			filter: `
			(
				tableExists("db1", "schema1", "table1")
				&& tableExists("db1", "schema2", "table2")
			) || (
				tableExists("db2", "schema1", "table1")
			)
			`,
			expected: `((instance_change_history.payload @> '{"changedResources":{"databases":[{"name":"db1", "schemas":[{"name":"schema1", "tables":[{"name":"table1"}]}, {"name":"schema2", "tables":[{"name":"table2"}]}]}]}}'::jsonb) OR (instance_change_history.payload @> '{"changedResources":{"databases":[{"name":"db2", "schemas":[{"name":"schema1", "tables":[{"name":"table1"}]}]}]}}'::jsonb))`,
		},
	}

	for _, test := range tests {
		text, err := generateResourceFilter(test.filter)
		require.NoError(t, err)
		require.Equal(t, strings.ReplaceAll(test.expected, " ", ""), strings.ReplaceAll(text, " ", ""), test.filter)
	}
}
