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
	_ advisor.Advisor = (*StatementDisallowCommitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
}

// StatementDisallowCommitAdvisor is the advisor checking for disallowing commit.
type StatementDisallowCommitAdvisor struct {
}

// Check checks for disallowing commit.
func (*StatementDisallowCommitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &disallowCommitOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type disallowCommitOmniRule struct {
	OmniBaseRule
}

func (*disallowCommitOmniRule) Name() string {
	return "StatementDisallowCommitRule"
}

func (r *disallowCommitOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CommitStmt)
	if !ok {
		return
	}
	text := strings.TrimSpace(r.StmtText)
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.StatementDisallowCommit.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", text),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
	})
}
