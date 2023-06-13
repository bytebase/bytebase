package snowflake

import "testing"

func TestNormalizeTableName(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{
			name: `TABLE_NAME`,
			want: `TABLE_NAME`,
		},
		{
			name: `"TABLE_NAME"`,
			want: `TABLE_NAME`,
		},
		{
			name: `table_name`,
			want: `TABLE_NAME`,
		},
		{
			name: `"table_name"`,
			want: `table_name`,
		},
		{
			name: `"table_name"""`,
			want: `table_name"`,
		},
		{
			name: `"😈😄"""`,
			want: `😈😄"`,
		},
		{
			name: `"DATABASE_NAME.SCHEMA_name.😈😄"""`,
			want: `DATABASE_NAME.SCHEMA_name.😈😄"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeIdentifierName(tc.name)
			if got != tc.want {
				t.Errorf("normalizeTableName() = %v, want %v", got, tc.want)
			}
		})
	}
}
