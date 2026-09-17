package mssql

import (
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

type stmtType = int

const (
	stmtTypeUnknown = 1 << iota
	// stmtTypeResultSetGenerating returns exactly one result set.
	stmtTypeResultSetGenerating
	stmtTypeRowCountGenerating
	// stmtTypeProcedure returns any number of result sets and row counts.
	stmtTypeProcedure
	// stmtTypeOutput is a data modification whose OUTPUT clause returns rows.
	stmtTypeOutput
)

func getStmtType(stmt string) (stmtType, error) {
	stmts, err := parser.ParseTSQL(stmt)
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
		if assignsVariables(n) {
			// SELECT @v = ... returns no result set or row count.
			return stmtTypeUnknown, nil
		}
		return stmtTypeResultSetGenerating | stmtTypeRowCountGenerating, nil

	case *ast.InsertStmt:
		return dmlStmtType(n.OutputClause), nil
	case *ast.UpdateStmt:
		return dmlStmtType(n.OutputClause), nil
	case *ast.DeleteStmt:
		return dmlStmtType(n.OutputClause), nil
	case *ast.MergeStmt:
		return dmlStmtType(n.OutputClause), nil

	case *ast.BulkInsertStmt, *ast.InsertBulkStmt, *ast.CopyIntoStmt,
		*ast.ReadtextStmt, *ast.WritetextStmt, *ast.UpdatetextStmt,
		*ast.ReceiveStmt, *ast.PredictStmt:
		return stmtTypeRowCountGenerating, nil

	case *ast.ExecStmt:
		return stmtTypeProcedure, nil

	case *ast.SetOptionStmt:
		if strings.EqualFold(n.Option, "NOEXEC") || strings.EqualFold(n.Option, "PARSEONLY") {
			// These skip statements without an error, so the result sets that do
			// arrive could not be matched to their statements.
			return stmtTypeUnknown, errors.Errorf("SET %s is not supported", strings.ToUpper(n.Option))
		}
		return stmtTypeUnknown, nil

	case *ast.IfStmt, *ast.WhileStmt, *ast.BeginEndStmt, *ast.TryCatchStmt,
		*ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.GotoStmt,
		*ast.LabelStmt, *ast.WaitForStmt:
		return stmtTypeUnknown, errors.Errorf("unsupported control flow statement")

	default:
		return stmtTypeUnknown, nil
	}
}

// dmlStmtType classifies a data modification, which also returns the affected
// rows as a result set when its OUTPUT clause has no INTO.
func dmlStmtType(output *ast.OutputClause) stmtType {
	if output != nil && output.IntoTable == nil {
		return stmtTypeResultSetGenerating | stmtTypeRowCountGenerating | stmtTypeOutput
	}
	return stmtTypeRowCountGenerating
}

func assignsVariables(sel *ast.SelectStmt) bool {
	if sel.TargetList == nil {
		return false
	}
	for _, item := range sel.TargetList.Items {
		if target, ok := item.(*ast.ResTarget); ok {
			if _, ok := target.Val.(*ast.SelectAssign); ok {
				return true
			}
		}
	}
	return false
}
