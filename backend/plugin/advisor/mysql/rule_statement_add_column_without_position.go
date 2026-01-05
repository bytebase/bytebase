package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementAddColumnWithoutPositionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION, &StatementAddColumnWithoutPositionAdvisor{})
}

// StatementAddColumnWithoutPositionAdvisor is the advisor checking for checking no position in ADD COLUMN clause.
type StatementAddColumnWithoutPositionAdvisor struct {
}

// Check checks for checking no position in ADD COLUMN clause.
func (*StatementAddColumnWithoutPositionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementAddColumnWithoutPositionRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementAddColumnWithoutPositionRule checks for no position in ADD COLUMN clause.
type StatementAddColumnWithoutPositionRule struct {
	BaseRule
}

// NewStatementAddColumnWithoutPositionRule creates a new StatementAddColumnWithoutPositionRule.
func NewStatementAddColumnWithoutPositionRule(level storepb.Advice_Status, title string) *StatementAddColumnWithoutPositionRule {
	return &StatementAddColumnWithoutPositionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementAddColumnWithoutPositionRule) Name() string {
	return "StatementAddColumnWithoutPositionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementAddColumnWithoutPositionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeAlterTable {
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementAddColumnWithoutPositionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementAddColumnWithoutPositionRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	if ctx.TableRef() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}

	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil || item.ADD_SYMBOL() == nil {
			continue
		}

		var position string

		switch {
		case item.Identifier() != nil && item.FieldDefinition() != nil:
			position = getPosition(item.Place())
		case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
			for _, tableElement := range item.TableElementList().AllTableElement() {
				if tableElement.ColumnDefinition() == nil {
					continue
				}
				if tableElement.ColumnDefinition().FieldDefinition() == nil {
					continue
				}

				position = getPosition(item.Place())
				if len(position) != 0 {
					break
				}
			}
		default:
		}

		if len(position) != 0 {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementAddColumnWithPosition.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("add column with position \"%s\"", position),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}

func getPosition(ctx mysql.IPlaceContext) string {
	if ctx == nil {
		return ""
	}
	place, ok := ctx.(*mysql.PlaceContext)
	if !ok || place == nil {
		return ""
	}

	switch {
	case place.FIRST_SYMBOL() != nil:
		return "FIRST"
	case place.AFTER_SYMBOL() != nil:
		return "AFTER"
	default:
		return ""
	}
}
