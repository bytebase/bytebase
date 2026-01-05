package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &statementDisallowCommitRule{
			BaseRule: BaseRule{level: level, title: checkCtx.Rule.Type.String()},
			tokens:   antlrAST.Tokens,
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type statementDisallowCommitRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
}

// Name returns the rule name for logging/debugging.
func (*statementDisallowCommitRule) Name() string {
	return "statement_disallow_commit"
}

// OnEnter is called when entering a parse tree node.
func (r *statementDisallowCommitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType != "Transactionstmt" {
		return nil
	}

	transactionCtx, ok := ctx.(*parser.TransactionstmtContext)
	if !ok {
		return nil
	}

	if !isTopLevel(transactionCtx.GetParent()) {
		return nil
	}

	// Check if this is a COMMIT statement
	if transactionCtx.COMMIT() == nil {
		return nil
	}

	stmtText := getTextFromTokens(r.tokens, transactionCtx)
	if stmtText == "" {
		stmtText = "<unknown statement>"
	}
	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.StatementDisallowCommit.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(transactionCtx.GetStart().GetLine()),
			Column: 0,
		},
	})

	return nil
}

// OnExit is called when exiting a parse tree node.
func (*statementDisallowCommitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}
