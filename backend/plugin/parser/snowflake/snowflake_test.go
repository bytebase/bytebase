package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnowSqlExtractOrdinaryIdentifier(t *testing.T) {
	testCases := []struct {
		description string
		name        string
		want        string
	}{
		{
			description: "Should convert object name to uppercase if it is not quoted",
			name:        `table_name`,
			want:        `TABLE_NAME`,
		},
		{
			description: "Should **NOT** convert object name to uppercase if it is quoted",
			name:        `"table_name"`,
			want:        `table_name`,
		},
		{
			description: `Should convert '""' to '"' if it is quoted`,
			name:        `"table_name"""`,
			want:        `table_name"`,
		},
		{
			description: `Should be fine with unicode characters`,
			name:        `"😈😄"""`,
			want:        `😈😄"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractSnowSQLOrdinaryIdentifier(tc.name)
			require.Equal(t, tc.want, got, tc.description)
		})
	}
}

func TestParseSnowSQL(t *testing.T) {
	testCase := []struct {
		sql string
		err string
	}{
		{
			sql: `SELECT t.a, t.b FRO table_name t;`,
			err: "Syntax error at line 1:31 \nrelated text: SELECT t.a, t.b FRO table_name t",
		},
		{
			sql: `SELECT t.a, t.b FROM table_name }t;`,
			err: "Syntax error at line 1:32 \nrelated text: SELECT t.a, t.b FROM table_name }",
		},
	}

	for _, tc := range testCase {
		_, err := ParseSnowSQL(tc.sql)
		if tc.err != "" {
			require.EqualError(t, err, tc.err, tc.sql)
		} else {
			require.NoError(t, err, tc.sql)
		}
	}
}
