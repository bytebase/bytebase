package tidb

import (
	"github.com/bytebase/omni/tidb/ast"
)

// exprLoc returns the source range of an expression node. omni stores the
// range in each node's named Loc field, which is not method-promoted, so the
// range is read per type.
func exprLoc(expr ast.ExprNode) (ast.Loc, bool) {
	switch e := expr.(type) {
	case *ast.ColumnRef:
		return e.Loc, true
	case *ast.BinaryExpr:
		return e.Loc, true
	case *ast.UnaryExpr:
		return e.Loc, true
	case *ast.FuncCallExpr:
		return e.Loc, true
	case *ast.CastExpr:
		return e.Loc, true
	case *ast.ConvertExpr:
		return e.Loc, true
	case *ast.InExpr:
		return e.Loc, true
	case *ast.LikeExpr:
		return e.Loc, true
	case *ast.IsExpr:
		return e.Loc, true
	case *ast.BetweenExpr:
		return e.Loc, true
	case *ast.CaseExpr:
		return e.Loc, true
	case *ast.ParenExpr:
		return e.Loc, true
	case *ast.RowExpr:
		return e.Loc, true
	case *ast.SubqueryExpr:
		return e.Loc, true
	case *ast.ExistsExpr:
		return e.Loc, true
	case *ast.MatchExpr:
		return e.Loc, true
	case *ast.IntervalExpr:
		return e.Loc, true
	case *ast.CollateExpr:
		return e.Loc, true
	case *ast.ExtractExpr:
		return e.Loc, true
	case *ast.MemberOfExpr:
		return e.Loc, true
	case *ast.PositionExpr:
		return e.Loc, true
	case *ast.DefaultExpr:
		return e.Loc, true
	case *ast.VariableRef:
		return e.Loc, true
	case *ast.IntLit:
		return e.Loc, true
	case *ast.FloatLit:
		return e.Loc, true
	case *ast.StringLit:
		return e.Loc, true
	case *ast.BoolLit:
		return e.Loc, true
	case *ast.NullLit:
		return e.Loc, true
	case *ast.HexLit:
		return e.Loc, true
	case *ast.BitLit:
		return e.Loc, true
	case *ast.TemporalLit:
		return e.Loc, true
	case *ast.StarExpr:
		return e.Loc, true
	default:
		return ast.NoLoc(), false
	}
}
