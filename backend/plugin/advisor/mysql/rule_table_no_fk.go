package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableNoFK, &TableNoFKAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleTableNoFK, &TableNoFKAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleTableNoFK, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableNoFKRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableNoFKRule checks table disallow foreign key.
type TableNoFKRule struct {
	BaseRule
}

// NewTableNoFKRule creates a new TableNoFKRule.
func NewTableNoFKRule(level storepb.Advice_Status, title string) *TableNoFKRule {
	return &TableNoFKRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*TableNoFKRule) Name() string {
	return "TableNoFKRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableNoFKRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
		// No action required for other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableNoFKRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableNoFKRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil || ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.TableConstraintDef() == nil {
			continue
		}
		r.handleTableConstraintDef(tableName, tableElement.TableConstraintDef())
	}
}

func (r *TableNoFKRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())

	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		switch {
		// ADD CONSTRANIT
		case option.ADD_SYMBOL() != nil && option.TableConstraintDef() != nil:
			r.handleTableConstraintDef(tableName, option.TableConstraintDef())
		default:
			continue
		}
	}
}

func (r *TableNoFKRule) handleTableConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "FOREIGN":
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		default:
		}
	}
}
