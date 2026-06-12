// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement for UPDATE and DELETE statement.
type WhereRequireForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &whereRequireForUpdateDeleteChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		checker.checkStmt(node, stmt.Text, stmt.BaseLine())
	}

	return checker.adviceList, nil
}

type whereRequireForUpdateDeleteChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// checkStmt flags UPDATE and DELETE statements without a WHERE clause,
// mirroring the legacy listener that fired on Update_statement and
// Delete_statement contexts. MERGE WHEN ... THEN UPDATE/DELETE actions are
// dedicated merge nodes in both ASTs and are not flagged.
func (c *whereRequireForUpdateDeleteChecker) checkStmt(node omniast.Node, text string, baseLine int) {
	omniast.Inspect(node, func(n omniast.Node) bool {
		switch stmt := n.(type) {
		case *omniast.UpdateStmt:
			if stmt.Where == nil {
				c.addAdvice("UPDATE", text, baseLine, stmt.Loc.Start)
			}
		case *omniast.DeleteStmt:
			if stmt.Where == nil {
				c.addAdvice("DELETE", text, baseLine, stmt.Loc.Start)
			}
		default:
			// Ignore other node types
		}
		return true
	})
}

func (c *whereRequireForUpdateDeleteChecker) addAdvice(statementType, text string, baseLine, offset int) {
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:        c.level,
		Code:          code.StatementNoWhere.Int32(),
		Title:         c.title,
		Content:       fmt.Sprintf("WHERE clause is required for %s statement.", statementType),
		StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, offset)),
	})
}
