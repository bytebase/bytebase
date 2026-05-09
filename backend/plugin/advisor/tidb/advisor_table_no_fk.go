package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
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
	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (c *tableNoFKChecker) checkStmt(ostmt OmniStmt) {
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		for _, constraint := range n.Constraints {
			if constraint == nil || constraint.Type != ast.ConstrForeignKey {
				continue
			}
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.TableHasFK.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", n.Table.Name),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.AbsoluteLine(constraint.Loc.Start)),
			})
		}
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return
		}
		stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			if cmd.Type != ast.ATAddConstraint || cmd.Constraint == nil {
				continue
			}
			if cmd.Constraint.Type != ast.ConstrForeignKey {
				continue
			}
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.TableHasFK.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", n.Table.Name),
				StartPosition: common.ConvertANTLRLineToPosition(stmtLine),
			})
		}
	default:
	}
}
