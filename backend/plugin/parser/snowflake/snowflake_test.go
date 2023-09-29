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
