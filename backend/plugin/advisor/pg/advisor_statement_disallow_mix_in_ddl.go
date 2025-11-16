package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
}

// StatementDisallowMixInDDLAdvisor is the advisor checking for disallow mix DDL and DML.
type StatementDisallowMixInDDLAdvisor struct {
}

// Check checks for disallow mix DDL and DML.
func (*StatementDisallowMixInDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	// Only check when change type is DDL
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
	default:
		return nil, nil
	}

	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowMixInDDLRule{
		BaseRule:       BaseRule{level: level, title: string(checkCtx.Rule.Type)},
		statementsText: checkCtx.Statements,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementDisallowMixInDDLRule struct {
	BaseRule
	statementsText string
}

// Name returns the rule name for logging/debugging
func (*statementDisallowMixInDDLRule) Name() string {
	return "statement_disallow_mix_in_ddl"
}

// OnEnter is called when entering a parse tree node
func (r *statementDisallowMixInDDLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Selectstmt":
		r.handleSelectstmt(ctx)
	case "Insertstmt":
		r.handleInsertstmt(ctx)
	case "Updatestmt":
		r.handleUpdatestmt(ctx)
	case "Deletestmt":
		r.handleDeletestmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node
func (*statementDisallowMixInDDLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleSelectstmt handles SELECT statements (DML)
func (r *statementDisallowMixInDDLRule) handleSelectstmt(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDMLAdvice(ctx)
}

// handleInsertstmt handles INSERT statements (DML)
func (r *statementDisallowMixInDDLRule) handleInsertstmt(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDMLAdvice(ctx)
}

// handleUpdatestmt handles UPDATE statements (DML)
func (r *statementDisallowMixInDDLRule) handleUpdatestmt(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDMLAdvice(ctx)
}

// handleDeletestmt handles DELETE statements (DML)
func (r *statementDisallowMixInDDLRule) handleDeletestmt(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDMLAdvice(ctx)
}

func (r *statementDisallowMixInDDLRule) addDMLAdvice(ctx antlr.ParserRuleContext) {
	// Extract the statement text including semicolon using character positions
	startPos := ctx.GetStart().GetStart()
	stopPos := ctx.GetStop().GetStop()

	// Find the semicolon after this statement
	stmtText := ""
	if stopPos+1 < len(r.statementsText) {
		// Look for semicolon
		endPos := stopPos + 1
		for endPos < len(r.statementsText) && r.statementsText[endPos] != ';' {
			endPos++
		}
		if endPos < len(r.statementsText) {
			stmtText = r.statementsText[startPos : endPos+1]
		} else {
			stmtText = r.statementsText[startPos:stopPos+1] + ";"
		}
	} else {
		stmtText = r.statementsText[startPos:stopPos+1] + ";"
	}

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.StatementDisallowMixDDLDML.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("Alter schema can only run DDL, %q is not DDL", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
