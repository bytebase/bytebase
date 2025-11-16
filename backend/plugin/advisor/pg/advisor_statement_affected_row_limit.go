package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	rule := &statementAffectedRowLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		maxRow:                   payload.Number,
		ctx:                      ctx,
		driver:                   checkCtx.Driver,
		usePostgresDatabaseOwner: checkCtx.UsePostgresDatabaseOwner,
		statementsText:           checkCtx.Statements,
	}

	if payload.Number > 0 && checkCtx.Driver != nil {
		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)
		return checker.GetAdviceList(), nil
	}

	return nil, nil
}

type statementAffectedRowLimitRule struct {
	BaseRule

	maxRow                   int
	driver                   *sql.DB
	ctx                      context.Context
	explainCount             int
	setRoles                 []string
	usePostgresDatabaseOwner bool
	statementsText           string
}

func (*statementAffectedRowLimitRule) Name() string {
	return "statement_affected_row_limit"
}

func (r *statementAffectedRowLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Variablesetstmt":
		if c, ok := ctx.(*parser.VariablesetstmtContext); ok {
			r.handleVariablesetstmt(c)
		}
	case "Updatestmt":
		if c, ok := ctx.(*parser.UpdatestmtContext); ok {
			r.handleUpdatestmt(c)
		}
	case "Deletestmt":
		if c, ok := ctx.(*parser.DeletestmtContext); ok {
			r.handleDeletestmt(c)
		}
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*statementAffectedRowLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementAffectedRowLimitRule) handleVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is SET ROLE
	if ctx.SET() != nil && ctx.Set_rest() != nil && ctx.Set_rest().Set_rest_more() != nil {
		setRestMore := ctx.Set_rest().Set_rest_more()
		if setRestMore.ROLE() != nil {
			// Store the SET ROLE statement text
			stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
			r.setRoles = append(r.setRoles, stmtText)
		}
	}
}

func (r *statementAffectedRowLimitRule) handleUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkAffectedRows(ctx)
}

func (r *statementAffectedRowLimitRule) handleDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkAffectedRows(ctx)
}

func (r *statementAffectedRowLimitRule) checkAffectedRows(ctx antlr.ParserRuleContext) {
	// Check if we've hit the maximum number of EXPLAIN queries
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}

	r.explainCount++

	// Get the statement text
	stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	normalizedStmt := advisor.NormalizeStatement(stmtText)

	// Run EXPLAIN to get estimated row count
	res, err := advisor.Query(r.ctx, advisor.QueryContext{
		UsePostgresDatabaseOwner: r.usePostgresDatabaseOwner,
		PreExecutions:            r.setRoles,
	}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", stmtText))

	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.InsertTooManyRows.Int32(),
			Title:   r.title,
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
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.Internal.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("failed to get row count for \"%s\": %s", normalizedStmt, err.Error()),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
		return
	}

	if rowCount > int64(r.maxRow) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementAffectedRowExceedsLimit.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The statement \"%s\" affected %d rows (estimated). The count exceeds %d.", normalizedStmt, rowCount, r.maxRow),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
