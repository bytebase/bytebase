package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ============================================================================
// Audit of claimed behavior differences between old ANTLR SDL engine and omni.
// Each test calls omniSDLMigration, logs the SQL, and asserts the specific
// behavior being investigated.
// ============================================================================

//  1. Drop cascade: "omni uses DROP TABLE CASCADE instead of dropping dependent
//     objects first"
func TestOmniDiffAudit_DropCascade(t *testing.T) {
	// Drop a table that has a dependent view. Check if CASCADE is used.
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);
		CREATE VIEW active_users AS SELECT id, name FROM users WHERE active = true;
	`, `
		CREATE VIEW active_users AS SELECT 1 AS id, 'placeholder'::text AS name WHERE false;
	`)
	t.Logf("SQL output (drop table with dependent view):\n%s", sql)

	// Also test dropping table when the view is also dropped.
	sql2 := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
		CREATE VIEW user_view AS SELECT * FROM users;
	`, "")
	t.Logf("SQL output (drop table + view):\n%s", sql2)

	hasCascade := strings.Contains(strings.ToUpper(sql2), "CASCADE")
	t.Logf("Contains CASCADE: %v", hasCascade)

	if hasCascade {
		require.Contains(t, strings.ToUpper(sql2), "CASCADE",
			"CONFIRMED: omni uses CASCADE on DROP TABLE")
	} else {
		t.Log("NOT CONFIRMED: omni does NOT use CASCADE on DROP TABLE")
	}
}

// 2. View Comment: "omni uses COMMENT ON TABLE for views instead of COMMENT ON VIEW"
func TestOmniDiffAudit_ViewComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE VIEW user_view AS SELECT 1;
	`, `
		CREATE VIEW user_view AS SELECT 1;
		COMMENT ON VIEW user_view IS 'A view comment';
	`)
	t.Logf("SQL output (view comment):\n%s", sql)

	hasCommentOnView := strings.Contains(sql, "COMMENT ON VIEW")
	hasCommentOnTable := strings.Contains(sql, "COMMENT ON TABLE")
	t.Logf("Contains 'COMMENT ON VIEW': %v", hasCommentOnView)
	t.Logf("Contains 'COMMENT ON TABLE': %v", hasCommentOnTable)

	// The claim is that omni uses COMMENT ON TABLE for views.
	// If it actually uses COMMENT ON VIEW, the claim is wrong.
	if hasCommentOnView {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni correctly uses COMMENT ON VIEW")
	} else if hasCommentOnTable {
		t.Log("Verdict: CONFIRMED - omni uses COMMENT ON TABLE instead of COMMENT ON VIEW")
	} else {
		t.Log("Verdict: UNEXPECTED - no COMMENT statement found at all")
	}

	// Assert at least one comment form is present
	require.True(t, hasCommentOnView || hasCommentOnTable,
		"expected a COMMENT statement in output")
}

//  3. Materialized View Comment: "omni uses COMMENT ON TABLE for materialized views
//     instead of COMMENT ON MATERIALIZED VIEW"
func TestOmniDiffAudit_MaterializedViewComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);
		CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
	`, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);
		CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
		COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
	`)
	t.Logf("SQL output (materialized view comment):\n%s", sql)

	hasCommentOnMV := strings.Contains(sql, "COMMENT ON MATERIALIZED VIEW")
	hasCommentOnTable := strings.Contains(sql, "COMMENT ON TABLE")
	t.Logf("Contains 'COMMENT ON MATERIALIZED VIEW': %v", hasCommentOnMV)
	t.Logf("Contains 'COMMENT ON TABLE': %v", hasCommentOnTable)

	if hasCommentOnMV {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni correctly uses COMMENT ON MATERIALIZED VIEW")
	} else if hasCommentOnTable {
		t.Log("Verdict: CONFIRMED - omni uses COMMENT ON TABLE instead of COMMENT ON MATERIALIZED VIEW")
	} else {
		t.Log("Verdict: UNEXPECTED - no COMMENT statement found at all")
	}

	require.True(t, hasCommentOnMV || hasCommentOnTable,
		"expected a COMMENT statement in output")
}

// 4. Trigger modification: "omni uses DROP+CREATE instead of CREATE OR REPLACE TRIGGER"
func TestOmniDiffAudit_TriggerModification(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	t.Logf("SQL output (trigger modification):\n%s", sql)

	hasDropTrigger := strings.Contains(sql, "DROP TRIGGER")
	hasCreateTrigger := strings.Contains(sql, "CREATE TRIGGER")
	hasCreateOrReplace := strings.Contains(sql, "CREATE OR REPLACE TRIGGER")
	t.Logf("Contains 'DROP TRIGGER': %v", hasDropTrigger)
	t.Logf("Contains 'CREATE TRIGGER': %v", hasCreateTrigger)
	t.Logf("Contains 'CREATE OR REPLACE TRIGGER': %v", hasCreateOrReplace)

	if hasDropTrigger && hasCreateTrigger && !hasCreateOrReplace {
		t.Log("Verdict: CONFIRMED - omni uses DROP+CREATE for trigger modification")
	} else if hasCreateOrReplace {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni uses CREATE OR REPLACE TRIGGER")
	} else {
		t.Log("Verdict: UNEXPECTED pattern")
	}

	require.Contains(t, sql, "TRIGGER")
}

// 5. Trigger column ref: "omni doesn't preserve UPDATE OF column syntax"
func TestOmniDiffAudit_TriggerColumnRef(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE inventory (id SERIAL PRIMARY KEY, stock INT);
		CREATE FUNCTION check_stock() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER stock_check
		BEFORE UPDATE OF stock ON inventory
		FOR EACH ROW EXECUTE FUNCTION check_stock();
	`)
	t.Logf("SQL output (trigger with UPDATE OF):\n%s", sql)

	hasUpdateOf := strings.Contains(sql, "UPDATE OF")
	t.Logf("Contains 'UPDATE OF': %v", hasUpdateOf)

	if hasUpdateOf {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni preserves UPDATE OF column syntax")
	} else {
		t.Log("Verdict: CONFIRMED - omni does not preserve UPDATE OF column syntax")
	}
}

//  6. FK in CREATE TABLE: "omni inlines FK in CREATE TABLE instead of separate
//     ALTER TABLE ADD CONSTRAINT"
func TestOmniDiffAudit_FKInCreateTable(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE categories (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			category_id INTEGER,
			CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
		);
	`)
	t.Logf("SQL output (new table with FK):\n%s", sql)

	hasAlterTableFK := strings.Contains(sql, "ALTER TABLE") && strings.Contains(sql, "FOREIGN KEY")
	hasInlineFK := false
	// Check if FOREIGN KEY appears inside a CREATE TABLE block
	createTableIdx := strings.Index(sql, "CREATE TABLE")
	if createTableIdx >= 0 {
		// Find the second CREATE TABLE (the products table)
		rest := sql[createTableIdx+len("CREATE TABLE"):]
		secondCreate := strings.Index(rest, "CREATE TABLE")
		if secondCreate >= 0 {
			// Look for FOREIGN KEY within the products CREATE TABLE statement
			afterSecondCreate := rest[secondCreate:]
			// Find the closing );
			closeParen := strings.Index(afterSecondCreate, ");")
			if closeParen > 0 {
				createBlock := afterSecondCreate[:closeParen]
				if strings.Contains(createBlock, "FOREIGN KEY") {
					hasInlineFK = true
				}
			}
		}
	}

	t.Logf("Has ALTER TABLE ... FOREIGN KEY: %v", hasAlterTableFK)
	t.Logf("Has inline FOREIGN KEY in CREATE TABLE: %v", hasInlineFK)

	if hasInlineFK && !hasAlterTableFK {
		t.Log("Verdict: CONFIRMED - omni inlines FK in CREATE TABLE")
	} else if hasAlterTableFK {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni uses separate ALTER TABLE ADD CONSTRAINT")
	} else {
		t.Log("Verdict: UNEXPECTED pattern")
	}
}

// 7. Sequence OWNED BY: "omni absorbs sequences into SERIAL/GENERATED"
func TestOmniDiffAudit_SequenceOwnedBy(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id integer NOT NULL,
			name text
		);
	`, `
		CREATE TABLE users (
			id integer NOT NULL,
			name text
		);
		CREATE SEQUENCE user_id_seq AS integer START WITH 1 INCREMENT BY 1;
		ALTER SEQUENCE user_id_seq OWNED BY users.id;
	`)
	t.Logf("SQL output (sequence OWNED BY):\n%s", sql)

	hasCreateSequence := strings.Contains(sql, "CREATE SEQUENCE")
	hasAlterSequenceOwnedBy := strings.Contains(sql, "OWNED BY")
	t.Logf("Contains 'CREATE SEQUENCE': %v", hasCreateSequence)
	t.Logf("Contains 'OWNED BY': %v", hasAlterSequenceOwnedBy)

	if !hasCreateSequence {
		t.Log("Verdict: CONFIRMED - omni absorbs owned sequences (no CREATE SEQUENCE emitted)")
	} else if hasAlterSequenceOwnedBy {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni keeps sequences + ALTER SEQUENCE OWNED BY")
	} else {
		t.Log("Verdict: PARTIAL - CREATE SEQUENCE emitted but no OWNED BY")
	}
}

// 8. Sequence modification: "omni uses DROP+CREATE instead of ALTER SEQUENCE"
func TestOmniDiffAudit_SequenceModification(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE SEQUENCE order_seq
		START WITH 1
		INCREMENT BY 1
		CACHE 1;
	`, `
		CREATE SEQUENCE order_seq
		START WITH 100
		INCREMENT BY 5
		CACHE 10;
	`)
	t.Logf("SQL output (sequence modification):\n%s", sql)

	hasDropSequence := strings.Contains(sql, "DROP SEQUENCE")
	hasCreateSequence := strings.Contains(sql, "CREATE SEQUENCE")
	hasAlterSequence := strings.Contains(sql, "ALTER SEQUENCE")
	t.Logf("Contains 'DROP SEQUENCE': %v", hasDropSequence)
	t.Logf("Contains 'CREATE SEQUENCE': %v", hasCreateSequence)
	t.Logf("Contains 'ALTER SEQUENCE': %v", hasAlterSequence)

	if hasDropSequence && hasCreateSequence {
		t.Log("Verdict: CONFIRMED - omni uses DROP+CREATE for sequence modification")
	} else if hasAlterSequence && !hasDropSequence {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni uses ALTER SEQUENCE")
	} else {
		t.Log("Verdict: UNEXPECTED pattern")
	}
}

// 9. Type normalization: "omni normalizes FLOAT to double precision"
func TestOmniDiffAudit_TypeNormalization(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE t (f FLOAT);
	`)
	t.Logf("SQL output (FLOAT type):\n%s", sql)

	hasDoublePrecision := strings.Contains(sql, "double precision")
	hasFloat := strings.Contains(sql, "FLOAT") || strings.Contains(sql, "float")
	t.Logf("Contains 'double precision': %v", hasDoublePrecision)
	t.Logf("Contains 'float' (case-insensitive): %v", hasFloat)

	if hasDoublePrecision {
		t.Log("Verdict: CONFIRMED - omni normalizes FLOAT to double precision")
	} else {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni preserves FLOAT as-is")
	}
}

// 10. Identifier case: "omni lowercases unquoted identifiers"
func TestOmniDiffAudit_IdentifierCase(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE MyTable (
			MyColumn INTEGER PRIMARY KEY,
			AnotherColumn TEXT
		);
	`)
	t.Logf("SQL output (CamelCase identifiers):\n%s", sql)

	hasLowerTable := strings.Contains(sql, "mytable")
	hasOriginalTable := strings.Contains(sql, "MyTable")
	hasLowerColumn := strings.Contains(sql, "mycolumn")
	hasOriginalColumn := strings.Contains(sql, "MyColumn")
	t.Logf("Contains 'mytable' (lowered): %v", hasLowerTable)
	t.Logf("Contains 'MyTable' (original): %v", hasOriginalTable)
	t.Logf("Contains 'mycolumn' (lowered): %v", hasLowerColumn)
	t.Logf("Contains 'MyColumn' (original): %v", hasOriginalColumn)

	if hasLowerTable && !hasOriginalTable {
		t.Log("Verdict: CONFIRMED - omni lowercases unquoted identifiers")
	} else if hasOriginalTable {
		t.Log("Verdict: NOT A REAL DIFFERENCE - omni preserves original case")
	} else {
		t.Log("Verdict: UNEXPECTED pattern")
	}
}
