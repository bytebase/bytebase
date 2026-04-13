package mssql

import (
	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

type stmtType = int

const (
	stmtTypeUnknown = 1 << iota
	stmtTypeResultSetGenerating
	stmtTypeRowCountGenerating
)

func getStmtType(stmt string) (stmtType, error) {
	stmts, err := parser.ParseTSQLOmni(stmt)
	if err != nil {
		return stmtTypeUnknown, err
	}

	var nodes []ast.Node
	for _, s := range stmts {
		if s.Empty() {
			continue
		}
		nodes = append(nodes, s.AST)
	}

	switch len(nodes) {
	case 0:
		return stmtTypeUnknown, nil
	case 1:
		return classifyStmtType(nodes[0])
	default:
		return stmtTypeUnknown, errors.Errorf("expected exactly 1 statement, got %d", len(nodes))
	}
}

func classifyStmtType(node ast.Node) (stmtType, error) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		if parser.HasSelectInto(n) {
			// SELECT ... INTO materialises a new table — no result set, only row count.
			return stmtTypeRowCountGenerating, nil
		}
		return stmtTypeResultSetGenerating | stmtTypeRowCountGenerating, nil

	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt,
		*ast.BulkInsertStmt, *ast.InsertBulkStmt, *ast.CopyIntoStmt,
		*ast.ReadtextStmt, *ast.WritetextStmt, *ast.UpdatetextStmt,
		*ast.ReceiveStmt, *ast.PredictStmt:
		return stmtTypeRowCountGenerating, nil

	case *ast.IfStmt, *ast.WhileStmt, *ast.BeginEndStmt, *ast.TryCatchStmt,
		*ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.GotoStmt,
		*ast.LabelStmt, *ast.WaitForStmt:
		return stmtTypeUnknown, errors.Errorf("unsupported control flow statement")

	default:
		return stmtTypeUnknown, nil
	}
}
