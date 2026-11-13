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
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/store/model"
)

type columnSet map[string]bool

func newColumnSet(columns []string) columnSet {
	res := make(columnSet)
	for _, col := range columns {
		res[col] = true
	}
	return res
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

// constraintAdviceName returns the constraint name suitable for
// embedding in advice content. Falls back to "PRIMARY" for unnamed
// PRIMARY KEY constraints (cumulative #28: pingcap-tidb accepted the
// non-standard `PRIMARY KEY index_name (cols)` extension and captured
// the index_name; omni follows standard MySQL grammar where PRIMARY
// KEY doesn't accept an index_name and silently drops it. "PRIMARY"
// is MySQL's canonical internal name for the primary key — better UX
// than the empty backticks the raw `c.Name` would produce).
//
// Promoted to utils.go in batch 18 (originally file-local in batch
// 17's advisor_index_key_number_limit.go) when a second consumer
// (advisor_index_no_duplicate_column) needed the same fallback.
func constraintAdviceName(c *omniast.Constraint) string {
	if c == nil {
		return ""
	}
	if name := constraintName(c); name != "" {
		return name
	}
	if c.Type == omniast.ConstrPrimaryKey {
		return "PRIMARY"
	}
	return ""
}

// constraintName returns the CONSTRAINT symbol, else the name written after
// FOREIGN KEY, which omni keeps in IndexName.
func constraintName(c *omniast.Constraint) string {
	if c.Name != "" {
		return c.Name
	}
	return c.IndexName
}

// dataTypeNameCompact returns a compact, lowercase type-name string
// for use in advice content + allowlist comparisons. Mirrors the mysql
// helper of the same name (mysql/utils.go). Length/scale info is
// intentionally omitted — pingcap's `tidbparser.TypeString` and the
// canonical fixture rendering both elide it (e.g. `varchar(5)` → "varchar").
func dataTypeNameCompact(dt *omniast.DataType) string {
	if dt == nil {
		return ""
	}
	return strings.ToLower(dt.Name)
}

// indexFamilyCallbacks parameterizes the shared CREATE TABLE / ALTER TABLE
// walk used by the index-family advisors (advisor_index_pk_type,
// advisor_index_type_no_blob, advisor_index_primary_key_type_allowlist).
// The three advisors share scaffolding — same top-level type switch, same
// AlterTable command-arm dispatch, same `pkData` accumulator — but diverge
// at every callback site (which constraint types they care about, which
// column-level constraint flags violations, what content the advice emits).
// Encoding the spine here removes ~30 lines of duplication per advisor;
// the per-rule logic stays in each advisor's callbacks.
//
// Each callback returns the [`pkData`] entries this command contributed,
// so the helper concatenates results without owning advice formatting
// (which differs per advisor and stays in-file).
type indexFamilyCallbacks struct {
	// onColumn fires for each column observed in CREATE TABLE columns and
	// ALTER TABLE ADD COLUMN. Implementations typically record the column
	// in the per-checker tracker (so ADD INDEX on a just-added column can
	// resolve its type later) and emit a violation if the column itself
	// declares a triggering inline constraint (e.g. PRIMARY KEY / UNIQUE).
	onColumn func(tableName string, line int, col *omniast.ColumnDef) []pkData
	// onConstraint fires for each top-level constraint observed: CREATE
	// TABLE table-level constraints and ALTER TABLE ADD CONSTRAINT / ADD
	// INDEX / ADD PRIMARY KEY / etc. Empirically tidb omni emits all
	// `ALTER TABLE ADD ...` forms via ATAddConstraint; ATAddIndex is also
	// dispatched here for sibling parity (cumulative #17).
	onConstraint func(tableName string, line int, constraint *omniast.Constraint) []pkData
	// onChangeColumn fires for ALTER TABLE CHANGE/MODIFY COLUMN. The
	// helper resolves the OLD column name (cmd.Name for CHANGE COLUMN
	// when set; falling back to the new column's name for MODIFY COLUMN
	// or unnamed CHANGE) so callbacks can clean up tracker state before
	// re-recording.
	onChangeColumn func(tableName string, oldColumnName string, line int, newCol *omniast.ColumnDef) []pkData
}

// collectIndexFamilyCreateTable walks a CreateTableStmt's columns and
// table-level constraints, dispatching each to the per-advisor callbacks.
// Line positions: column line is the column's start; constraint line is
// the constraint's start. Returns the concatenated pkData; advice
// formatting/emission stays in the caller.
func collectIndexFamilyCreateTable(ostmt Stmt, n *omniast.CreateTableStmt, cb indexFamilyCallbacks) []pkData {
	if n == nil || n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	var result []pkData
	for _, column := range n.Columns {
		if column == nil {
			continue
		}
		if cb.onColumn != nil {
			result = append(result, cb.onColumn(tableName, ostmt.AbsoluteLine(column.Loc.Start), column)...)
		}
	}
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		if cb.onConstraint != nil {
			result = append(result, cb.onConstraint(tableName, ostmt.AbsoluteLine(constraint.Loc.Start), constraint)...)
		}
	}
	return result
}

// collectIndexFamilyAlterTable walks an AlterTableStmt's commands,
// dispatching ATAddColumn / ATAddConstraint+ATAddIndex / ATChangeColumn+
// ATModifyColumn to the per-advisor callbacks. Other command types are
// ignored. The line for every command is the top-level statement's start
// (matching pingcap-typed visitors that read `node.OriginTextPosition()`
// for ALTER TABLE specs). Returns the concatenated pkData.
func collectIndexFamilyAlterTable(ostmt Stmt, n *omniast.AlterTableStmt, cb indexFamilyCallbacks) []pkData {
	if n == nil || n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
	var result []pkData
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case omniast.ATAddColumn:
			if cb.onColumn == nil {
				continue
			}
			for _, column := range addColumnTargets(cmd) {
				result = append(result, cb.onColumn(tableName, stmtLine, column)...)
			}
		case omniast.ATAddConstraint, omniast.ATAddIndex:
			// Empirically tidb omni emits only ATAddConstraint for
			// `ALTER TABLE ... ADD …` forms (bare and named); ATAddIndex
			// is reserved (cumulative #17). Dual arm preserved for sibling
			// parity with the naming-convention advisors and forward-compat
			// against grammar evolution.
			if cmd.Constraint != nil && cb.onConstraint != nil {
				result = append(result, cb.onConstraint(tableName, stmtLine, cmd.Constraint)...)
			}
		case omniast.ATChangeColumn, omniast.ATModifyColumn:
			if cmd.Column == nil || cb.onChangeColumn == nil {
				continue
			}
			newCol := cmd.Column
			oldColumnName := newCol.Name
			if cmd.Type == omniast.ATChangeColumn && cmd.Name != "" {
				// CHANGE COLUMN: cmd.Name is the OLD column name.
				oldColumnName = cmd.Name
			}
			result = append(result, cb.onChangeColumn(tableName, oldColumnName, stmtLine, newCol)...)
		default:
		}
	}
	return result
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

// stmtNodes returns the nodes of stmts, index for index, for a rule that
// judges the statements together.
func stmtNodes(stmts []Stmt) []omniast.Node {
	nodes := make([]omniast.Node, len(stmts))
	for i, stmt := range stmts {
		nodes[i] = stmt.Node
	}
	return nodes
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

// v1reviewSchema returns the snapshot md holds for an omni rule, an empty
// one when md is nil.
func v1reviewSchema(md *model.DatabaseMetadata) v1review.Schema {
	if md == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
}
