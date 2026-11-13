// Package tidb implements the SQL advisor rules for TiDB.
package tidb

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	omniast "github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

type columnSet map[string]bool

func newColumnSet(columns []string) columnSet {
	res := make(columnSet)
	for _, col := range columns {
		res[col] = true
	}
	return res
}

type tableState map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tableState) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	slices.Sort(tableList)
	return tableList
}

type tablePK map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tablePK) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	slices.Sort(tableList)
	return tableList
}

// Stmt bundles an omni/tidb AST node with the source text and base line of
// the statement it came from. The base line is needed to convert byte offsets
// inside Text into absolute line numbers in the original full SQL.
type Stmt struct {
	Node     omniast.Node
	Text     string
	BaseLine int // 0-based line index of the first line of Text in the original SQL
}

// AbsoluteLine converts a byte offset within s.Text into a 1-based line number
// in the original SQL.
func (s Stmt) AbsoluteLine(byteOffset int) int {
	pos := tidbparser.ByteOffsetToRunePosition(s.Text, byteOffset)
	return s.BaseLine + int(pos.Line)
}

// TrimmedText returns Text with surrounding whitespace removed. Suitable
// for embedding the statement text into advice content; raw Text may
// include leading/trailing newlines from the original multi-statement
// split.
func (s Stmt) TrimmedText() string {
	return strings.TrimSpace(s.Text)
}

// FirstTokenLine returns the 1-based absolute line of the first
// non-whitespace character in s.Text. Matches pingcap's
// OriginTextPosition: pingcap's lexer strips leading whitespace but
// keeps comments as part of the statement, so its reported line points
// at the first comment OR keyword. Used as the StartPosition for
// statement-level advices.
func (s Stmt) FirstTokenLine() int {
	for i, r := range s.Text {
		if !unicode.IsSpace(r) {
			return s.AbsoluteLine(i)
		}
	}
	return s.AbsoluteLine(0)
}

// AST-shape accessors for omni TableOption / DatabaseOption / ColumnDef.
//
// Cumulative shape divergence #14: omni replaces pingcap's
// `{Tp: enum, StrValue}` model with `{Name: "CANONICAL KEYWORD", Value}`
// for both Table and Database options. omni normalizes Name to uppercase
// regardless of user case, but EqualFold is used here for safety. These
// accessors centralize the lookup so future advisors don't repeat the
// shape-aware logic.

// omni preserves the user's keyword form on TableOption.Name and
// DatabaseOption.Name — CHARSET and CHARACTER SET are NOT canonicalized
// to a single form. Pingcap canonicalized at the enum level (both
// produced TableOptionCharset / DatabaseOptionCharset). Accessors below
// take a name-set and match any of them, EqualFold-style. Helps batch
// authors avoid the silent-regression failure mode where mechanically
// using only the canonical name misses the shorthand form.
var (
	optionNamesCharset = []string{"CHARACTER SET", "CHARSET"}
	optionNamesCollate = []string{"COLLATE"}
	optionNamesComment = []string{"COMMENT"}
)

// databaseOption returns the lowercased value of the first option
// whose Name matches any in `names` (case-insensitive), or "" if none
// match. For case-insensitive identifiers (charset/collation names);
// user-content values like COMMENT should use *Present variants.
func databaseOption(opts []*omniast.DatabaseOption, names []string) string {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		for _, name := range names {
			if strings.EqualFold(opt.Name, name) {
				return strings.ToLower(opt.Value)
			}
		}
	}
	return ""
}

// tableOption returns the lowercased value of the first option whose
// Name matches any in `names` (case-insensitive), or "" if none match.
// Case-insensitive-identifier semantics (see databaseOption).
func tableOption(opts []*omniast.TableOption, names []string) string {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		for _, name := range names {
			if strings.EqualFold(opt.Name, name) {
				return strings.ToLower(opt.Value)
			}
		}
	}
	return ""
}

// tableOptionPresent reports whether any option whose Name matches
// any in `names` is in the slice; returns its raw value (case preserved —
// user content). Distinguishes "no clause" from "clause with empty value"
// (e.g. COMMENT ”).
func tableOptionPresent(opts []*omniast.TableOption, names []string) (string, bool) {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		for _, name := range names {
			if strings.EqualFold(opt.Name, name) {
				return opt.Value, true
			}
		}
	}
	return "", false
}

// optionNameMatches reports whether `optName` (case-insensitively)
// matches any name in `names`. Used for single-option checks where the
// option is already plucked from a slice (e.g. AlterTableCmd.Option).
func optionNameMatches(optName string, names []string) bool {
	for _, n := range names {
		if strings.EqualFold(optName, n) {
			return true
		}
	}
	return false
}

// columnCharset returns the column-level CHARACTER SET (lowercased)
// from ColumnDef.TypeName.Charset. Empty if no clause.
func columnCharset(col *omniast.ColumnDef) string {
	if col == nil || col.TypeName == nil {
		return ""
	}
	return strings.ToLower(col.TypeName.Charset)
}

// columnCollate returns the column-level COLLATE (lowercased) from
// ColumnDef.TypeName.Collate. Empty if no clause. Note: omni stores
// column-level COLLATE on the DataType, NOT in Constraints[] — the
// ColConstrCollate enum value exists but isn't populated by the parser
// for this form.
func columnCollate(col *omniast.ColumnDef) string {
	if col == nil || col.TypeName == nil {
		return ""
	}
	return strings.ToLower(col.TypeName.Collate)
}

// columnHasComment reports whether the column has a COMMENT clause.
// The value is in col.Comment, but the "clause was written" signal lives
// in Constraints[] as ColConstrComment — needed to distinguish
// "COMMENT ”" (present, empty) from no COMMENT clause at all.
func columnHasComment(col *omniast.ColumnDef) bool {
	if col == nil {
		return false
	}
	for _, c := range col.Constraints {
		if c != nil && c.Type == omniast.ColConstrComment {
			return true
		}
	}
	return false
}

// collectColumnViolations walks an Stmt's CREATE TABLE columns and
// ALTER TABLE ADD/CHANGE/MODIFY COLUMN commands, returning a columnData
// entry for every column where isViolation returns true. Line for
// CREATE TABLE columns is the column's start; for ALTER TABLE commands,
// the top-level statement's start (matching pingcap-typed visitors that
// read `node.OriginTextPosition()` for ALTER TABLE specs).
//
// Used by column-attribute advisors that share the shape "walk columns,
// apply a predicate, emit advice on violations" — currently
// advisor_column_auto_increment_must_integer and
// advisor_column_auto_increment_must_unsigned. Extensible to future
// advisors with the same shape (column_maximum_character_length,
// column_type_disallow_list, etc.). Per-advisor advice formatting and
// the rule-specific predicate stay in the caller.
//
// Not appropriate for advisors with table-level filtering
// (advisor_column_require_default's table-level PK exemption) or those
// that read non-column structures (advisor_column_auto_increment_initial_value's
// table options).
func collectColumnViolations(ostmt Stmt, isViolation func(*omniast.ColumnDef) bool) []columnData {
	if isViolation == nil {
		return nil
	}
	var cols []columnData
	switch n := ostmt.Node.(type) {
	case *omniast.CreateTableStmt:
		if n.Table == nil {
			return nil
		}
		tableName := n.Table.Name
		for _, column := range n.Columns {
			if column == nil {
				continue
			}
			if isViolation(column) {
				cols = append(cols, columnData{
					table:  tableName,
					column: column.Name,
					line:   ostmt.AbsoluteLine(column.Loc.Start),
				})
			}
		}
	case *omniast.AlterTableStmt:
		if n.Table == nil {
			return nil
		}
		tableName := n.Table.Name
		stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			switch cmd.Type {
			case omniast.ATAddColumn:
				for _, column := range addColumnTargets(cmd) {
					if column == nil {
						continue
					}
					if isViolation(column) {
						cols = append(cols, columnData{table: tableName, column: column.Name, line: stmtLine})
					}
				}
			case omniast.ATChangeColumn, omniast.ATModifyColumn:
				if cmd.Column == nil {
					continue
				}
				if isViolation(cmd.Column) {
					cols = append(cols, columnData{table: tableName, column: cmd.Column.Name, line: stmtLine})
				}
			default:
			}
		}
	default:
	}
	return cols
}

// firstAlterCommandMatching returns the index of the first AlterTableCmd
// in n.Commands satisfying matcher, or -1 if none match. Used by ALTER-
// TABLE-only advisors that emit a single advice per statement
// (cardinality-1) on the first triggering command — `disallow_changing`
// and `disallow_changing_order` share this exact shape.
//
// Returns -1 when n is nil, n.Commands is empty, or matcher returns
// false for every non-nil command.
func firstAlterCommandMatching(n *omniast.AlterTableStmt, matcher func(*omniast.AlterTableCmd) bool) int {
	if n == nil || matcher == nil {
		return -1
	}
	for i, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		if matcher(cmd) {
			return i
		}
	}
	return -1
}

// isTimeType reports whether the column type is DATETIME or
// TIMESTAMP. Pingcap dispatched on `mysql.TypeDatetime` /
// `mysql.TypeTimestamp` — distinct type bytes, no unification.
// omni surfaces both as their own DataType.Name with no aliasing.
func isTimeType(dt *omniast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "DATETIME", "TIMESTAMP":
		return true
	default:
		return false
	}
}

// isDefaultCurrentTime reports whether a column has
// `DEFAULT CURRENT_TIMESTAMP` (or one of its synonyms: NOW(),
// LOCALTIME, LOCALTIMESTAMP). The omni AST puts the default
// expression on `col.DefaultValue` (as an ExprNode); a function-call
// default surfaces as `*omniast.FuncCallExpr` with `Name` carrying
// the function token.
func isDefaultCurrentTime(col *omniast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return isCurrentTimeFuncCall(col.DefaultValue)
}

// isOnUpdateCurrentTime reports whether a column has
// `ON UPDATE CURRENT_TIMESTAMP` (or its synonyms). Omni surfaces
// the on-update expression on `col.OnUpdate` — a separate top-level
// field on ColumnDef (NOT inside Constraints[]).
func isOnUpdateCurrentTime(col *omniast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return isCurrentTimeFuncCall(col.OnUpdate)
}

// isCurrentTimeFuncCall checks whether the given expression is a
// function call to one of the CURRENT_TIMESTAMP synonyms:
// CURRENT_TIMESTAMP, NOW, LOCALTIME, LOCALTIMESTAMP. Pingcap
// canonicalized these to lowercase via `FnName.L`; omni keeps the
// user's original case in `FuncCallExpr.Name`, so we compare
// case-insensitively.
func isCurrentTimeFuncCall(expr omniast.ExprNode) bool {
	if expr == nil {
		return false
	}
	fc, ok := expr.(*omniast.FuncCallExpr)
	if !ok {
		return false
	}
	switch strings.ToUpper(fc.Name) {
	case "NOW", "CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP":
		return true
	default:
		return false
	}
}

// isRandFuncCall checks whether the given expression is a
// function call to RAND(). Used by insert_disallow_order_by_rand.
// Pingcap-tidb compared via `FnName.L == ast.Rand` (lowercase
// canonicalization); omni keeps the user's original case in
// `FuncCallExpr.Name`, so we compare case-insensitively. RAND
// accepts an optional integer seed argument; we match regardless
// of arity (matching pingcap's predecessor — which type-asserted
// the function call but didn't inspect Args).
func isRandFuncCall(expr omniast.ExprNode) bool {
	if expr == nil {
		return false
	}
	fc, ok := expr.(*omniast.FuncCallExpr)
	if !ok {
		return false
	}
	return strings.EqualFold(fc.Name, "RAND")
}

// isCharOrBinaryType reports whether the column type is a
// fixed-length character/binary type (CHAR or BINARY). Pingcap-tidb
// dispatched on `mysql.TypeString` which covered BOTH `CHAR` and
// `BINARY` via charset distinction (charset-pair unification, same
// shape as cumulative #18 BLOB/TEXT and #20 TINYINT/BOOLEAN). Omni
// splits to distinct `DataType.Name = "CHAR"` and `"BINARY"`.
// Cumulative #22 — both must be matched to preserve pingcap behavior.
// Mysql analog only matches CHAR (long-standing mysql gap; pre-omni
// mysql used ANTLR token symbols and never matched BINARY either —
// NOT a regression).
func isCharOrBinaryType(dt *omniast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "CHAR", "BINARY":
		return true
	default:
		return false
	}
}

// charLength returns the declared length of a CHAR/BINARY column,
// applying the MySQL default of 1 when no explicit length is given.
// Pingcap's `column.Tp.GetFlen()` returned the canonical default
// width for bare CHAR / BARE BINARY; omni keeps Length=0 in that
// case. Matches both omni-builder behavior and INFORMATION_SCHEMA
// rendering.
//
// Returns 0 for non-CHAR/BINARY types — callers gate on
// isCharOrBinaryType first.
func charLength(dt *omniast.DataType) int {
	if dt == nil || !isCharOrBinaryType(dt) {
		return 0
	}
	if dt.Length == 0 {
		// MySQL default: bare CHAR → CHAR(1); bare BINARY → BINARY(1).
		return 1
	}
	return dt.Length
}

// isIntegerType reports whether the column type is an integer type
// from the perspective of pingcap-tidb's `isInteger` helper. Pingcap
// dispatched on `mysql.TypeTiny`/`TypeShort`/`TypeInt24`/`TypeLong`/`TypeLonglong`
// (5 type bytes), and crucially `mysql.TypeTiny` was shared by TINYINT,
// BOOL, and BOOLEAN (BOOL/BOOLEAN are TINYINT aliases). Omni splits
// the alias group: BOOL/BOOLEAN both normalize to `DataType.Name = "BOOLEAN"`
// (verified empirically), while TINYINT stays as "TINYINT". Match all 7
// omni names (6 covering pingcap's 5 type bytes + "INTEGER" as a
// defensive alias even though omni pre-normalizes INTEGER → INT per
// cumulative #7). Cumulative #20 documents the TINYINT/BOOLEAN split.
//
// NOTE: this diverges from `mysql/utils.go`'s `isIntegerType`,
// which currently omits "BOOLEAN". Pre-omni mysql ANTLR did not treat
// BOOL as TINYINT — that's a long-standing mysql behavior gap, NOT a
// regression introduced by mysql's omni migration. Out of scope here.
func isIntegerType(dt *omniast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "BOOLEAN":
		return true
	default:
		return false
	}
}

// addColumnTargets returns the column definitions added by an ATAddColumn
// cmd. omni populates either cmd.Columns (multi-column ADD COLUMN (...))
// or cmd.Column (single ADD COLUMN); read both defensively.
func addColumnTargets(cmd *omniast.AlterTableCmd) []*omniast.ColumnDef {
	if cmd == nil {
		return nil
	}
	if len(cmd.Columns) > 0 {
		return cmd.Columns
	}
	if cmd.Column != nil {
		return []*omniast.ColumnDef{cmd.Column}
	}
	return nil
}

// needDefault reports whether a column needs a DEFAULT clause to be
// considered "well-formed" by the column_set_default_for_not_null and
// column_require_default rules. Returns false (default not needed) when:
//   - column is auto-increment (auto-generated value)
//   - column is generated (computed from other columns)
//   - column is PRIMARY KEY (auto-incrementing-or-required by other means)
//   - column type is BLOB/TEXT (omni stores TEXT as a separate name from
//     BLOB; pingcap unified them under TypeBlob — both must be checked)
//   - column type is JSON or geometry (no meaningful default)
func needDefault(col *omniast.ColumnDef) bool {
	if col == nil {
		return false
	}
	if col.AutoIncrement || col.Generated != nil {
		return false
	}
	for _, c := range col.Constraints {
		if c != nil && c.Type == omniast.ColConstrPrimaryKey {
			return false
		}
	}
	if col.TypeName == nil {
		return false
	}
	switch strings.ToUpper(col.TypeName.Name) {
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
		"TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"JSON",
		"GEOMETRY", "POINT", "LINESTRING", "POLYGON",
		"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION":
		return false
	}
	return true
}

// stmtsCacheKey is the advisor.Context.Memo key for the per-review
// omni-parse result. All migrated tidb advisors share one parse pass through
// this cache.
const stmtsCacheKey = "tidb.stmts"

// getTiDBNodes returns omni-parsed statements for migrated advisors. The
// dispatcher has already parsed and populated stmt.AST; this helper unwraps the
// nodes and memoizes the result on checkCtx.Memo (one parse per review).
//
// Type switch over stmt.AST:
//   - *tidbparser.AST: omni accepted — collect the node.
//   - default: unknown AST type — warn (a future tidb AST type shouldn't
//     silently drop statements).
func getTiDBNodes(checkCtx advisor.Context) ([]Stmt, error) {
	if cached, ok := checkCtx.Memo(stmtsCacheKey); ok {
		if stmts, typeOK := cached.([]Stmt); typeOK {
			return stmts, nil
		}
	}

	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var result []Stmt
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.Empty || stmt.AST == nil {
			continue
		}
		switch a := stmt.AST.(type) {
		case *tidbparser.AST:
			result = append(result, Stmt{
				Node:     a.Node,
				Text:     a.Text,
				BaseLine: stmt.BaseLine(),
			})
		default:
			slog.Warn("unexpected stmt.AST type for tidb dispatcher",
				slog.String("type", fmt.Sprintf("%T", a)))
		}
	}

	checkCtx.SetMemo(stmtsCacheKey, result)
	return result, nil
}

// indexColumns extracts column names from an omni IndexColumn list.
// Expression-based parts that are not bare column refs are silently
// skipped — e.g., functional indexes like `INDEX idx ((LOWER(name)))`
// contribute no name. Any rule that needs to inspect index expressions
// should not route through this helper.
func indexColumns(cols []*omniast.IndexColumn) []string {
	if len(cols) == 0 {
		return nil
	}
	var names []string
	for _, col := range cols {
		if col == nil {
			continue
		}
		if ref, ok := col.Expr.(*omniast.ColumnRef); ok {
			names = append(names, ref.Column)
		}
	}
	return names
}
