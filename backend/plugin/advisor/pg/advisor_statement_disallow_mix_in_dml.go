package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

// StatementDisallowMixInDMLAdvisor is the advisor checking for disallow mix DDL and DML.
type StatementDisallowMixInDMLAdvisor struct {
}

// Check checks for disallow mix DDL and DML.
func (*StatementDisallowMixInDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	// Only check when change type is DML
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DML:
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

	rule := &statementDisallowMixInDMLRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		statementsText: checkCtx.Statements,
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementDisallowMixInDMLRule struct {
	BaseRule

	statementsText string
}

func (*statementDisallowMixInDMLRule) Name() string {
	return "statement_disallow_mix_in_dml"
}

func (r *statementDisallowMixInDMLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx)
	case "Indexstmt":
		r.handleIndexstmt(ctx)
	case "Altertablestmt":
		r.handleAltertablestmt(ctx)
	case "Dropstmt":
		r.handleDropstmt(ctx)
	case "Createschemastmt":
		r.handleCreateschemastmt(ctx)
	case "Createseqstmt":
		r.handleCreateseqstmt(ctx)
	case "Alterseqstmt":
		r.handleAlterseqstmt(ctx)
	case "Viewstmt":
		r.handleViewstmt(ctx)
	case "Createfunctionstmt":
		r.handleCreatefunctionstmt(ctx)
	case "Createtrigstmt":
		r.handleCreatetrigstmt(ctx)
	case "Renamestmt":
		r.handleRenamestmt(ctx)
	case "Alterobjectschemastmt":
		r.handleAlterobjectschemastmt(ctx)
	case "Alterenumstmt":
		r.handleAlterenumstmt(ctx)
	case "Altercompositetypestmt":
		r.handleAltercompositetypestmt(ctx)
	case "Createextensionstmt":
		r.handleCreateextensionstmt(ctx)
	case "Createdbstmt":
		r.handleCreatedbstmt(ctx)
	case "Creatematviewstmt":
		r.handleCreatematviewstmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*statementDisallowMixInDMLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementDisallowMixInDMLRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE TABLE")
}

func (r *statementDisallowMixInDMLRule) handleIndexstmt(ctx antlr.ParserRuleContext) {
	indexCtx, ok := ctx.(*parser.IndexstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(indexCtx, "CREATE INDEX")
}

func (r *statementDisallowMixInDMLRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(alterCtx, "ALTER TABLE")
}

func (r *statementDisallowMixInDMLRule) handleDropstmt(ctx antlr.ParserRuleContext) {
	dropCtx, ok := ctx.(*parser.DropstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(dropCtx, "DROP")
}

func (r *statementDisallowMixInDMLRule) handleCreateschemastmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreateschemastmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE SCHEMA")
}

func (r *statementDisallowMixInDMLRule) handleCreateseqstmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreateseqstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE SEQUENCE")
}

func (r *statementDisallowMixInDMLRule) handleAlterseqstmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AlterseqstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(alterCtx, "ALTER SEQUENCE")
}

func (r *statementDisallowMixInDMLRule) handleViewstmt(ctx antlr.ParserRuleContext) {
	viewCtx, ok := ctx.(*parser.ViewstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(viewCtx, "CREATE VIEW")
}

func (r *statementDisallowMixInDMLRule) handleCreatefunctionstmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatefunctionstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE FUNCTION")
}

func (r *statementDisallowMixInDMLRule) handleCreatetrigstmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatetrigstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE TRIGGER")
}

func (r *statementDisallowMixInDMLRule) handleRenamestmt(ctx antlr.ParserRuleContext) {
	renameCtx, ok := ctx.(*parser.RenamestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(renameCtx, "RENAME")
}

func (r *statementDisallowMixInDMLRule) handleAlterobjectschemastmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AlterobjectschemastmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(alterCtx, "ALTER SET SCHEMA")
}

func (r *statementDisallowMixInDMLRule) handleAlterenumstmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AlterenumstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(alterCtx, "ALTER TYPE")
}

func (r *statementDisallowMixInDMLRule) handleAltercompositetypestmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AltercompositetypestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(alterCtx, "ALTER TYPE")
}

func (r *statementDisallowMixInDMLRule) handleCreateextensionstmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreateextensionstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE EXTENSION")
}

func (r *statementDisallowMixInDMLRule) handleCreatedbstmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatedbstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE DATABASE")
}

func (r *statementDisallowMixInDMLRule) handleCreatematviewstmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatematviewstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.addDDLAdvice(createCtx, "CREATE MATERIALIZED VIEW")
}

func (r *statementDisallowMixInDMLRule) addDDLAdvice(ctx antlr.ParserRuleContext, _ string) {
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
		Content: fmt.Sprintf("Data change can only run DML, %q is not DML", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
