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
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnNoNullRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnNoNullRule checks for column no NULL value.
type ColumnNoNullRule struct {
	BaseRule
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnNoNullRule) Name() string {
	return "ColumnNoNullRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnNoNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnNoNullRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *ColumnNoNullRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		fieldDef := tableElement.ColumnDefinition().FieldDefinition()
		if fieldDef == nil {
			continue
		}

		// Check nullability directly from AST
		if isNullable(fieldDef) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          advisor.ColumnCannotNull.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", tableName, columnName),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + tableElement.ColumnDefinition().GetStart().GetLine()),
			})
		}
	}
}

// isNullable checks if a column is nullable based on its field definition.
// Default is nullable unless explicitly marked as NOT NULL or PRIMARY KEY.
func isNullable(fieldDef mysql.IFieldDefinitionContext) bool {
	for _, attribute := range fieldDef.AllColumnAttribute() {
		if attribute == nil {
			continue
		}
		// NOT NULL
		if attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil {
			return false
		}
		// PRIMARY KEY implies NOT NULL
		if attribute.GetValue() != nil && attribute.GetValue().GetTokenType() == mysql.MySQLParserKEY_SYMBOL {
			return false
		}
	}
	return true
}

func (r *ColumnNoNullRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				if isNullable(item.FieldDefinition()) {
					r.AddAdvice(&storepb.Advice{
						Status:        r.level,
						Code:          advisor.ColumnCannotNull.Int32(),
						Title:         r.title,
						Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", tableName, columnName),
						StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + item.GetStart().GetLine()),
					})
				}
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					if isNullable(tableElement.ColumnDefinition().FieldDefinition()) {
						r.AddAdvice(&storepb.Advice{
							Status:        r.level,
							Code:          advisor.ColumnCannotNull.Int32(),
							Title:         r.title,
							Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", tableName, columnName),
							StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + tableElement.GetStart().GetLine()),
						})
					}
				}
			default:
			}
		// change column or modify column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if isNullable(item.FieldDefinition()) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          advisor.ColumnCannotNull.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", tableName, columnName),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + item.GetStart().GetLine()),
				})
			}
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if isNullable(item.FieldDefinition()) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          advisor.ColumnCannotNull.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", tableName, columnName),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + item.GetStart().GetLine()),
				})
			}
		default:
		}
	}
}
