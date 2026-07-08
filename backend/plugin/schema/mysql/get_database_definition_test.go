package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
