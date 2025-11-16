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
	_ advisor.Advisor = (*ColumnAutoIncrementMustUnsignedAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnAutoIncrementMustUnsigned, &ColumnAutoIncrementMustUnsignedAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnAutoIncrementMustUnsigned, &ColumnAutoIncrementMustUnsignedAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnAutoIncrementMustUnsigned, &ColumnAutoIncrementMustUnsignedAdvisor{})
}

// ColumnAutoIncrementMustUnsignedAdvisor is the advisor checking for unsigned auto-increment column.
type ColumnAutoIncrementMustUnsignedAdvisor struct {
}

// Check checks for unsigned auto-increment column.
func (*ColumnAutoIncrementMustUnsignedAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnAutoIncrementMustUnsignedRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnAutoIncrementMustUnsignedRule checks for unsigned auto-increment column.
type ColumnAutoIncrementMustUnsignedRule struct {
	BaseRule
}

// NewColumnAutoIncrementMustUnsignedRule creates a new ColumnAutoIncrementMustUnsignedRule.
func NewColumnAutoIncrementMustUnsignedRule(level storepb.Advice_Status, title string) *ColumnAutoIncrementMustUnsignedRule {
	return &ColumnAutoIncrementMustUnsignedRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnAutoIncrementMustUnsignedRule) Name() string {
	return "ColumnAutoIncrementMustUnsignedRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnAutoIncrementMustUnsignedRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnAutoIncrementMustUnsignedRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnAutoIncrementMustUnsignedRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
	}
}

func (r *ColumnAutoIncrementMustUnsignedRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}

	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		var columnName string
		switch {
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
				}
			default:
			}
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			if item.FieldDefinition().DataType() == nil {
				continue
			}
			columnName = mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			if item.FieldDefinition().DataType() == nil {
				continue
			}
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		default:
		}
	}
}

func (r *ColumnAutoIncrementMustUnsignedRule) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	if !r.isAutoIncrementColumnIsUnsigned(ctx) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.AutoIncrementColumnSigned.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", tableName, columnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (r *ColumnAutoIncrementMustUnsignedRule) isAutoIncrementColumnIsUnsigned(ctx mysql.IFieldDefinitionContext) bool {
	if r.isAutoIncrementColumn(ctx) && !r.isUnsigned(ctx) {
		return false
	}
	return true
}

func (*ColumnAutoIncrementMustUnsignedRule) isAutoIncrementColumn(ctx mysql.IFieldDefinitionContext) bool {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.AUTO_INCREMENT_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (*ColumnAutoIncrementMustUnsignedRule) isUnsigned(ctx mysql.IFieldDefinitionContext) bool {
	if ctx.DataType() == nil {
		return false
	}

	// Check if UNSIGNED is specified in the data type
	dataTypeText := ctx.DataType().GetParser().GetTokenStream().GetTextFromRuleContext(ctx.DataType())
	upperText := strings.ToUpper(dataTypeText)

	// UNSIGNED is explicitly specified or ZEROFILL (which implies UNSIGNED)
	return strings.Contains(upperText, "UNSIGNED") || strings.Contains(upperText, "ZEROFILL")
}
