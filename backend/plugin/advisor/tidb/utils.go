// package tidb implements the SQL advisor rules for MySQL.
package tidb

import (
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/types"
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

// tableNewColumn tracks per-statement column definitions by name, scoped
// to the un-migrated pingcap-AST index advisors. Delete when those migrate.
// Tracked: https://linear.app/bytebase/issue/BYT-9395
type columnNameToColumnDef map[string]*ast.ColumnDef
type tableNewColumn map[string]columnNameToColumnDef

func (t tableNewColumn) set(tableName, columnName string, colDef *ast.ColumnDef) {
	if _, ok := t[tableName]; !ok {
		t[tableName] = make(columnNameToColumnDef)
	}
	t[tableName][columnName] = colDef
}

func (t tableNewColumn) get(tableName, columnName string) (colDef *ast.ColumnDef, ok bool) {
	if _, ok := t[tableName]; !ok {
		return nil, false
	}
	col, ok := t[tableName][columnName]
	return col, ok
}

func (t tableNewColumn) delete(tableName, columnName string) {
	if _, ok := t[tableName]; !ok {
		return
	}
	delete(t[tableName], columnName)
}

func restoreNode(node ast.Node, flag format.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func needDefault(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		switch option.Tp {
		case ast.ColumnOptionAutoIncrement, ast.ColumnOptionPrimaryKey, ast.ColumnOptionGenerated:
			return false
		default:
			// Other options
		}
	}

	if types.IsTypeBlob(column.Tp.GetType()) {
		return false
	}
	switch column.Tp.GetType() {
	case mysql.TypeJSON, mysql.TypeGeometry:
		return false
	default:
		// Other types can have default values
	}
	return true
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

// canNull reports whether a pingcap-AST column may hold NULL (no NOT NULL
// or PRIMARY KEY option). Scoped to un-migrated advisors; delete when
// advisor_column_set_default_for_not_null.go migrates.
// Tracked: https://linear.app/bytebase/issue/BYT-9362
func canNull(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionNotNull || option.Tp == ast.ColumnOptionPrimaryKey {
			return false
		}
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
