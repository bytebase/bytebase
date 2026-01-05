// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number limit.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 {
		return nil, nil
	}

	rule := NewIndexKeyNumberLimitRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, int(numberPayload.Number))
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList()
}

// IndexKeyNumberLimitRule is the rule implementation for index key number limit.
type IndexKeyNumberLimitRule struct {
	BaseRule

	currentDatabase string
	max             int
}

// NewIndexKeyNumberLimitRule creates a new IndexKeyNumberLimitRule.
func NewIndexKeyNumberLimitRule(level storepb.Advice_Status, title string, currentDatabase string, maxKeys int) *IndexKeyNumberLimitRule {
	return &IndexKeyNumberLimitRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		max:             maxKeys,
	}
}

// Name returns the rule name.
func (*IndexKeyNumberLimitRule) Name() string {
	return "index.key-number-limit"
}

// OnEnter is called when the parser enters a rule context.
func (r *IndexKeyNumberLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Table_index_clause":
		r.handleTableIndexClause(ctx.(*parser.Table_index_clauseContext))
	case "Out_of_line_constraint":
		r.handleOutOfLineConstraint(ctx.(*parser.Out_of_line_constraintContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*IndexKeyNumberLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *IndexKeyNumberLimitRule) handleTableIndexClause(ctx *parser.Table_index_clauseContext) {
	keys := len(ctx.AllIndex_expr_option())
	if keys > r.max {
		r.AddAdvice(
			r.level,
			code.IndexKeyNumberExceedsLimit.Int32(),
			fmt.Sprintf("Index key number should be less than or equal to %d", r.max),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}

func (r *IndexKeyNumberLimitRule) handleOutOfLineConstraint(ctx *parser.Out_of_line_constraintContext) {
	keys := len(ctx.AllColumn_name())
	if keys > r.max {
		r.AddAdvice(
			r.level,
			code.IndexKeyNumberExceedsLimit.Int32(),
			fmt.Sprintf("Index key number should be less than or equal to %d", r.max),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
