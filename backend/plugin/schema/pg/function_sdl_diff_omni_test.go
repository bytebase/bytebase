package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/catalog"
)

func TestOmniFunctionSDLDiff(t *testing.T) {
	tests := []struct {
		name            string
		fromSDL         string
		toSDL           string
		wantContains    []string
		wantEmpty       bool
		wantAddCount    int
		wantDropCount   int
		wantModifyCount int
	}{
		{
			name:    "Create new function",
			fromSDL: ``,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			wantContains: []string{"CREATE FUNCTION", "get_user_count"},
			wantAddCount: 1,
		},
		{
			name: "Drop function",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);`,
			wantContains:  []string{"DROP FUNCTION", "get_user_count"},
			wantDropCount: 1,
		},
		{
			name: "Modify function body",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users WHERE active = true);
END;
$$ LANGUAGE plpgsql;`,
			wantContains:    []string{"get_user_count"},
			wantModifyCount: 1,
		},
		{
			name: "No changes to function",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			wantEmpty: true,
		},
		{
			name: "Multiple functions with different changes",
			fromSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_all_users()
RETURNS SETOF users AS $$
BEGIN
    RETURN QUERY SELECT * FROM users;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			toSDL: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_all_users()
RETURNS SETOF users AS $$
BEGIN
    RETURN QUERY SELECT * FROM users;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_admin_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users WHERE role = 'admin');
END;
$$ LANGUAGE plpgsql;`,
			wantContains:  []string{"get_admin_count", "get_user_count"},
			wantAddCount:  1, // get_admin_count
			wantDropCount: 1, // get_user_count
		},
		{
			name: "Schema-qualified function names",
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

CREATE OR REPLACE FUNCTION test_schema.get_product_count()
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM test_schema.products);
END;
$$ LANGUAGE plpgsql;`,
			wantContains: []string{"CREATE FUNCTION", "get_product_count"},
			wantAddCount: 1,
		},
		{
			name: "Function with complex signature",
			fromSDL: `
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    amount DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW()
);`,
			toSDL: `
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    amount DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION calculate_total(
    p_start_date DATE,
    p_end_date DATE,
    p_discount_rate DECIMAL DEFAULT 0.0
)
RETURNS DECIMAL(10,2) AS $$
BEGIN
    RETURN (
        SELECT COALESCE(SUM(amount * (1 - p_discount_rate)), 0)
        FROM orders
        WHERE created_at::DATE BETWEEN p_start_date AND p_end_date
    );
END;
$$ LANGUAGE plpgsql;`,
			wantContains: []string{"CREATE FUNCTION", "calculate_total"},
			wantAddCount: 1,
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

			// Verify diff counts.
			from, err := catalog.LoadSDL(strings.TrimSpace(tt.fromSDL))
			require.NoError(t, err)
			to, err := catalog.LoadSDL(strings.TrimSpace(tt.toSDL))
			require.NoError(t, err)
			diff := catalog.Diff(from, to)

			addCount, dropCount, modifyCount := 0, 0, 0
			for _, f := range diff.Functions {
				switch f.Action {
				case catalog.DiffAdd:
					addCount++
				case catalog.DiffDrop:
					dropCount++
				case catalog.DiffModify:
					modifyCount++
				default:
				}
			}

			require.Equal(t, tt.wantAddCount, addCount, "function ADD count mismatch")
			require.Equal(t, tt.wantDropCount, dropCount, "function DROP count mismatch")
			require.Equal(t, tt.wantModifyCount, modifyCount, "function MODIFY count mismatch")
		})
	}
}

func TestOmniSameNameFunctionsWithDifferentSignatures(t *testing.T) {
	tests := []struct {
		name            string
		fromSDL         string
		toSDL           string
		wantAddCount    int
		wantDropCount   int
		wantModifyCount int
		description     string
	}{
		{
			name: "Add second function with same name but different signature",
			fromSDL: `
CREATE FUNCTION "public"."my_procedure"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Function with one parameter: %', p_id;
END;
$$;`,
			toSDL: `
CREATE FUNCTION "public"."my_procedure"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Function with one parameter: %', p_id;
END;
$$;

CREATE FUNCTION "public"."my_procedure"(p_id integer, p_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Function with two parameters: % - %', p_id, p_name;
END;
$$;`,
			wantAddCount:  1,
			wantDropCount: 0,
			description:   "Should add the new function without dropping the old one",
		},
		{
			name: "Three overloaded functions with same name",
			fromSDL: `
CREATE FUNCTION "public"."calculate"(x integer) RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT x * 2;
$$;`,
			toSDL: `
CREATE FUNCTION "public"."calculate"(x integer) RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT x * 2;
$$;

CREATE FUNCTION "public"."calculate"(x integer, y integer) RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT x + y;
$$;

CREATE FUNCTION "public"."calculate"(x numeric, y numeric, z numeric) RETURNS numeric
    LANGUAGE sql
    AS $$
    SELECT x + y + z;
$$;`,
			wantAddCount:  2,
			wantDropCount: 0,
			description:   "Should add two new overloaded functions",
		},
		{
			name: "Remove one overloaded function, keep others",
			fromSDL: `
CREATE FUNCTION "public"."process"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process one param';
END;
$$;

CREATE FUNCTION "public"."process"(p_id integer, p_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process two params';
END;
$$;

CREATE FUNCTION "public"."process"(p_id integer, p_name text, p_email text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process three params';
END;
$$;`,
			toSDL: `
CREATE FUNCTION "public"."process"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process one param';
END;
$$;

CREATE FUNCTION "public"."process"(p_id integer, p_name text, p_email text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process three params';
END;
$$;`,
			wantAddCount:  0,
			wantDropCount: 1,
			description:   "Should drop only the middle function (2 params), keep others",
		},
		{
			name: "Modify one overloaded function, keep others unchanged",
			fromSDL: `
CREATE FUNCTION "public"."get_info"(p_id integer) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'User ID: ' || p_id::text;
$$;

CREATE FUNCTION "public"."get_info"(p_name text) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'User Name: ' || p_name;
$$;`,
			toSDL: `
CREATE FUNCTION "public"."get_info"(p_id integer) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'Updated User ID: ' || p_id::text;
$$;

CREATE FUNCTION "public"."get_info"(p_name text) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'User Name: ' || p_name;
$$;`,
			wantAddCount:    0,
			wantDropCount:   0,
			wantModifyCount: 1,
			description:     "Should modify only the first function, keep second unchanged",
		},
		{
			name: "Procedures with same name but different signatures",
			fromSDL: `
CREATE PROCEDURE "public"."update_status"(p_id integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET status = 'active' WHERE id = p_id;
END;
$$;`,
			toSDL: `
CREATE PROCEDURE "public"."update_status"(p_id integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET status = 'active' WHERE id = p_id;
END;
$$;

CREATE PROCEDURE "public"."update_status"(p_id integer, p_new_status text)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET status = p_new_status WHERE id = p_id;
END;
$$;`,
			wantAddCount:  1,
			wantDropCount: 0,
			description:   "Should add the new procedure without dropping the old one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			t.Logf("Description: %s", tt.description)
			t.Logf("Migration SQL:\n%s", sql)

			from, err := catalog.LoadSDL(strings.TrimSpace(tt.fromSDL))
			require.NoError(t, err)
			to, err := catalog.LoadSDL(strings.TrimSpace(tt.toSDL))
			require.NoError(t, err)
			diff := catalog.Diff(from, to)

			addCount, dropCount, modifyCount := 0, 0, 0
			for _, f := range diff.Functions {
				switch f.Action {
				case catalog.DiffAdd:
					addCount++
				case catalog.DiffDrop:
					dropCount++
				case catalog.DiffModify:
					modifyCount++
				default:
				}
			}

			require.Equal(t, tt.wantAddCount, addCount, "function ADD count mismatch")
			require.Equal(t, tt.wantDropCount, dropCount, "function DROP count mismatch")
			require.Equal(t, tt.wantModifyCount, modifyCount, "function MODIFY count mismatch")

			// Ensure no unexpected drops (BYT-7994 regression check).
			if dropCount > tt.wantDropCount {
				t.Errorf("BUG: Found %d DROP action(s), expected %d. "+
					"Functions with same name but different signatures should be identified separately.",
					dropCount, tt.wantDropCount)
			}
		})
	}
}
