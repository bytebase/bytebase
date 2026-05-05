// package tidb implements the SQL advisor rules for MySQL.
package tidb

import (
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/types"
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

// getTiDBNodes extracts TiDB AST nodes from the advisor context.
//
// Soft-fail on bridge miss: when stmt.AST is a PingCapASTProvider
// (post-dispatcher-flip *OmniAST) whose AsPingCapAST() returns (nil, false),
// the statement is skipped rather than surfaced as an error. This honors
// Phase 1.5 invariant #2 from the omni-tidb completion plan — un-migrated
// advisors emit no advice for the skipped statement; the review continues.
// The bridge has already logged the parse failure at debug level
// (omni.go:AsPingCapAST).
//
// True engine-mismatch (stmt.AST is some unrelated type, neither *tidb.AST
// nor a PingCapASTProvider) is still surfaced as an error — that's a
// programmer error in registration, not a soft-fail case.
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
			// Bridge miss → soft-fail (skip). Engine mismatch → error.
			// PingCapASTProvider implementations route through GetTiDBAST →
			// AsPingCapAST; a (nil, false) here means the bridge tried and
			// pingcap rejected the statement. Skip rather than abort the rule.
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

// canNull reports whether the given pingcap-AST column may have NULL values
// (i.e. the column has no NOT NULL or PRIMARY KEY constraint).
//
// Moved here from advisor_column_no_null.go during its migration to omni AST.
// Still consumed by advisor_column_set_default_for_not_null.go (un-migrated).
// Delete when set_default_for_not_null migrates to omni AST.
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

// getTiDBOmniNodes extracts omni/tidb AST nodes from the advisor context by
// re-parsing each statement's text with omni. Used by advisors during the
// migration off the native pingcap parser.
//
// While the registered ParseStatementsFunc still returns native pingcap ASTs
// (preserved by Phase 1.5 for backward compat with un-migrated advisors), this
// helper re-parses with omni. Migrated advisors call this; un-migrated
// advisors continue to call getTiDBNodes.
//
// Phase 1.5 invariants enforced here:
//
//   - Single-parse-per-review: result is cached on advisor.Context.Memo so
//     subsequent advisors in the same review reuse the parse work. Cost stays
//     1× regardless of how many advisors migrate.
//   - Soft-fail on omni grammar gaps: a statement that fails to parse with
//     omni is logged and skipped. Other statements (and other advisors) keep
//     working. This protects review continuity while omni grammar catches up
//     to pingcap on Tier 4 deferred features (FLASHBACK, SEQUENCE, BATCH DML,
//     etc. — see plans/2026-04-23-omni-tidb-completion-plan.md §Phase 2).
//
// After all advisors are migrated and the dispatcher is flipped to omni, this
// helper can be simplified to read directly from checkCtx.ParsedStatements
// without re-parsing.
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
			// Soft-fail: omni may not yet support every TiDB grammar feature
			// pingcap accepts. Skip this statement; advisors emit no advice
			// for it but the review continues. Promote to higher log level
			// when this signals a real omni regression vs. an expected gap.
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
