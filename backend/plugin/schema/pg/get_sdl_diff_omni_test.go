package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/catalog"
)

func TestOmniGetSDLDiff_InitializationScenario(t *testing.T) {
	tests := []struct {
		name    string
		fromSDL string
		toSDL   string
		// What to check in the migration SQL.
		wantEmpty       bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:    "initialization_with_empty_previous_SDL",
			fromSDL: "",
			toSDL:   "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			wantContains: []string{
				"CREATE TABLE",
				"users",
			},
		},
		{
			name:    "initialization_with_complex_schema",
			fromSDL: "",
			toSDL:   "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);\nCREATE VIEW user_view AS SELECT * FROM users;",
			wantContains: []string{
				"CREATE TABLE",
				"users",
				"CREATE VIEW",
				"user_view",
			},
		},
		{
			name:    "non_initialization_normal_diff",
			fromSDL: "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			toSDL:   "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL, email TEXT);",
			wantContains: []string{
				"ALTER TABLE",
				"email",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)

			if tt.wantEmpty {
				require.Empty(t, sql)
				return
			}

			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
			for _, s := range tt.wantNotContains {
				require.NotContains(t, sql, s)
			}
		})
	}
}

func TestOmniGetSDLDiff_InitializationDiffCounts(t *testing.T) {
	tests := []struct {
		name              string
		fromSDL           string
		toSDL             string
		wantRelations     int
		wantFunctions     int
		wantSequences     int
		wantRelationNames []string
	}{
		{
			name:    "initialization_with_empty_previous_SDL",
			fromSDL: "",
			toSDL:   "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			// Table creation generates a relation entry.
			wantRelations:     1,
			wantFunctions:     0,
			wantRelationNames: []string{"users"},
		},
		{
			name:              "initialization_with_complex_schema",
			fromSDL:           "",
			toSDL:             "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);\nCREATE VIEW user_view AS SELECT * FROM users;",
			wantRelations:     2, // table + view
			wantFunctions:     0,
			wantRelationNames: []string{"users", "user_view"},
		},
		{
			name:              "non_initialization_normal_diff",
			fromSDL:           "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			toSDL:             "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL, email TEXT);",
			wantRelations:     1, // one table modified
			wantRelationNames: []string{"users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, err := catalog.LoadSDL(strings.TrimSpace(tt.fromSDL))
			require.NoError(t, err)
			to, err := catalog.LoadSDL(strings.TrimSpace(tt.toSDL))
			require.NoError(t, err)
			diff := catalog.Diff(from, to)

			require.Equal(t, tt.wantRelations, len(diff.Relations), "unexpected relation count")
			require.Equal(t, tt.wantFunctions, len(diff.Functions), "unexpected function count")

			var names []string
			for _, r := range diff.Relations {
				names = append(names, r.Name)
			}
			require.ElementsMatch(t, tt.wantRelationNames, names)
		})
	}
}

func TestOmniGetSDLDiff_MinimalChangesScenario(t *testing.T) {
	// Same table definition, no semantic change -> empty diff.
	sql := omniSDLMigration(t,
		"CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
		"CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
	)
	require.Empty(t, sql)
}
