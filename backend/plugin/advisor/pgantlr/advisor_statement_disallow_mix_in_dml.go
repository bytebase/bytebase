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

	checker := &statementDisallowMixInDMLChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementDisallowMixInDMLChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterCreatestmt handles CREATE TABLE statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE TABLE")
}

// EnterIndexstmt handles CREATE INDEX statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE INDEX")
}

// EnterAltertablestmt handles ALTER TABLE statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "ALTER TABLE")
}

// EnterDropstmt handles DROP statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "DROP")
}

// EnterCreateschemastmt handles CREATE SCHEMA statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE SCHEMA")
}

// EnterCreateseqstmt handles CREATE SEQUENCE statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE SEQUENCE")
}

// EnterAlterseqstmt handles ALTER SEQUENCE statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "ALTER SEQUENCE")
}

// EnterViewstmt handles CREATE VIEW statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE VIEW")
}

// EnterCreatefunctionstmt handles CREATE FUNCTION statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE FUNCTION")
}

// EnterCreatetrigstmt handles CREATE TRIGGER statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE TRIGGER")
}

// EnterRenamestmt handles RENAME statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "RENAME")
}

// EnterAlterobjectschemastmt handles ALTER ... SET SCHEMA statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterAlterobjectschemastmt(ctx *parser.AlterobjectschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "ALTER SET SCHEMA")
}

// EnterAlterenumstmt handles ALTER TYPE ... ADD VALUE statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterAlterenumstmt(ctx *parser.AlterenumstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "ALTER TYPE")
}

// EnterAltercompositetypestmt handles ALTER TYPE (composite) statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterAltercompositetypestmt(ctx *parser.AltercompositetypestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "ALTER TYPE")
}

// EnterCreateextensionstmt handles CREATE EXTENSION statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE EXTENSION")
}

// EnterCreatedbstmt handles CREATE DATABASE statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE DATABASE")
}

// EnterCreatematviewstmt handles CREATE MATERIALIZED VIEW statements (DDL)
func (c *statementDisallowMixInDMLChecker) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.addDDLAdvice(ctx, "CREATE MATERIALIZED VIEW")
}

func (c *statementDisallowMixInDMLChecker) addDDLAdvice(ctx antlr.ParserRuleContext, _ string) {
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
		Content: fmt.Sprintf("Data change can only run DML, %q is not DML", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
