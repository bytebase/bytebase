package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewInsertMustSpecifyColumnRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// InsertMustSpecifyColumnRule checks for to enforce column specified.
type InsertMustSpecifyColumnRule struct {
	BaseRule
	hasSelect bool
	text      string
}

// NewInsertMustSpecifyColumnRule creates a new InsertMustSpecifyColumnRule.
func NewInsertMustSpecifyColumnRule(level storepb.Advice_Status, title string) *InsertMustSpecifyColumnRule {
	return &InsertMustSpecifyColumnRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*InsertMustSpecifyColumnRule) Name() string {
	return "InsertMustSpecifyColumnRule"
}

// OnEnter is called when entering a parse tree node.
func (r *InsertMustSpecifyColumnRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeInsertStatement:
		r.checkInsertStatement(ctx.(*mysql.InsertStatementContext))
	case NodeTypeSelectItemList:
		r.checkSelectItemList(ctx.(*mysql.SelectItemListContext))
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*InsertMustSpecifyColumnRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *InsertMustSpecifyColumnRule) checkInsertStatement(ctx *mysql.InsertStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.InsertQueryExpression() != nil {
		r.hasSelect = true
	}

	if ctx.InsertFromConstructor() == nil {
		return
	}

	if ctx.InsertFromConstructor() != nil && ctx.InsertFromConstructor().Fields() != nil && len(ctx.InsertFromConstructor().Fields().AllInsertIdentifier()) > 0 {
		// has columns.
		return
	}
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.InsertNotSpecifyColumn.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", r.text),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
	})
}

func (r *InsertMustSpecifyColumnRule) checkSelectItemList(ctx *mysql.SelectItemListContext) {
	if r.hasSelect && ctx.MULT_OPERATOR() != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.InsertNotSpecifyColumn.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
