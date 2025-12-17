// package tidb implements the SQL advisor rules for MySQL.
package tidb

import (
	"slices"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
func getTiDBNodes(checkCtx advisor.Context) ([]ast.StmtNode, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}
	var stmtNodes []ast.StmtNode
	for _, unifiedAST := range checkCtx.AST {
		tidbAST, ok := tidbparser.GetTiDBAST(unifiedAST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected TiDB parser result")
		}
		stmtNodes = append(stmtNodes, tidbAST.Node)
	}
	return stmtNodes, nil
}

// ParsedStatementInfo contains all info needed for checking a single statement.
type ParsedStatementInfo struct {
	Node     ast.StmtNode
	BaseLine int
	Text     string
}

// getParsedStatements extracts statement info from the advisor context.
// This is the preferred way to access statements - use stmtInfo.Text directly
// instead of calling node.Text().
// nolint:unused
func getParsedStatements(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
	if checkCtx.ParsedStatements == nil {
		// Fallback to old behavior for backward compatibility
		return getParsedStatementsFromAST(checkCtx)
	}

	var results []ParsedStatementInfo
	for _, stmt := range checkCtx.ParsedStatements {
		// Skip empty statements (no AST)
		if stmt.AST == nil {
			continue
		}
		tidbAST, ok := tidbparser.GetTiDBAST(stmt.AST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected TiDB parser result")
		}
		results = append(results, ParsedStatementInfo{
			Node:     tidbAST.Node,
			BaseLine: stmt.BaseLine,
			Text:     stmt.Text,
		})
	}
	return results, nil
}

// getParsedStatementsFromAST is the fallback when ParsedStatements is not available.
// Deprecated: Use getParsedStatements with ParsedStatements field instead.
// nolint:unused
func getParsedStatementsFromAST(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}

	var results []ParsedStatementInfo
	for _, unifiedAST := range checkCtx.AST {
		tidbAST, ok := tidbparser.GetTiDBAST(unifiedAST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected TiDB parser result")
		}
		// Use node.Text() as fallback since Text field is not available
		results = append(results, ParsedStatementInfo{
			Node:     tidbAST.Node,
			BaseLine: getLineOffset(tidbAST.StartPosition),
			Text:     tidbAST.Node.Text(),
		})
	}
	return results, nil
}

// getLineOffset converts a 1-based Position to 0-based line offset.
// nolint:unused
func getLineOffset(pos *storepb.Position) int {
	if pos == nil {
		return 0
	}
	return int(pos.Line) - 1
}
