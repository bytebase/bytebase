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
)

var (
	_ advisor.Advisor = (*WhereRequirementForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequirementForUpdateDeleteAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequirementForUpdateDeleteAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequirementForUpdateDeleteAdvisor{})
}

// WhereRequirementForUpdateDeleteAdvisor is the advisor checking for the WHERE clause requirement for UPDATE/DELETE statements.
type WhereRequirementForUpdateDeleteAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &whereRequirementForUpdateDeleteOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereRequirementForUpdateDeleteOmniRule struct {
	OmniBaseRule
}

func (*whereRequirementForUpdateDeleteOmniRule) Name() string {
	return "WhereRequirementForUpdateDeleteRule"
}

func (r *whereRequirementForUpdateDeleteOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	switch n := node.(type) {
	case *ast.DeleteStmt:
		if n.Where == nil {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("\"%s\" requires WHERE clause", text),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
			})
		}
	case *ast.UpdateStmt:
		if n.Where == nil {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("\"%s\" requires WHERE clause", text),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
			})
		}
	default:
	}
}
