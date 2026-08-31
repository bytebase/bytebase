package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/qb"
)

func TestEscapeLikePattern(t *testing.T) {
	// `%` and `_` are wildcards to LIKE. Left raw, a search for a statement
	// holding either matched rows the user never asked for.
	require.Equal(t, `100\% off`, escapeLikePattern("100% off"))
	require.Equal(t, `a\_b`, escapeLikePattern("a_b"))
	require.Equal(t, `a\\b`, escapeLikePattern(`a\b`))
	require.Equal(t, "select 1", escapeLikePattern("select 1"))
	require.Equal(t, `%100\% off%`, containsPattern("100% off"))
}

func TestContainsEscapesLikeWildcards(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantArg string
		build   func(string) (*qb.Query, error)
	}{
		{
			name:    "database name",
			filter:  `name.contains("a_b")`,
			wantArg: `%a\_b%`,
			build:   func(f string) (*qb.Query, error) { return GetListDatabaseFilter("default", f) },
		},
		{
			name:    "instance name",
			filter:  `name.contains("a_b")`,
			wantArg: `%a\_b%`,
			build:   GetListInstanceFilter,
		},
		{
			name:    "project name",
			filter:  `name.contains("a_b")`,
			wantArg: `%a\_b%`,
			build:   func(f string) (*qb.Query, error) { return GetListProjectFilter("default", f) },
		},
		{
			name:    "plan title",
			filter:  `title.contains("100% off")`,
			wantArg: `%100\% off%`,
			build:   GetListPlanFilter,
		},
		{
			name:    "access grant query",
			filter:  `query.contains("WHERE x LIKE '%a%'")`,
			wantArg: `%WHERE x LIKE '\%a\%'%`,
			build:   GetListAccessGrantFilter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := tt.build(tt.filter)
			require.NoError(t, err)
			sql, args, err := query.ToSQL()
			require.NoError(t, err)
			require.Contains(t, sql, `ESCAPE '\'`)
			require.Equal(t, []any{tt.wantArg}, args)
		})
	}
}

func TestLabelFilterAcceptsIndexSyntax(t *testing.T) {
	// Label keys allow dashes (`^[a-z][a-z0-9_-]{0,62}$`). CEL parses
	// `labels.cost-center` as subtraction, so the dotted form never reaches the
	// store — index syntax is the only one that carries such a key. The key is
	// bound rather than interpolated, since index syntax admits any string.
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
		build    func(string) (*qb.Query, error)
	}{
		{
			name:     "database, dashed key",
			filter:   `labels["cost-center"] == "eng"`,
			wantSQL:  "(db.metadata->'labels'->>$1::text = $2)",
			wantArgs: []any{"cost-center", "eng"},
			build:    func(f string) (*qb.Query, error) { return GetListDatabaseFilter("default", f) },
		},
		{
			name:     "database, dotted key still works",
			filter:   `labels.environment == "prod"`,
			wantSQL:  "(db.metadata->'labels'->>$1::text = $2)",
			wantArgs: []any{"environment", "prod"},
			build:    func(f string) (*qb.Query, error) { return GetListDatabaseFilter("default", f) },
		},
		{
			name:     "database, key is bound not interpolated",
			filter:   `labels["a' OR '1'='1"] == "x"`,
			wantSQL:  "(db.metadata->'labels'->>$1::text = $2)",
			wantArgs: []any{"a' OR '1'='1", "x"},
			build:    func(f string) (*qb.Query, error) { return GetListDatabaseFilter("default", f) },
		},
		{
			name:     "instance, dashed key in a list",
			filter:   `labels["cost-center"] in ["eng", "ops"]`,
			wantSQL:  "(instance.metadata->'labels'->>$1::text = ANY($2))",
			wantArgs: []any{"cost-center", []any{"eng", "ops"}},
			build:    GetListInstanceFilter,
		},
		{
			name:     "project, dashed key",
			filter:   `labels["cost-center"] == "eng"`,
			wantSQL:  "(project.setting->'labels'->>$1::text = $2)",
			wantArgs: []any{"cost-center", "eng"},
			build:    func(f string) (*qb.Query, error) { return GetListProjectFilter("default", f) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := tt.build(tt.filter)
			require.NoError(t, err)
			sql, args, err := query.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}
