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

// canNull reports whether the given pingcap-AST column may have NULL values
// (i.e. the column has no NOT NULL or PRIMARY KEY constraint).
//
// Moved here from advisor_column_no_null.go during its migration to omni AST.
// Still consumed by un-migrated pingcap-AST advisors (e.g.
// advisor_column_set_default_for_not_null.go). Delete when all consumers are
// migrated to omni AST.
func canNull(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionNotNull || option.Tp == ast.ColumnOptionPrimaryKey {
			return false
		}
	}
	return true
}

// getTiDBOmniNodes extracts omni/tidb AST nodes from the advisor context by
// re-parsing each statement's text with omni. Used by advisors during the
// migration off the native pingcap parser.
//
// While the registered ParseStatementsFunc still returns native pingcap ASTs
// (preserved by Phase 1.5 for backward compat with un-migrated advisors), this
// helper re-parses with omni. Migrated advisors call this; un-migrated
// advisors continue to call getTiDBNodes. After all advisors are migrated and
// the dispatcher is flipped to omni, this helper can be simplified to read
// directly from checkCtx.ParsedStatements without re-parsing.
func getTiDBOmniNodes(checkCtx advisor.Context) ([]OmniStmt, error) {
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
			return nil, errors.Wrap(err, "failed to parse statement with omni/tidb")
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
	return result, nil
}
