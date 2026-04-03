package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementJoinStrictColumnAttrsAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_JOIN_STRICT_COLUMN_ATTRS, &StatementJoinStrictColumnAttrsAdvisor{})
}

type StatementJoinStrictColumnAttrsAdvisor struct {
}

func (*StatementJoinStrictColumnAttrsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &joinStrictColumnAttrsOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}
	if checkCtx.DBSchema != nil {
		rule.dbMetadata = model.NewDatabaseMetadata(checkCtx.DBSchema, nil, nil, storepb.Engine_MYSQL, checkCtx.IsObjectCaseSensitive)
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

// SourceTable represents a table in the FROM clause.
type SourceTable struct {
	Name  string
	Alias string
}

// ColumnAttr represents column attributes for join checking.
type ColumnAttr struct {
	Table     string
	Column    string
	Type      string
	Charset   string
	Collation string
}

type joinStrictColumnAttrsOmniRule struct {
	OmniBaseRule
	dbMetadata *model.DatabaseMetadata
}

func (*joinStrictColumnAttrsOmniRule) Name() string {
	return "StatementJoinStrictColumnAttrsRule"
}

func (r *joinStrictColumnAttrsOmniRule) OnStatement(node ast.Node) {
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return
	}
	r.checkSelect(sel)
}

func (r *joinStrictColumnAttrsOmniRule) checkSelect(sel *ast.SelectStmt) {
	if sel == nil {
		return
	}
	if sel.SetOp != ast.SetOpNone {
		r.checkSelect(sel.Left)
		r.checkSelect(sel.Right)
		return
	}

	// Collect source tables from FROM clause.
	var sourceTables []SourceTable
	for _, from := range sel.From {
		ast.Inspect(from, func(n ast.Node) bool {
			t, ok := n.(*ast.TableRef)
			if !ok {
				return true
			}
			st := SourceTable{Name: t.Name}
			if t.Alias != "" {
				st.Alias = t.Alias
			}
			sourceTables = append(sourceTables, st)
			return false
		})
	}

	// Check joins for column attribute mismatches.
	for _, from := range sel.From {
		r.checkJoins(from, sourceTables)
	}
}

func (r *joinStrictColumnAttrsOmniRule) checkJoins(te ast.TableExpr, sourceTables []SourceTable) {
	if te == nil {
		return
	}
	join, ok := te.(*ast.JoinClause)
	if !ok {
		return
	}

	// Check USING condition.
	if using, ok := join.Condition.(*ast.UsingCondition); ok {
		// Get the right table name.
		rightTable := r.getTableName(join.Right)
		for _, colName := range using.Columns {
			rightCol := &ColumnAttr{Table: rightTable, Column: colName}
			for _, st := range sourceTables {
				if st.Name == rightTable || st.Alias == rightTable {
					continue
				}
				r.checkColumnAttrs(&ColumnAttr{Table: st.Name, Column: colName}, rightCol)
			}
		}
	}

	// Check ON condition for equality comparisons.
	if on, ok := join.Condition.(*ast.OnCondition); ok {
		r.checkOnExpr(on.Expr)
	}

	r.checkJoins(join.Left, sourceTables)
	r.checkJoins(join.Right, sourceTables)
}

func (*joinStrictColumnAttrsOmniRule) getTableName(te ast.TableExpr) string {
	if te == nil {
		return ""
	}
	switch t := te.(type) {
	case *ast.TableRef:
		return t.Name
	default:
		return ""
	}
}

func (r *joinStrictColumnAttrsOmniRule) checkOnExpr(expr ast.ExprNode) {
	if expr == nil {
		return
	}
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return
	}
	if bin.Op == ast.BinOpAnd {
		r.checkOnExpr(bin.Left)
		r.checkOnExpr(bin.Right)
		return
	}
	if bin.Op != ast.BinOpEq {
		return
	}
	leftCol := r.extractColumnAttr(bin.Left)
	rightCol := r.extractColumnAttr(bin.Right)
	r.checkColumnAttrs(leftCol, rightCol)
}

func (*joinStrictColumnAttrsOmniRule) extractColumnAttr(expr ast.ExprNode) *ColumnAttr {
	ref, ok := expr.(*ast.ColumnRef)
	if !ok {
		return nil
	}
	if ref.Table == "" || ref.Column == "" {
		return nil
	}
	return &ColumnAttr{Table: ref.Table, Column: ref.Column}
}

func (r *joinStrictColumnAttrsOmniRule) checkColumnAttrs(left, right *ColumnAttr) {
	if left == nil || right == nil {
		return
	}

	leftTable := r.findTable(left.Table)
	rightTable := r.findTable(right.Table)
	if leftTable == nil || rightTable == nil {
		return
	}
	leftColumn := leftTable.GetColumn(left.Column)
	rightColumn := rightTable.GetColumn(right.Column)
	if leftColumn == nil || rightColumn == nil {
		return
	}
	if leftColumn.GetProto().Type != rightColumn.GetProto().Type || leftColumn.GetProto().CharacterSet != rightColumn.GetProto().CharacterSet || leftColumn.GetProto().Collation != rightColumn.GetProto().Collation {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.StatementJoinColumnAttrsNotMatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("%s.%s and %s.%s column fields do not match", left.Table, left.Column, right.Table, right.Column),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine),
		})
	}
}

func (r *joinStrictColumnAttrsOmniRule) findTable(tableName string) *model.TableMetadata {
	if r.dbMetadata == nil {
		return nil
	}
	return r.dbMetadata.GetSchemaMetadata("").GetTable(tableName)
}

// nolint:unused
func extractJoinInfoFromText(text string) *ColumnAttr {
	elements := strings.Split(text, ".")
	if len(elements) != 2 {
		return nil
	}
	return &ColumnAttr{
		Table:  elements[0],
		Column: elements[1],
	}
}
