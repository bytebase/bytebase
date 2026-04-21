package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestIsSystemResource(t *testing.T) {
	cases := []struct {
		name                string
		database            string
		ignoreCaseSensitive bool
		want                bool
	}{
		// information_schema — reserved by MySQL server, always case-insensitive.
		{"info_schema lowercase, case-sensitive instance", "information_schema", false, true},
		{"info_schema uppercase, case-sensitive instance", "INFORMATION_SCHEMA", false, true},
		{"info_schema mixed case, case-sensitive instance", "Information_Schema", false, true},
		{"info_schema uppercase, case-insensitive instance", "INFORMATION_SCHEMA", true, true},

		// performance_schema — reserved by MySQL server, always case-insensitive.
		{"perf_schema lowercase, case-sensitive instance", "performance_schema", false, true},
		{"perf_schema uppercase, case-sensitive instance", "PERFORMANCE_SCHEMA", false, true},

		// mysql — on-disk schema, only case-insensitive when the instance is
		// (lower_case_table_names != 0). Case-sensitive instances can host a
		// separate user schema named e.g. `MySQL`, which must NOT be treated
		// as a system schema.
		{"mysql lowercase, case-sensitive instance", "mysql", false, true},
		{"mysql uppercase, case-sensitive instance (user schema)", "MYSQL", false, false},
		{"mysql mixed case, case-sensitive instance (user schema)", "MySQL", false, false},
		{"mysql uppercase, case-insensitive instance", "MYSQL", true, true},
		{"mysql mixed case, case-insensitive instance", "MySQL", true, true},

		// Arbitrary user schemas are never system.
		{"user schema lowercase", "byt9309_db", false, false},
		{"user schema uppercase", "BYT9309_DB", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSystemResource(base.ColumnResource{Database: tc.database}, tc.ignoreCaseSensitive)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestIsSystemResourceBYT9309 pins the specific regression: uppercase
// `INFORMATION_SCHEMA` on a case-sensitive instance must be recognized as
// a system schema, so `isMixedQuery` can short-circuit and prevent the
// span extractor from flagging it as a not-found user database — which
// previously cascaded into a nil-pointer panic in the schema syncer.
func TestIsSystemResourceBYT9309(t *testing.T) {
	set := base.SourceColumnSet{
		{Database: "INFORMATION_SCHEMA", Table: "PROCESSLIST"}: true,
	}
	allSystem, mixed := isMixedQuery(set, false /* case-sensitive instance */)
	require.True(t, allSystem, "INFORMATION_SCHEMA.PROCESSLIST should be treated as system on a case-sensitive instance")
	require.False(t, mixed)
}
