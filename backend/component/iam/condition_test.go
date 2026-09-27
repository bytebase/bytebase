package iam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConditionScopesResources(t *testing.T) {
	testCases := []struct {
		name       string
		expression string
		want       bool
		wantErr    bool
	}{
		{name: "unconditional", expression: "", want: false},
		{
			name:       "expiry only",
			expression: `request.time < timestamp("2099-01-01T00:00:00Z")`,
			want:       false,
		},
		{name: "database equality", expression: `resource.database == "instances/i/databases/d"`, want: true},
		{name: "database negated", expression: `resource.database != "instances/i/databases/d"`, want: true},
		{name: "database list", expression: `resource.database in ["a", "b"]`, want: true},
		{name: "environment", expression: `resource.environment_id == "prod"`, want: true},
		{name: "schema", expression: `resource.schema_name == "public"`, want: true},
		{name: "table", expression: `resource.table_name.startsWith("audit_")`, want: true},
		{
			name:       "scope plus expiry",
			expression: `resource.database == "instances/i/databases/d" && request.time < timestamp("2099-01-01T00:00:00Z")`,
			want:       true,
		},
		{
			// A macro hides the reference from a plain syntax-tree walk.
			name:       "scope inside a comprehension",
			expression: `["a", "b"].exists(d, d == resource.database)`,
			want:       true,
		},
		{name: "not an expression", expression: "this is not cel", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := conditionScopesResources(tc.expression)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
