package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/catalog"
)

func TestOmniViewSDLDiff(t *testing.T) {
	tests := []struct {
		name         string
		fromSDL      string
		toSDL        string
		wantContains []string
		wantEmpty    bool
		// Diff-level assertions on Relations (views are relations in omni).
		wantViewAdd  int
		wantViewDrop int
	}{
		{
			name:    "Create new view",
			fromSDL: ``,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE VIEW active_users AS
SELECT id, name
FROM users
WHERE active = true;`,
			wantContains: []string{"CREATE VIEW", "active_users"},
			wantViewAdd:  1,
		},
		{
			name: "Drop view",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE VIEW active_users AS
SELECT id, name
FROM users
WHERE active = true;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);`,
			wantContains: []string{"DROP VIEW", "active_users"},
			wantViewDrop: 1,
		},
		{
			name: "Modify view definition",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE VIEW active_users AS
SELECT id, name
FROM users
WHERE active = true;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE VIEW active_users AS
SELECT id, name, 'active' as status
FROM users
WHERE active = true;`,
			wantContains: []string{"active_users"},
		},
		{
			name: "No changes to view",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE VIEW active_users AS
SELECT id, name
FROM users
WHERE active = true;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE VIEW active_users AS
SELECT id, name
FROM users
WHERE active = true;`,
			wantEmpty: true,
		},
		{
			name: "Multiple views with different changes",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    role TEXT
);

CREATE VIEW all_users AS
SELECT * FROM users;

CREATE VIEW active_users AS
SELECT id, name FROM users WHERE active = true;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    role TEXT
);

CREATE VIEW all_users AS
SELECT * FROM users;

CREATE VIEW admin_users AS
SELECT id, name FROM users WHERE role = 'admin';`,
			wantContains: []string{"admin_users", "active_users"},
			wantViewAdd:  1, // admin_users
			wantViewDrop: 1, // active_users
		},
		{
			name: "Schema-qualified view names",
			fromSDL: `
CREATE SCHEMA test_schema;
CREATE TABLE test_schema.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);`,
			toSDL: `
CREATE SCHEMA test_schema;
CREATE TABLE test_schema.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);

CREATE VIEW test_schema.product_summary AS
SELECT id, name FROM test_schema.products;`,
			wantContains: []string{"CREATE VIEW", "product_summary"},
			wantViewAdd:  1,
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

			// Verify diff-level view counts when specified.
			if tt.wantViewAdd > 0 || tt.wantViewDrop > 0 {
				from, err := catalog.LoadSDL(strings.TrimSpace(tt.fromSDL))
				require.NoError(t, err)
				to, err := catalog.LoadSDL(strings.TrimSpace(tt.toSDL))
				require.NoError(t, err)
				diff := catalog.Diff(from, to)

				addCount, dropCount := 0, 0
				for _, r := range diff.Relations {
					switch r.Action {
					case catalog.DiffAdd:
						// Only count views (relations with To that are views).
						if r.To != nil && r.To.RelKind == 'v' {
							addCount++
						}
					case catalog.DiffDrop:
						if r.From != nil && r.From.RelKind == 'v' {
							dropCount++
						}
					default:
					}
				}
				require.Equal(t, tt.wantViewAdd, addCount, "view ADD count mismatch")
				require.Equal(t, tt.wantViewDrop, dropCount, "view DROP count mismatch")
			}
		})
	}
}
