package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableNoFKOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableNoFKOmniRule struct {
	OmniBaseRule
}

func (*tableNoFKOmniRule) Name() string {
	return "TableNoFKRule"
}

func (r *tableNoFKOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *tableNoFKOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, constraint := range n.Constraints {
		if constraint.Type == ast.ConstrForeignKey {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(constraint.Loc))),
			})
		}
	}
}

func (r *tableNoFKOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		if cmd.Type == ast.ATAddConstraint && cmd.Constraint != nil && cmd.Constraint.Type == ast.ConstrForeignKey {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(cmd.Constraint.Loc))),
			})
		}
	}
}
