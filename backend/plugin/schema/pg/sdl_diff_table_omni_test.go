package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/catalog"
)

func TestOmniTableDiffScenarios(t *testing.T) {
	testCases := []struct {
		name            string
		fromSDL         string
		toSDL           string
		wantContains    []string
		wantNotContains []string
		wantEmpty       bool
		// Diff-level assertions.
		wantAddCount    int
		wantModifyCount int
		wantDropCount   int
		wantNames       []string
	}{
		{
			name:    "Create new table",
			fromSDL: "",
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			wantContains: []string{"CREATE TABLE", "users"},
			wantAddCount: 1,
			wantNames:    []string{"users"},
		},
		{
			name: "Drop existing table",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			toSDL:         "",
			wantContains:  []string{"DROP TABLE", "users"},
			wantDropCount: 1,
			wantNames:     []string{"users"},
		},
		{
			name: "Modify existing table",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255)
			);`,
			wantContains:    []string{"ALTER TABLE", "email"},
			wantModifyCount: 1,
			wantNames:       []string{"users"},
		},
		{
			name: "Mixed operations",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);

			CREATE TABLE old_table (
				id SERIAL PRIMARY KEY
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255)
			);

			CREATE TABLE products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			wantContains:    []string{"products", "old_table", "email"},
			wantAddCount:    1, // products
			wantModifyCount: 1, // users
			wantDropCount:   1, // old_table
			wantNames:       []string{"users", "products", "old_table"},
		},
		{
			name: "Schema qualified tables",
			fromSDL: `CREATE TABLE public.users (
				id SERIAL PRIMARY KEY
			);`,
			toSDL: `CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);

			CREATE SCHEMA admin;
			CREATE TABLE admin.settings (
				key VARCHAR(255) PRIMARY KEY,
				value TEXT
			);`,
			wantContains:    []string{"settings", "name"},
			wantAddCount:    1, // admin.settings
			wantModifyCount: 1, // public.users
			wantNames:       []string{"users", "settings"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tc.fromSDL, tc.toSDL)

			if tc.wantEmpty {
				require.Empty(t, sql)
				return
			}

			for _, s := range tc.wantContains {
				require.Contains(t, sql, s)
			}
			for _, s := range tc.wantNotContains {
				require.NotContains(t, sql, s)
			}

			// Verify diff counts.
			from, err := catalog.LoadSDL(strings.TrimSpace(tc.fromSDL))
			require.NoError(t, err)
			to, err := catalog.LoadSDL(strings.TrimSpace(tc.toSDL))
			require.NoError(t, err)
			diff := catalog.Diff(from, to)

			addCount, modifyCount, dropCount := 0, 0, 0
			var names []string
			for _, r := range diff.Relations {
				names = append(names, r.Name)
				switch r.Action {
				case catalog.DiffAdd:
					addCount++
				case catalog.DiffModify:
					modifyCount++
				case catalog.DiffDrop:
					dropCount++
				default:
				}
			}

			require.Equal(t, tc.wantAddCount, addCount, "ADD count mismatch")
			require.Equal(t, tc.wantModifyCount, modifyCount, "MODIFY count mismatch")
			require.Equal(t, tc.wantDropCount, dropCount, "DROP count mismatch")
			require.ElementsMatch(t, tc.wantNames, names, "relation names mismatch")
		})
	}
}
