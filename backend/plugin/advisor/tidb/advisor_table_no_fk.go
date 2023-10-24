package tidb

import (
	"fmt"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
	_ ast.Visitor     = (*tableNoFKChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLTableNoFK, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableNoFKChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

// Enter implements the ast.Visitor interface.
func (checker *tableNoFKChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			if constraint.Tp == ast.ConstraintForeignKey {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.TableHasFK,
					Title:   checker.title,
					Content: fmt.Sprintf("Foreign key is not allowed in the table `%s`", node.Table.Name),
					Line:    constraint.OriginTextPosition(),
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			if spec.Tp == ast.AlterTableAddConstraint && spec.Constraint.Tp == ast.ConstraintForeignKey {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.TableHasFK,
					Title:   checker.title,
					Content: fmt.Sprintf("Foreign key is not allowed in the table `%s`", node.Table.Name),
					Line:    in.OriginTextPosition(),
				})
			}
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*tableNoFKChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
