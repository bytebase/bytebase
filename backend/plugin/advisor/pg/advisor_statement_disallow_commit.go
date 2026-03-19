package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowCommitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
}

// StatementDisallowCommitAdvisor is the advisor checking for disallowing COMMIT statements.
type StatementDisallowCommitAdvisor struct {
}

// Check checks for disallowing COMMIT statements.
func (*StatementDisallowCommitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowCommitRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowCommitRule struct {
	OmniBaseRule
}

func (*statementDisallowCommitRule) Name() string {
	return "statement_disallow_commit"
}

func (r *statementDisallowCommitRule) OnStatement(node ast.Node) {
	txn, ok := node.(*ast.TransactionStmt)
	if !ok {
		return
	}

	if txn.Kind != ast.TRANS_STMT_COMMIT {
		return
	}

	stmtText := r.TrimmedStmtText()
	if stmtText == "" {
		stmtText = "<unknown statement>"
	}
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.StatementDisallowCommit.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", stmtText),
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
