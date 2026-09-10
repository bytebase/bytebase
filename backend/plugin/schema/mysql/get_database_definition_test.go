package mysql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	// Import MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetTableDefinitionWithCheckConstraint(t *testing.T) {
	table := &storepb.TableMetadata{
		Name: "t1",
		Columns: []*storepb.ColumnMetadata{
			{Name: "type", Type: "varchar(10)", Nullable: true, Default: "NULL"},
			{Name: "amount", Type: "decimal(10,2)", Nullable: true, Default: "NULL"},
		},
		CheckConstraints: []*storepb.CheckConstraintMetadata{
			{
				Name: "c1",
				// The expression as synced from MySQL after unescaping the
				// information_schema escaped form ((`type` = _utf8mb4\'A\')...).
				Expression: "(((`type` = _utf8mb4'A') and (`amount` is not null)))",
			},
		},
		Engine:    "InnoDB",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_general_ci",
	}

	got, err := GetTableDefinition("", table, nil)
	require.NoError(t, err)

	want := "--\n" +
		"-- Table structure for `t1`\n" +
		"--\n" +
		"CREATE TABLE `t1` (\n" +
		"  `type` varchar(10) DEFAULT NULL,\n" +
		"  `amount` decimal(10,2) DEFAULT NULL,\n" +
		"  CONSTRAINT `c1` CHECK (((`type` = _utf8mb4'A') and (`amount` is not null)))\n" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n"
	require.Equal(t, want, got)
	require.NotContains(t, got, `\'`)
}

func TestGetFunctionDefinitionWithMultilineParameters(t *testing.T) {
	function := &storepb.FunctionMetadata{
		Name: "f1",
		// The definition as synced from SHOW CREATE FUNCTION: the parameter
		// list keeps its original multi-line formatting, and the CHARSET and
		// COLLATE attributes of the RETURNS clause have been stripped.
		Definition: "CREATE FUNCTION `f1`(\n" +
			"    p_a BIGINT,\n" +
			"    p_b VARCHAR(36)\n" +
			") RETURNS char(36)\n" +
			"    NO SQL\n" +
			"    DETERMINISTIC\n" +
			"BEGIN\n" +
			"    RETURN p_b;\n" +
			"END",
		CharacterSetClient:  "utf8mb4",
		CollationConnection: "utf8mb4_unicode_ci",
		SqlMode:             "ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES",
	}

	got, err := GetFunctionDefinition("", function)
	require.NoError(t, err)

	want := "--\n" +
		"-- Function structure for `f1`\n" +
		"--\n" +
		"SET character_set_client = utf8mb4;\n" +
		"SET character_set_results = utf8mb4;\n" +
		"SET collation_connection = utf8mb4_unicode_ci;\n" +
		"SET sql_mode = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES';\n" +
		"CREATE FUNCTION `f1`(\n" +
		"    p_a BIGINT,\n" +
		"    p_b VARCHAR(36)\n" +
		") RETURNS char(36)\n" +
		"    NO SQL\n" +
		"    DETERMINISTIC\n" +
		"BEGIN\n" +
		"    RETURN p_b;\n" +
		"END;;\n" +
		"DELIMITER ;\n\n"
	require.Equal(t, want, got)
}

func TestRenderColumnDefaultBitLiteral(t *testing.T) {
	testCases := []struct {
		name       string
		columnType string
		// def is the value stored in ColumnMetadata.Default. For a BIT column the MySQL
		// sync stores the bit literal QUOTE()-escaped, e.g. b'0' becomes 'b\'0\''.
		def  string
		want string
	}{
		{name: "bit1 quoted-escaped", columnType: "bit(1)", def: `'b\'0\''`, want: "b'0'"},
		{name: "bit8 quoted-escaped", columnType: "bit(8)", def: `'b\'101\''`, want: "b'101'"},
		{name: "bit no width", columnType: "bit", def: `'b\'1\''`, want: "b'1'"},
		{name: "bit already bare literal", columnType: "bit(4)", def: "b'1010'", want: "b'1010'"},
		{name: "bit hex 0x form", columnType: "bit(8)", def: "0xFF", want: "0xFF"},
		{name: "bit hex x'' quoted-escaped", columnType: "bit(8)", def: `'x\'1f\''`, want: "x'1f'"},
		// Non-bit columns and non-literal defaults are emitted verbatim.
		{name: "varchar string default untouched", columnType: "varchar(10)", def: "'hello'", want: "'hello'"},
		{name: "varchar string that looks bit-ish but not bit column", columnType: "varchar(10)", def: `'b\'0\''`, want: `'b\'0\''`},
		{name: "int numeric default untouched", columnType: "int", def: "42", want: "42"},
		// A BIT column whose default is not a bit/hex literal (defensive) is left verbatim.
		{name: "bit with non-literal default", columnType: "bit(8)", def: "'abc'", want: "'abc'"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderColumnDefault(&storepb.ColumnMetadata{Type: tc.columnType, Default: tc.def})
			require.Equal(t, tc.want, got)
		})
	}
}

func TestPrintColumnClauseBitDefaultUnquoted(t *testing.T) {
	table := &storepb.TableMetadata{Name: "t", Charset: "utf8mb4"}
	var buf strings.Builder
	col := &storepb.ColumnMetadata{Name: "b1", Type: "bit(1)", Nullable: false, Default: `'b\'0\''`}
	require.NoError(t, printColumnClause(&buf, col, table))
	require.Equal(t, "  `b1` bit(1) NOT NULL DEFAULT b'0'", buf.String())

	// A BIT column with no default (b64 BIT(64)) must stay fine: NULL default emitted as
	// DEFAULT NULL, never as a bit literal.
	var buf2 strings.Builder
	col2 := &storepb.ColumnMetadata{Name: "b64", Type: "bit(64)", Nullable: true, Default: "NULL"}
	require.NoError(t, printColumnClause(&buf2, col2, table))
	require.Equal(t, "  `b64` bit(64) DEFAULT NULL", buf2.String())
}

func TestNormalizeFunctionalIndexExpr(t *testing.T) {
	testCases := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "multi-valued json index with utf8mb4 introducer and escaped quotes",
			// The form synced from information_schema.STATISTICS.EXPRESSION.
			expr: "(cast(json_extract(`tags`,_utf8mb4\\'$.ids\\') as unsigned array))",
			want: "(cast(json_extract(`tags`,'$.ids') as unsigned array))",
		},
		{
			name: "latin1 introducer",
			expr: "(json_extract(`c`,_latin1\\'$.a\\'))",
			want: "(json_extract(`c`,'$.a'))",
		},
		{
			name: "no introducer no escaping is unchanged",
			expr: "(json_extract(`tags`,'$.ids'))",
			want: "(json_extract(`tags`,'$.ids'))",
		},
		{
			name: "introducer-like underscore inside string literal is preserved",
			// _utf8mb4 appears INSIDE the literal here; only a real introducer (before the
			// opening quote) must be stripped.
			expr: "(concat(`c`,_utf8mb4\\'_utf8mb4 is text\\'))",
			want: "(concat(`c`,'_utf8mb4 is text'))",
		},
		{
			name: "leading-underscore identifier not before a quote is kept",
			expr: "(`_weird`)",
			want: "(`_weird`)",
		},
		{
			name: "backticked identifier with underscore is untouched",
			expr: "(json_extract(`my_col`,_utf8mb4\\'$.k\\'))",
			want: "(json_extract(`my_col`,'$.k'))",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeFunctionalIndexExpr(tc.expr))
		})
	}
}

func TestPrintIndexClauseFunctionalExprNormalized(t *testing.T) {
	var buf strings.Builder
	index := &storepb.IndexMetadata{
		Name:        "idx_tags",
		Visible:     true,
		Expressions: []string{"(cast(json_extract(`tags`,_utf8mb4\\'$.ids\\') as unsigned array))"},
	}
	require.NoError(t, printIndexClause(&buf, index))
	require.Equal(t, ",\n  KEY `idx_tags` ((cast(json_extract(`tags`,'$.ids') as unsigned array)))", buf.String())
	require.NotContains(t, buf.String(), "_utf8mb4")
	require.NotContains(t, buf.String(), `\'`)
}

// TestStripViewBodyDatabaseQualifier pins the token-scanner rewrite (X6). The failure
// shapes were live-verified: (a) `db`.`db`.`col` (a table named like the database, the
// 5.7 three-part form) was double-stripped down to `col`; (b) a backtick alias
// containing a quote (`it's`) flipped the old scanner's literal state so NO subsequent
// qualifier was stripped; (c) a genuine string literal containing the qualifier bytes
// was corrupted.
func TestStripViewBodyDatabaseQualifier(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		dbName string
		want   string
	}{
		{
			name:   "table_named_like_database_keeps_table_qualifier",
			body:   "select `db`.`db`.`col` AS `col` from `db`.`db`",
			dbName: "db",
			want:   "select `db`.`col` AS `col` from `db`",
		},
		{
			name: "alias_with_quote_does_not_flip_literal_state",
			// 5.7 derived-table shape: all three qualifiers must strip even though the
			// alias `it's` contains a single quote.
			body:   "select `x`.`a` AS `it's` from (select `mydb`.`t`.`a` AS `a` from `mydb`.`t` where (`mydb`.`t`.`a` > 0)) `x`",
			dbName: "mydb",
			want:   "select `x`.`a` AS `it's` from (select `t`.`a` AS `a` from `t` where (`t`.`a` > 0)) `x`",
		},
		{
			name:   "control_alias_without_quote",
			body:   "select `x`.`a` AS `plain` from (select `mydb`.`t`.`a` AS `a` from `mydb`.`t`) `x`",
			dbName: "mydb",
			want:   "select `x`.`a` AS `plain` from (select `t`.`a` AS `a` from `t`) `x`",
		},
		{
			name:   "cross_database_reference_preserved",
			body:   "select `otherdb`.`t`.`c` from `otherdb`.`t` join `mydb`.`u` on (`otherdb`.`t`.`id` = `mydb`.`u`.`id`)",
			dbName: "mydb",
			want:   "select `otherdb`.`t`.`c` from `otherdb`.`t` join `u` on (`otherdb`.`t`.`id` = `u`.`id`)",
		},
		{
			name: "db_named_segment_inside_foreign_reference_preserved",
			// `mydb` here is the TABLE part of a cross-database reference — a
			// continuation after ".", never a qualifier.
			body:   "select `otherdb`.`mydb`.`c` from `otherdb`.`mydb`",
			dbName: "mydb",
			want:   "select `otherdb`.`mydb`.`c` from `otherdb`.`mydb`",
		},
		{
			name: "quote_flip_then_literal_corruption",
			// The live-verified composed corruption: with the old scanner, the quote
			// inside alias `it's` flipped the literal state, so the REAL literal's
			// interior was treated as top-level (its `mydb`. stripped — data change)
			// while the genuine trailing qualifier survived.
			body:   "select `it's` AS a, '`mydb`.`t` and  spaces' AS lit from `mydb`.`t`",
			dbName: "mydb",
			want:   "select `it's` AS a, '`mydb`.`t` and  spaces' AS lit from `t`",
		},
		{
			name:   "string_literal_containing_qualifier_untouched",
			body:   "select '`mydb`.`t`' AS lit, `mydb`.`t`.`c` from `mydb`.`t`",
			dbName: "mydb",
			want:   "select '`mydb`.`t`' AS lit, `t`.`c` from `t`",
		},
		{
			name:   "double_quoted_literal_untouched",
			body:   `select "` + "`mydb`.`t`" + `" AS lit from ` + "`mydb`.`t`",
			dbName: "mydb",
			want:   `select "` + "`mydb`.`t`" + `" AS lit from ` + "`t`",
		},
		{
			name:   "qualifier_not_followed_by_identifier_preserved",
			body:   "select `mydb`.* from `mydb`.`t`",
			dbName: "mydb",
			want:   "select `mydb`.* from `t`",
		},
		{
			name:   "empty_db_name_no_op",
			body:   "select `t`.`c` from `t`",
			dbName: "",
			want:   "select `t`.`c` from `t`",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, stripViewBodyDatabaseQualifier(tc.body, tc.dbName))
		})
	}
}

// TestStripLeadingDefiner pins the definer-clause parser (X7): quoted accounts may
// legally contain spaces (`my user`@`%`), '@', and doubled/escaped quotes — the old
// cut-at-first-space produced corrupt "CREATE user`@`%` …" output.
func TestStripLeadingDefiner(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain_account",
			in:   "CREATE DEFINER=`root`@`localhost` EVENT `e` ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT `e` ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "quoted_user_with_space",
			in:   "CREATE DEFINER=`my user`@`%` EVENT `e` ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT `e` ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "quoted_host_with_space_function",
			in:   "CREATE DEFINER=`a``b`@`local host` FUNCTION `f`() RETURNS int DETERMINISTIC RETURN 1",
			want: "CREATE FUNCTION `f`() RETURNS int DETERMINISTIC RETURN 1",
		},
		{
			name: "single_quoted_account_procedure",
			in:   "CREATE DEFINER='my user'@'%' PROCEDURE `p`() BEGIN END",
			want: "CREATE PROCEDURE `p`() BEGIN END",
		},
		{
			name: "double_quoted_account",
			in:   `CREATE DEFINER="my user"@"%" EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t`,
			want: "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "unquoted_account",
			in:   "CREATE DEFINER=root@localhost EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "current_user",
			in:   "CREATE DEFINER=CURRENT_USER EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "current_user_parens",
			in:   "CREATE DEFINER=CURRENT_USER() EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "spaces_around_equals",
			in:   "CREATE DEFINER = `my user`@`%` EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "no_definer_unchanged",
			in:   "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
			want: "CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO DELETE FROM t",
		},
		{
			name: "unterminated_quote_left_unchanged",
			in:   "CREATE DEFINER=`broken EVENT e DO SELECT 1",
			want: "CREATE DEFINER=`broken EVENT e DO SELECT 1",
		},
		{
			name: "not_a_create_statement_unchanged",
			in:   "ALTER DEFINER=`root`@`%` EVENT e COMMENT 'x'",
			want: "ALTER DEFINER=`root`@`%` EVENT e COMMENT 'x'",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, stripLeadingDefiner(tc.in))
		})
	}
}

// TestPrintColumnClauseSRIDPresence pins the presence semantics of the SRID attribute
// (X5): explicit SRID 0 must be emitted (it is a valid spatial reference system,
// distinct from "no SRID"), and an unset SRID emits nothing.
func TestPrintColumnClauseSRIDPresence(t *testing.T) {
	srid := func(v uint32) *uint32 { return &v }
	table := &storepb.TableMetadata{Name: "t"}

	render := func(col *storepb.ColumnMetadata) string {
		var buf strings.Builder
		require.NoError(t, printColumnClause(&buf, col, table))
		return buf.String()
	}

	require.Equal(t, "  `pt` point NOT NULL /*!80003 SRID 0 */",
		render(&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(0)}))
	require.Equal(t, "  `pt` point NOT NULL /*!80003 SRID 4326 */",
		render(&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)}))
	// Custom SRSs above int32 range must render unmangled.
	require.Equal(t, "  `pt` point NOT NULL /*!80003 SRID 3000000000 */",
		render(&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(3000000000)}))
	require.Equal(t, "  `pt` point NOT NULL",
		render(&storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false}))
}

// TestPrintColumnClauseInvisibleCommentOrder pins the SDL dumper's INVISIBLE/COMMENT order
// (BYT-9830). MySQL's canonical SHOW CREATE emits INVISIBLE before COMMENT for both regular
// and generated columns (verified against 8.0.32), so the dumped form must match — otherwise
// the dump diverges from SHOW CREATE and from the migration generator's generated-column path.
func TestPrintColumnClauseInvisibleCommentOrder(t *testing.T) {
	srid := func(v uint32) *uint32 { return &v }
	table := &storepb.TableMetadata{Name: "t"}

	render := func(col *storepb.ColumnMetadata) string {
		var buf strings.Builder
		require.NoError(t, printColumnClause(&buf, col, table))
		return buf.String()
	}

	// Regular INVISIBLE + COMMENT column: INVISIBLE precedes COMMENT.
	require.Equal(t, "  `a` int DEFAULT NULL /*!80023 INVISIBLE */ COMMENT 'c'",
		render(&storepb.ColumnMetadata{Name: "a", Type: "int", Nullable: true, Default: "NULL", IsInvisible: true, Comment: "c"}))

	// Generated spatial INVISIBLE + COMMENT column: same canonical INVISIBLE-before-COMMENT order.
	require.Equal(t,
		"  `loc` point GENERATED ALWAYS AS (st_srid(point(`lng`,`lat`),4326)) STORED NOT NULL /*!80003 SRID 4326 */ /*!80023 INVISIBLE */ COMMENT 'geo'",
		render(&storepb.ColumnMetadata{Name: "loc", Type: "point", Nullable: false, Srid: srid(4326), IsInvisible: true, Comment: "geo", Generation: &storepb.GenerationMetadata{
			Type:       storepb.GenerationMetadata_TYPE_STORED,
			Expression: "st_srid(point(`lng`,`lat`),4326)",
		}}))
}

// TestWriteColumnDefinitionBodyGeneratedOrder pins the migration generator's attribute
// order for a generated column (BYT-9830). MySQL's grammar requires the
// `GENERATED ALWAYS AS (...) STORED|VIRTUAL` clause to precede NOT NULL and the SRID
// attribute; emitting SRID/NOT NULL first (the prior order) is rejected with ERROR 1064
// for a generated spatial column. The canonical order — verified against MySQL 8.0.32
// via SHOW CREATE — is:
//
//	type GENERATED ALWAYS AS (expr) STORED|VIRTUAL [NOT NULL] [SRID] [INVISIBLE] [COMMENT]
func TestWriteColumnDefinitionBodyGeneratedOrder(t *testing.T) {
	srid := func(v uint32) *uint32 { return &v }
	stored := &storepb.GenerationMetadata{
		Type:       storepb.GenerationMetadata_TYPE_STORED,
		Expression: "st_srid(point(`lng`,`lat`),4326)",
	}

	render := func(col *storepb.ColumnMetadata) string {
		var buf strings.Builder
		writeColumnDefinitionBody(&buf, col)
		return buf.String()
	}

	// Generated spatial column, NOT NULL + SRID — the failing case. The generation clause
	// must come first, then NOT NULL, then SRID.
	require.Equal(t,
		"point GENERATED ALWAYS AS (st_srid(point(`lng`,`lat`),4326)) STORED NOT NULL /*!80003 SRID 4326 */",
		render(&storepb.ColumnMetadata{Name: "loc", Type: "point", Nullable: false, Srid: srid(4326), Generation: stored}))

	// Nullable generated spatial column: no NOT NULL, SRID still after the generation clause.
	require.Equal(t,
		"point GENERATED ALWAYS AS (st_srid(point(`lng`,`lat`),4326)) STORED /*!80003 SRID 0 */",
		render(&storepb.ColumnMetadata{Name: "loc", Type: "point", Nullable: true, Srid: srid(0), Generation: stored}))

	// Generated spatial column that is also INVISIBLE and has a COMMENT: INVISIBLE precedes
	// COMMENT, matching SHOW CREATE's canonical order for both regular and generated columns.
	require.Equal(t,
		"point GENERATED ALWAYS AS (st_srid(point(`lng`,`lat`),4326)) STORED NOT NULL /*!80003 SRID 4326 */ /*!80023 INVISIBLE */ COMMENT 'geo'",
		render(&storepb.ColumnMetadata{Name: "loc", Type: "point", Nullable: false, Srid: srid(4326), IsInvisible: true, Comment: "geo", Generation: stored}))

	// Regular (non-generated) INVISIBLE column with a COMMENT: INVISIBLE precedes COMMENT,
	// the same canonical order as the generated column above (verified against 8.0.32).
	require.Equal(t,
		"int DEFAULT NULL /*!80023 INVISIBLE */ COMMENT 'c'",
		render(&storepb.ColumnMetadata{Name: "a", Type: "int", Nullable: true, Default: "NULL", IsInvisible: true, Comment: "c"}))

	// Plain (non-spatial) VIRTUAL generated column still emits the generation clause and
	// nothing spurious.
	require.Equal(t,
		"int GENERATED ALWAYS AS (`a` + 1) VIRTUAL",
		render(&storepb.ColumnMetadata{Name: "b", Type: "int", Nullable: true, Generation: &storepb.GenerationMetadata{
			Type:       storepb.GenerationMetadata_TYPE_VIRTUAL,
			Expression: "`a` + 1",
		}}))

	// Regression: a non-generated NOT NULL spatial column keeps the regular order
	// (NOT NULL then SRID, no generation clause).
	require.Equal(t,
		"point NOT NULL /*!80003 SRID 4326 */",
		render(&storepb.ColumnMetadata{Name: "loc", Type: "point", Nullable: false, Srid: srid(4326)}))
}

//nolint:tparallel
func TestGetDatabaseDefinition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}

	ctx := context.Background()

	// Start shared MySQL container for all subtests
	container := testcontainer.SharedMySQLContainer(t)

	type testCase struct {
		description string
		originalDDL string
	}

	testCases := []testCase{
		{
			description: "Basic tables with various column types",
			originalDDL: `
CREATE TABLE users (
	id INT PRIMARY KEY AUTO_INCREMENT,
	username VARCHAR(50) NOT NULL UNIQUE,
	email VARCHAR(100) NOT NULL,
	age INT CHECK (age >= 18),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	profile JSON,
	is_active BOOLEAN DEFAULT TRUE,
	INDEX idx_email (email),
	INDEX idx_created_at (created_at)
);

CREATE TABLE posts (
	id INT PRIMARY KEY AUTO_INCREMENT,
	user_id INT NOT NULL,
	title VARCHAR(200) NOT NULL,
	content TEXT,
	published_at DATETIME,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	INDEX idx_user_published (user_id, published_at)
);
`,
		},
		{
			description: "Generated columns and complex indexes",
			originalDDL: `
CREATE TABLE products (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	price DECIMAL(10, 2) NOT NULL,
	tax_rate DECIMAL(5, 2) NOT NULL DEFAULT 0.08,
	price_with_tax DECIMAL(10, 2) AS (price * (1 + tax_rate)) STORED,
	description TEXT,
	tags JSON,
	FULLTEXT idx_fulltext (name, description)
);

CREATE TABLE inventory (
	id INT PRIMARY KEY AUTO_INCREMENT,
	product_id INT NOT NULL,
	warehouse VARCHAR(50) NOT NULL,
	quantity INT NOT NULL DEFAULT 0,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	UNIQUE KEY uk_product_warehouse (product_id, warehouse),
	FOREIGN KEY (product_id) REFERENCES products(id)
);
`,
		},
		{
			description: "Views and triggers",
			originalDDL: `
CREATE TABLE orders (
	id INT PRIMARY KEY AUTO_INCREMENT,
	order_number VARCHAR(20) NOT NULL UNIQUE,
	customer_name VARCHAR(100) NOT NULL,
	total_amount DECIMAL(10, 2) NOT NULL,
	status VARCHAR(20) DEFAULT 'pending',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_history (
	id INT PRIMARY KEY AUTO_INCREMENT,
	order_id INT NOT NULL,
	old_status VARCHAR(20),
	new_status VARCHAR(20),
	changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (order_id) REFERENCES orders(id)
);

CREATE VIEW pending_orders AS
SELECT id, order_number, customer_name, total_amount, created_at
FROM orders
WHERE status = 'pending'
ORDER BY created_at DESC;

CREATE TRIGGER order_status_change
AFTER UPDATE ON orders
FOR EACH ROW
BEGIN
	IF OLD.status != NEW.status THEN
		INSERT INTO order_history (order_id, old_status, new_status)
		VALUES (NEW.id, OLD.status, NEW.status);
	END IF;
END;
`,
		},
		{
			description: "Stored procedures and functions",
			originalDDL: `
CREATE TABLE accounts (
	id INT PRIMARY KEY AUTO_INCREMENT,
	account_number VARCHAR(20) NOT NULL UNIQUE,
	balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

DELIMITER $$

CREATE FUNCTION calculate_interest(principal DECIMAL(15, 2), rate DECIMAL(5, 4), years INT)
RETURNS DECIMAL(15, 2)
DETERMINISTIC
READS SQL DATA
BEGIN
	RETURN principal * POW(1 + rate, years);
END$$

CREATE PROCEDURE transfer_funds(
	IN from_account VARCHAR(20),
	IN to_account VARCHAR(20),
	IN amount DECIMAL(15, 2)
)
BEGIN
	DECLARE from_balance DECIMAL(15, 2);
	
	START TRANSACTION;
	
	SELECT balance INTO from_balance
	FROM accounts
	WHERE account_number = from_account
	FOR UPDATE;
	
	IF from_balance >= amount THEN
		UPDATE accounts
		SET balance = balance - amount
		WHERE account_number = from_account;
		
		UPDATE accounts
		SET balance = balance + amount
		WHERE account_number = to_account;
		
		COMMIT;
	ELSE
		ROLLBACK;
		SIGNAL SQLSTATE '45000'
		SET MESSAGE_TEXT = 'Insufficient funds';
	END IF;
END$$

DELIMITER ;
`,
		},
		{
			description: "Partitioned tables",
			originalDDL: `
-- RANGE partition
CREATE TABLE sales (
	id INT NOT NULL AUTO_INCREMENT,
	sale_date DATE NOT NULL,
	product_id INT NOT NULL,
	quantity INT NOT NULL,
	amount DECIMAL(10, 2) NOT NULL,
	PRIMARY KEY (id, sale_date)
) PARTITION BY RANGE (YEAR(sale_date)) (
	PARTITION p2022 VALUES LESS THAN (2023),
	PARTITION p2023 VALUES LESS THAN (2024),
	PARTITION p2024 VALUES LESS THAN (2025),
	PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- HASH partition
CREATE TABLE employees (
	id INT NOT NULL,
	name VARCHAR(100) NOT NULL,
	department_id INT NOT NULL,
	hired_date DATE,
	PRIMARY KEY (id)
) PARTITION BY HASH(id) PARTITIONS 4;

-- LIST partition
CREATE TABLE customer_regions (
	id INT NOT NULL AUTO_INCREMENT,
	customer_name VARCHAR(100) NOT NULL,
	region VARCHAR(20) NOT NULL,
	sales_amount DECIMAL(10, 2),
	PRIMARY KEY (id, region)
) PARTITION BY LIST COLUMNS(region) (
	PARTITION p_north VALUES IN ('north', 'northeast', 'northwest'),
	PARTITION p_south VALUES IN ('south', 'southeast', 'southwest'),
	PARTITION p_east VALUES IN ('east'),
	PARTITION p_west VALUES IN ('west'),
	PARTITION p_central VALUES IN ('central')
);

-- KEY partition
CREATE TABLE user_sessions (
	session_id VARCHAR(64) NOT NULL,
	user_id INT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (session_id)
) PARTITION BY KEY() PARTITIONS 8;

-- RANGE COLUMNS partition
CREATE TABLE order_archive (
	order_id INT NOT NULL,
	order_date DATE NOT NULL,
	customer_id INT NOT NULL,
	status VARCHAR(20) NOT NULL,
	total_amount DECIMAL(10, 2),
	PRIMARY KEY (order_id, order_date)
) PARTITION BY RANGE COLUMNS(order_date) (
	PARTITION p_2022_q1 VALUES LESS THAN ('2022-04-01'),
	PARTITION p_2022_q2 VALUES LESS THAN ('2022-07-01'),
	PARTITION p_2022_q3 VALUES LESS THAN ('2022-10-01'),
	PARTITION p_2022_q4 VALUES LESS THAN ('2023-01-01'),
	PARTITION p_2023_and_later VALUES LESS THAN (MAXVALUE)
);
`,
		},
		{
			description: "Events",
			originalDDL: `
CREATE TABLE daily_stats (
	id INT PRIMARY KEY AUTO_INCREMENT,
	stat_date DATE NOT NULL UNIQUE,
	total_orders INT DEFAULT 0,
	total_revenue DECIMAL(15, 2) DEFAULT 0.00,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE EVENT IF NOT EXISTS update_daily_stats
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
	INSERT INTO daily_stats (stat_date, total_orders, total_revenue)
	VALUES (CURDATE() - INTERVAL 1 DAY, 0, 0.00)
	ON DUPLICATE KEY UPDATE
		total_orders = VALUES(total_orders),
		total_revenue = VALUES(total_revenue);
`,
		},
		{
			description: "Character sets and collations",
			originalDDL: `
CREATE TABLE translations (
	id INT PRIMARY KEY AUTO_INCREMENT,
	language_code VARCHAR(5) CHARACTER SET ascii NOT NULL,
	content_key VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
	translation TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	UNIQUE KEY uk_lang_key (language_code, content_key)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`,
		},
		{
			description: "Check constraints with string literals and function with multiline parameters",
			originalDDL: `
CREATE TABLE transactions (
	id INT PRIMARY KEY AUTO_INCREMENT,
	type VARCHAR(10),
	amount DECIMAL(10, 2),
	CONSTRAINT c1 CHECK (type = 'A' AND amount IS NOT NULL)
);

DELIMITER $$

CREATE FUNCTION lookup_token(
	p_a BIGINT,
	p_b VARCHAR(36)
) RETURNS CHAR(36) CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci
	NO SQL
	DETERMINISTIC
BEGIN
	RETURN p_b;
END$$

DELIMITER ;
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			// Create unique test databases for parallel execution
			testDBName := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
			newDBName := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

			// Create initial test database
			_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDBName))
			require.NoError(t, err)

			// Step 1: Initialize the database schema and get metadata A
			driverA, err := createMySQLDriver(ctx, container.GetHost(), container.GetPort(), testDBName)
			require.NoError(t, err)
			defer driverA.Close(ctx)

			_, err = driverA.Execute(ctx, tc.originalDDL, db.ExecuteOptions{})
			require.NoError(t, err)

			metadataA, err := driverA.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Step 2: Call GetDatabaseDefinition to generate the database definition X
			defCtx := schema.GetDefinitionContext{
				SkipBackupSchema: false,
				PrintHeader:      true,
			}
			definitionX, err := schema.GetDatabaseDefinition(storepb.Engine_MYSQL, defCtx, metadataA)
			require.NoError(t, err)
			require.NotEmpty(t, definitionX)

			// Step 3: Create a new database and apply the generated DDL
			_, err = container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", newDBName))
			require.NoError(t, err)

			driverB, err := createMySQLDriver(ctx, container.GetHost(), container.GetPort(), newDBName)
			require.NoError(t, err)
			defer driverB.Close(ctx)

			_, err = driverB.Execute(ctx, definitionX, db.ExecuteOptions{})
			require.NoError(t, err)

			metadataB, err := driverB.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Step 4: Compare the database metadata A and B, should be the same
			normalizeMetadata(metadataA)
			normalizeMetadata(metadataB)

			opts := []cmp.Option{
				protocmp.Transform(),
				protocmp.IgnoreFields(&storepb.TableMetadata{}, "row_count", "data_size", "index_size", "data_free"),
			}

			// Add custom ignored fields for specific test cases (for events test)
			if tc.description == "Events" {
				// Ignore time-specific fields that vary between runs
				opts = append(opts, protocmp.IgnoreFields(&storepb.EventMetadata{}, "time_zone", "sql_mode", "character_set_client"))
			}

			if diff := cmp.Diff(metadataA, metadataB, opts...); diff != "" {
				t.Errorf("Metadata mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// normalizeMetadata normalizes the metadata to ignore differences that don't affect schema equivalence
func normalizeMetadata(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database name as it will differ between original and recreated
	metadata.Name = ""

	// Normalize AUTO_INCREMENT values to 0 as they can differ
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			// Clear runtime-specific values
			table.RowCount = 0
			table.DataSize = 0
			table.IndexSize = 0
			table.DataFree = 0

			// Normalize column defaults
			for _, column := range table.Columns {
				// MySQL might represent defaults differently
				if def := column.GetDefault(); def != "" {
					// Normalize CURRENT_TIMESTAMP variations
					if def == "CURRENT_TIMESTAMP" ||
						def == "current_timestamp()" ||
						def == "now()" {
						column.Default = "CURRENT_TIMESTAMP"
					}
				}
			}

			// Remove duplicate check constraints (sometimes appear as both message and string)
			seen := make(map[string]bool)
			var uniqueChecks []*storepb.CheckConstraintMetadata
			for _, check := range table.CheckConstraints {
				key := fmt.Sprintf("%s:%s", check.Name, check.Expression)
				if !seen[key] {
					seen[key] = true
					uniqueChecks = append(uniqueChecks, check)
				}
			}
			table.CheckConstraints = uniqueChecks
		}
	}
}

// TestGetDatabaseDefinitionWithConnectedDeps tests the ability to handle complex foreign key dependencies
func TestGetDatabaseDefinitionWithConnectedDeps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}

	// Unique per run: TestMain keeps one container for the whole package, so a
	// fixed name collides with itself under go test -count=2.
	databaseName := fmt.Sprintf("test_complex_deps_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

	const (
		originalDDL = `
CREATE TABLE department (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	parent_id INT,
	FOREIGN KEY (parent_id) REFERENCES department(id) ON DELETE SET NULL
);

CREATE TABLE employee (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	department_id INT,
	manager_id INT,
	FOREIGN KEY (department_id) REFERENCES department(id) ON DELETE SET NULL,
	FOREIGN KEY (manager_id) REFERENCES employee(id) ON DELETE SET NULL
);

CREATE TABLE project (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	lead_id INT NOT NULL,
	department_id INT NOT NULL,
	FOREIGN KEY (lead_id) REFERENCES employee(id),
	FOREIGN KEY (department_id) REFERENCES department(id)
);

CREATE TABLE project_member (
	project_id INT NOT NULL,
	employee_id INT NOT NULL,
	role VARCHAR(50),
	PRIMARY KEY (project_id, employee_id),
	FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
	FOREIGN KEY (employee_id) REFERENCES employee(id) ON DELETE CASCADE
);
`
	)

	ctx := context.Background()

	// Start MySQL container
	container := testcontainer.SharedMySQLContainer(t)

	// Create test database
	_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", databaseName))
	require.NoError(t, err)

	// Step 1: Initialize the database schema
	driverA, err := createMySQLDriver(ctx, container.GetHost(), container.GetPort(), databaseName)
	require.NoError(t, err)
	defer driverA.Close(ctx)

	_, err = driverA.Execute(ctx, originalDDL, db.ExecuteOptions{})
	require.NoError(t, err)

	metadataA, err := driverA.SyncDBSchema(ctx)
	require.NoError(t, err)

	// Step 2: Generate definition
	defCtx := schema.GetDefinitionContext{
		SkipBackupSchema: false,
		PrintHeader:      true,
	}
	definitionX, err := schema.GetDatabaseDefinition(storepb.Engine_MYSQL, defCtx, metadataA)
	require.NoError(t, err)

	// Step 3: Create new database and apply definition
	newDBName := fmt.Sprintf("%s_recreated", databaseName)
	_, err = container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", newDBName))
	require.NoError(t, err)

	driverB, err := createMySQLDriver(ctx, container.GetHost(), container.GetPort(), newDBName)
	require.NoError(t, err)
	defer driverB.Close(ctx)

	_, err = driverB.Execute(ctx, definitionX, db.ExecuteOptions{})
	require.NoError(t, err)

	metadataB, err := driverB.SyncDBSchema(ctx)
	require.NoError(t, err)

	// Compare
	normalizeMetadata(metadataA)
	normalizeMetadata(metadataB)

	opts := []cmp.Option{
		protocmp.Transform(),
		protocmp.IgnoreFields(&storepb.TableMetadata{}, "row_count", "data_size", "index_size", "data_free"),
	}

	if diff := cmp.Diff(metadataA, metadataB, opts...); diff != "" {
		t.Errorf("Metadata mismatch (-want +got):\n%s", diff)
	}
}
