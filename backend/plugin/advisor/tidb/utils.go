// package tidb implements the SQL advisor rules for MySQL.
//
//nolint:unused // Kept for PingCap-based TiDB advisors during the omni migration.
package tidb

import (
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	omniast "github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

// getTiDBNodes extracts pingcap-AST nodes for un-migrated advisors.
//
// On a PingCapASTProvider whose AsPingCapAST returns (nil, false) — i.e.
// the bridge tried and pingcap rejected the statement — the statement is
// skipped, not surfaced as an error. A non-provider, non-*AST input is
// still surfaced as an engine-mismatch error.
func getTiDBNodes(checkCtx advisor.Context) ([]ast.StmtNode, error) {
	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var stmtNodes []ast.StmtNode
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		tidbAST, ok := tidbparser.GetTiDBAST(stmt.AST)
		if !ok {
			if _, isProvider := stmt.AST.(tidbparser.PingCapASTProvider); isProvider {
				continue
			}
			return nil, errors.New("AST type mismatch: expected TiDB parser result")
		}
		stmtNodes = append(stmtNodes, tidbAST.Node)
	}
	return stmtNodes, nil
}

// OmniStmt bundles an omni/tidb AST node with the source text and base line of
// the statement it came from. The base line is needed to convert byte offsets
// inside Text into absolute line numbers in the original full SQL.
type OmniStmt struct {
	Node     omniast.Node
	Text     string
	BaseLine int // 0-based line index of the first line of Text in the original SQL
}

// AbsoluteLine converts a byte offset within s.Text into a 1-based line number
// in the original SQL.
func (s OmniStmt) AbsoluteLine(byteOffset int) int {
	pos := tidbparser.ByteOffsetToRunePosition(s.Text, byteOffset)
	return s.BaseLine + int(pos.Line)
}

// TrimmedText returns Text with surrounding whitespace removed. Suitable
// for embedding the statement text into advice content; raw Text may
// include leading/trailing newlines from the original multi-statement
// split.
func (s OmniStmt) TrimmedText() string {
	return strings.TrimSpace(s.Text)
}

// FirstTokenLine returns the 1-based absolute line of the first
// non-whitespace character in s.Text. Matches pingcap's
// OriginTextPosition: pingcap's lexer strips leading whitespace but
// keeps comments as part of the statement, so its reported line points
// at the first comment OR keyword. Used as the StartPosition for
// statement-level advices.
func (s OmniStmt) FirstTokenLine() int {
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
	omniOptionNamesCharset = []string{"CHARACTER SET", "CHARSET"}
	omniOptionNamesCollate = []string{"COLLATE"}
	omniOptionNamesComment = []string{"COMMENT"}
)

// omniDatabaseOption returns the lowercased value of the first option
// whose Name matches any in `names` (case-insensitive), or "" if none
// match. For case-insensitive identifiers (charset/collation names);
// user-content values like COMMENT should use *Present variants.
func omniDatabaseOption(opts []*omniast.DatabaseOption, names []string) string {
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

// omniTableOption returns the lowercased value of the first option whose
// Name matches any in `names` (case-insensitive), or "" if none match.
// Case-insensitive-identifier semantics (see omniDatabaseOption).
func omniTableOption(opts []*omniast.TableOption, names []string) string {
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

// omniTableOptionPresent reports whether any option whose Name matches
// any in `names` is in the slice; returns its raw value (case preserved —
// user content). Distinguishes "no clause" from "clause with empty value"
// (e.g. COMMENT ”).
func omniTableOptionPresent(opts []*omniast.TableOption, names []string) (string, bool) {
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

// omniOptionNameMatches reports whether `optName` (case-insensitively)
// matches any name in `names`. Used for single-option checks where the
// option is already plucked from a slice (e.g. AlterTableCmd.Option).
func omniOptionNameMatches(optName string, names []string) bool {
	for _, n := range names {
		if strings.EqualFold(optName, n) {
			return true
		}
	}
	return false
}

// omniColumnCharset returns the column-level CHARACTER SET (lowercased)
// from ColumnDef.TypeName.Charset. Empty if no clause.
func omniColumnCharset(col *omniast.ColumnDef) string {
	if col == nil || col.TypeName == nil {
		return ""
	}
	return strings.ToLower(col.TypeName.Charset)
}

// omniColumnCollate returns the column-level COLLATE (lowercased) from
// ColumnDef.TypeName.Collate. Empty if no clause. Note: omni stores
// column-level COLLATE on the DataType, NOT in Constraints[] — the
// ColConstrCollate enum value exists but isn't populated by the parser
// for this form.
func omniColumnCollate(col *omniast.ColumnDef) string {
	if col == nil || col.TypeName == nil {
		return ""
	}
	return strings.ToLower(col.TypeName.Collate)
}

// omniColumnHasComment reports whether the column has a COMMENT clause.
// The value is in col.Comment, but the "clause was written" signal lives
// in Constraints[] as ColConstrComment — needed to distinguish
// "COMMENT ”" (present, empty) from no COMMENT clause at all.
func omniColumnHasComment(col *omniast.ColumnDef) bool {
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

// omniDataTypeNameCompact returns a compact, lowercase type-name string
// for use in advice content + allowlist comparisons. Mirrors the mysql
// helper of the same name (mysql/utils_omni.go). Length/scale info is
// intentionally omitted — pingcap's `tidbparser.TypeString` and the
// canonical fixture rendering both elide it (e.g. `varchar(5)` → "varchar").
func omniDataTypeNameCompact(dt *omniast.DataType) string {
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
func collectIndexFamilyCreateTable(ostmt OmniStmt, n *omniast.CreateTableStmt, cb indexFamilyCallbacks) []pkData {
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
func collectIndexFamilyAlterTable(ostmt OmniStmt, n *omniast.AlterTableStmt, cb indexFamilyCallbacks) []pkData {
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

// omniNeedDefault reports whether a column needs a DEFAULT clause to be
// considered "well-formed" by the column_set_default_for_not_null and
// column_require_default rules. Returns false (default not needed) when:
//   - column is auto-increment (auto-generated value)
//   - column is generated (computed from other columns)
//   - column is PRIMARY KEY (auto-incrementing-or-required by other means)
//   - column type is BLOB/TEXT (omni stores TEXT as a separate name from
//     BLOB; pingcap unified them under TypeBlob — both must be checked)
//   - column type is JSON or geometry (no meaningful default)
func omniNeedDefault(col *omniast.ColumnDef) bool {
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

// omniStmtsCacheKey is the advisor.Context.Memo key for the per-review
// omni-parse result. All migrated tidb advisors share one parse pass through
// this cache.
const omniStmtsCacheKey = "tidb.omniStmts"

// getTiDBOmniNodes returns omni-parsed statements for migrated advisors.
//
// Two invariants:
//   - Single-parse-per-review: result is cached on checkCtx.Memo, so all
//     migrated advisors in one review share one parse pass.
//   - Soft-fail per statement: omni parse errors are logged at debug and
//     the statement is skipped; the review never breaks on grammar gaps.
func getTiDBOmniNodes(checkCtx advisor.Context) ([]OmniStmt, error) {
	if cached, ok := checkCtx.Memo(omniStmtsCacheKey); ok {
		if stmts, typeOK := cached.([]OmniStmt); typeOK {
			return stmts, nil
		}
	}

	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var result []OmniStmt
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.Empty {
			continue
		}
		list, err := tidbparser.ParseTiDBOmni(stmt.Text)
		if err != nil {
			slog.Debug("omni/tidb parse failed; skipping statement for omni-aware advisors",
				slog.String("error", err.Error()),
			)
			continue
		}
		if list == nil {
			continue
		}
		for _, item := range list.Items {
			result = append(result, OmniStmt{
				Node:     item,
				Text:     stmt.Text,
				BaseLine: stmt.BaseLine(),
			})
		}
	}

	checkCtx.SetMemo(omniStmtsCacheKey, result)
	return result, nil
}

// indexMetaData captures naming metadata used by the index/UK/FK convention
// rules. Plain Go fields, AST-agnostic — shared between all 3 advisors that
// were previously coupled via this struct in advisor_naming_index_convention.go.
type indexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
	line      int
}

// getTemplateRegexp formats the template as regex by substituting tokens.
// Shared by the index/UK/FK naming convention rules.
func getTemplateRegexp(template string, templateList []string, tokens map[string]string) (*regexp.Regexp, error) {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}
	return regexp.Compile(template)
}

// omniIndexColumns extracts column names from an omni IndexColumn list.
// Expression-based parts that are not bare column refs are silently
// skipped — e.g., functional indexes like `INDEX idx ((LOWER(name)))`
// contribute no name to the column-list substitution. This matches the
// mysql omni analog and the pingcap-typed naming rules' historical
// "name-only" behavior; any future rule that needs to inspect index
// expressions should not route through this helper.
func omniIndexColumns(cols []*omniast.IndexColumn) []string {
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

// namingRuleConfig parameterizes the naming-convention rule scaffold for the
// index, unique-key, and foreign-key advisors. Only the per-rule labels and
// the mismatch advice code differ — everything else (payload validation,
// regex match, length check, advice emission) is shared.
type namingRuleConfig struct {
	mismatchCode       code.Code
	typeNoun           string // "Index" / "Unique key" / "Foreign key" — embedded in advice content
	internalErrorTitle string
}

// runNamingConventionRule is the shared scaffold for the index/UK/FK naming
// rules. The collect closure returns []*indexMetaData, so all callers are
// coupled to that 4-field shape. If a future naming rule needs additional
// per-finding fields (e.g. an isUnique flag or a per-finding line distinct
// from the metadata's), expect either widening indexMetaData (everyone
// pays) or replacing the closure return type (breaks the helper). The
// current shape is the lowest common denominator across the 3 rules.
//
// Behavior: parses the naming payload, walks each statement through `collect`,
// and emits regex-mismatch + length-overflow advices.
//
// The advice content is byte-identical to the pre-extraction inline form
// (`"<noun> in table ..."` and `"<noun> `<name>` in table ..."`), so existing
// fixture coverage is the safety net for this refactor — no fixture updates
// needed.
//
// Mysql analogs left similar duplication inline (pre-Sonar-gate); we extract
// here because the trio is new code and Sonar gates duplication on new code.
// Future naming-rule migrations (naming_table, naming_column,
// naming_auto_increment_column) reuse this scaffold.
func runNamingConventionRule(
	checkCtx advisor.Context,
	cfg namingRuleConfig,
	collect func(OmniStmt) []*indexMetaData,
) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}
	formatStr := namingPayload.Format
	templateList, _ := advisor.ParseTemplateTokens(formatStr)
	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}
	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		for _, indexData := range collect(ostmt) {
			regex, err := getTemplateRegexp(formatStr, templateList, indexData.metaData)
			if err != nil {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Code:    code.Internal.Int32(),
					Title:   cfg.internalErrorTitle,
					Content: fmt.Sprintf("%q meet internal error %q", ostmt.TrimmedText(), err.Error()),
				})
				continue
			}
			if !regex.MatchString(indexData.indexName) {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          cfg.mismatchCode.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("%s in table `%s` mismatches the naming convention, expect %q but found `%s`", cfg.typeNoun, indexData.tableName, regex, indexData.indexName),
					StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
				})
			}
			if maxLength > 0 && len(indexData.indexName) > maxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          cfg.mismatchCode.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("%s `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", cfg.typeNoun, indexData.indexName, indexData.tableName, maxLength),
					StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
				})
			}
		}
	}
	return adviceList, nil
}
