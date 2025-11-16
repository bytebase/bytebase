package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnRequireDefault, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnRequireDefault, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnRequireDefault, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnRequireDefaultRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnRequireDefaultRule checks for column default requirement.
type ColumnRequireDefaultRule struct {
	BaseRule
}

// NewColumnRequireDefaultRule creates a new ColumnRequireDefaultRule.
func NewColumnRequireDefaultRule(level storepb.Advice_Status, title string) *ColumnRequireDefaultRule {
	return &ColumnRequireDefaultRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnRequireDefaultRule) Name() string {
	return "ColumnRequireDefaultRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnRequireDefaultRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnRequireDefaultRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *ColumnRequireDefaultRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	pkColumns := r.getPKColumns(ctx)
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		if pkColumns[columnName] {
			continue
		}
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		r.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
	}
}

func (r *ColumnRequireDefaultRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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

		var columnName string
		switch {
		// add column
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
		// modify column
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			columnName = mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		default:
		}
	}
}

func (r *ColumnRequireDefaultRule) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	if !r.hasDefault(ctx) && r.columnNeedDefault(ctx) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NoDefault.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column `%s`.`%s` doesn't have DEFAULT.", tableName, columnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (*ColumnRequireDefaultRule) hasDefault(ctx mysql.IFieldDefinitionContext) bool {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.DEFAULT_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (*ColumnRequireDefaultRule) getPKColumns(ctx *mysql.CreateTableContext) map[string]bool {
	pkColumn := make(map[string]bool)
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.TableConstraintDef() == nil {
			continue
		}

		if tableElement.TableConstraintDef().GetType_().GetTokenType() != mysql.MySQLParserPRIMARY_SYMBOL {
			continue
		}
		if tableElement.TableConstraintDef().KeyListVariants() == nil {
			continue
		}
		columnList := mysqlparser.NormalizeKeyListVariants(tableElement.TableConstraintDef().KeyListVariants())
		for _, column := range columnList {
			pkColumn[column] = true
		}
	}

	// Also check for inline primary key definitions
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		for _, attr := range tableElement.ColumnDefinition().FieldDefinition().AllColumnAttribute() {
			if attr.PRIMARY_SYMBOL() != nil {
				pkColumn[columnName] = true
				break
			}
		}
	}

	return pkColumn
}

func (*ColumnRequireDefaultRule) columnNeedDefault(ctx mysql.IFieldDefinitionContext) bool {
	if ctx.GENERATED_SYMBOL() != nil {
		return false
	}
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.AUTO_INCREMENT_SYMBOL() != nil || attr.PRIMARY_SYMBOL() != nil {
			return false
		}
	}

	if ctx.DataType() == nil {
		return false
	}

	switch ctx.DataType().GetType_().GetTokenType() {
	case mysql.MySQLParserBLOB_SYMBOL,
		mysql.MySQLParserTINYBLOB_SYMBOL,
		mysql.MySQLParserMEDIUMBLOB_SYMBOL,
		mysql.MySQLParserLONGBLOB_SYMBOL,
		mysql.MySQLParserJSON_SYMBOL,
		mysql.MySQLParserTINYTEXT_SYMBOL,
		mysql.MySQLParserTEXT_SYMBOL,
		mysql.MySQLParserMEDIUMTEXT_SYMBOL,
		mysql.MySQLParserLONGTEXT_SYMBOL,
		mysql.MySQLParserLONG_SYMBOL,
		mysql.MySQLParserSERIAL_SYMBOL,
		mysql.MySQLParserGEOMETRY_SYMBOL,
		mysql.MySQLParserGEOMETRYCOLLECTION_SYMBOL,
		mysql.MySQLParserPOINT_SYMBOL,
		mysql.MySQLParserMULTIPOINT_SYMBOL,
		mysql.MySQLParserLINESTRING_SYMBOL,
		mysql.MySQLParserMULTILINESTRING_SYMBOL,
		mysql.MySQLParserPOLYGON_SYMBOL,
		mysql.MySQLParserMULTIPOLYGON_SYMBOL:
		return false
	default:
	}
	return true
}
