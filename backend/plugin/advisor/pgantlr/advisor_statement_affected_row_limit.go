package pgantlr

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
func (*StatementAffectedRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &statementAffectedRowLimitChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		maxRow:                       payload.Number,
		ctx:                          ctx,
		driver:                       checkCtx.Driver,
		usePostgresDatabaseOwner:     checkCtx.UsePostgresDatabaseOwner,
		statementsText:               checkCtx.Statements,
	}

	// Only run EXPLAIN queries if we have a limit and database connection
	if payload.Number > 0 && checker.driver != nil {
		antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)
	}

	return checker.adviceList, nil
}

type statementAffectedRowLimitChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList               []*storepb.Advice
	level                    storepb.Advice_Status
	title                    string
	maxRow                   int
	driver                   *sql.DB
	ctx                      context.Context
	explainCount             int
	setRoles                 []string
	usePostgresDatabaseOwner bool
	statementsText           string
}

// EnterVariablesetstmt handles SET ROLE statements
func (c *statementAffectedRowLimitChecker) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is SET ROLE
	if ctx.SET() != nil && ctx.Set_rest() != nil && ctx.Set_rest().Set_rest_more() != nil {
		setRestMore := ctx.Set_rest().Set_rest_more()
		if setRestMore.ROLE() != nil {
			// Store the SET ROLE statement text
			stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
			c.setRoles = append(c.setRoles, stmtText)
		}
	}
}

// EnterUpdatestmt handles UPDATE statements
func (c *statementAffectedRowLimitChecker) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.checkAffectedRows(ctx)
}

// EnterDeletestmt handles DELETE statements
func (c *statementAffectedRowLimitChecker) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	c.checkAffectedRows(ctx)
}

func (c *statementAffectedRowLimitChecker) checkAffectedRows(ctx antlr.ParserRuleContext) {
	// Check if we've hit the maximum number of EXPLAIN queries
	if c.explainCount >= common.MaximumLintExplainSize {
		return
	}

	c.explainCount++

	// Get the statement text
	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	normalizedStmt := advisor.NormalizeStatement(stmtText)

	// Run EXPLAIN to get estimated row count
	res, err := advisor.Query(c.ctx, advisor.QueryContext{
		UsePostgresDatabaseOwner: c.usePostgresDatabaseOwner,
		PreExecutions:            c.setRoles,
	}, c.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", stmtText))

	if err != nil {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.InsertTooManyRows.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", normalizedStmt, err.Error()),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
		return
	}

	rowCount, err := getAffectedRows(res)
	if err != nil {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.Internal.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("failed to get row count for \"%s\": %s", normalizedStmt, err.Error()),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
		return
	}

	if rowCount > int64(c.maxRow) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.StatementAffectedRowExceedsLimit.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("The statement \"%s\" affected %d rows (estimated). The count exceeds %d.", normalizedStmt, rowCount, c.maxRow),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
