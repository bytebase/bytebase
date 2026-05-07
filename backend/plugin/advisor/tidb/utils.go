// package tidb implements the SQL advisor rules for MySQL.
package tidb

import (
	"log/slog"
	"slices"
	"strings"

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
