package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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

	checker := &statementDisallowMixInDDLChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementDisallowMixInDDLChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterSelectstmt handles SELECT statements (DML)
func (c *statementDisallowMixInDDLChecker) EnterSelectstmt(ctx *parser.SelectstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDMLAdvice(ctx, "SELECT")
}

// EnterInsertstmt handles INSERT statements (DML)
func (c *statementDisallowMixInDDLChecker) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDMLAdvice(ctx, "INSERT")
}

// EnterUpdatestmt handles UPDATE statements (DML)
func (c *statementDisallowMixInDDLChecker) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDMLAdvice(ctx, "UPDATE")
}

// EnterDeletestmt handles DELETE statements (DML)
func (c *statementDisallowMixInDDLChecker) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDMLAdvice(ctx, "DELETE")
}

func (c *statementDisallowMixInDDLChecker) addDMLAdvice(ctx antlr.ParserRuleContext, _ string) {
	// Extract the statement text including semicolon using character positions
	startPos := ctx.GetStart().GetStart()
	stopPos := ctx.GetStop().GetStop()

	// Find the semicolon after this statement
	stmtText := ""
	if stopPos+1 < len(c.statementsText) {
		// Look for semicolon
		endPos := stopPos + 1
		for endPos < len(c.statementsText) && c.statementsText[endPos] != ';' {
			endPos++
		}
		if endPos < len(c.statementsText) {
			stmtText = c.statementsText[startPos : endPos+1]
		} else {
			stmtText = c.statementsText[startPos:stopPos+1] + ";"
		}
	} else {
		stmtText = c.statementsText[startPos:stopPos+1] + ";"
	}

	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.StatementDisallowMixDDLDML.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("Alter schema can only run DDL, %q is not DDL", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
