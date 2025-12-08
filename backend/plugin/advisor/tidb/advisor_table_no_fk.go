package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
	_ ast.Visitor     = (*tableNoFKChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, err := getTiDBNodes(checkCtx)

	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableNoFKChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// Enter implements the ast.Visitor interface.
func (checker *tableNoFKChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			if constraint.Tp == ast.ConstraintForeignKey {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          code.TableHasFK.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", node.Table.Name),
					StartPosition: common.ConvertANTLRLineToPosition(constraint.OriginTextPosition()),
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			if spec.Tp == ast.AlterTableAddConstraint && spec.Constraint.Tp == ast.ConstraintForeignKey {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          code.TableHasFK.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", node.Table.Name),
					StartPosition: common.ConvertANTLRLineToPosition(in.OriginTextPosition()),
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
