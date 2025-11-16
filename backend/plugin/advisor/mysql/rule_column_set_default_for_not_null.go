package mysql

import (
	"context"
	"fmt"

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
	_ advisor.Advisor = (*ColumnSetDefaultForNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnSetDefaultForNotNull, &ColumnSetDefaultForNotNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleColumnSetDefaultForNotNull, &ColumnSetDefaultForNotNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnSetDefaultForNotNull, &ColumnSetDefaultForNotNullAdvisor{})
}

// ColumnSetDefaultForNotNullAdvisor is the advisor checking for set default value for not null column.
type ColumnSetDefaultForNotNullAdvisor struct {
}

// Check checks for set default value for not null column.
func (*ColumnSetDefaultForNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnSetDefaultForNotNullRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnSetDefaultForNotNullRule checks for set default value for not null column.
type ColumnSetDefaultForNotNullRule struct {
	BaseRule
}

// NewColumnSetDefaultForNotNullRule creates a new ColumnSetDefaultForNotNullRule.
func NewColumnSetDefaultForNotNullRule(level storepb.Advice_Status, title string) *ColumnSetDefaultForNotNullRule {
	return &ColumnSetDefaultForNotNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnSetDefaultForNotNullRule) Name() string {
	return "ColumnSetDefaultForNotNullRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnSetDefaultForNotNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
func (*ColumnSetDefaultForNotNullRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (*ColumnSetDefaultForNotNullRule) getPKColumns(ctx *mysql.CreateTableContext) map[string]bool {
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
	return pkColumn
}

func (r *ColumnSetDefaultForNotNullRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	pkColumns := r.getPKColumns(ctx)

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		field := tableElement.ColumnDefinition().FieldDefinition()
		if field == nil {
			continue
		}

		if pkColumns[columnName] || r.isPrimaryKey(field) {
			continue
		}
		if !r.canNull(field) && !r.hasDefault(field) && r.columnNeedDefault(field) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.NotNullColumnWithNoDefault.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", tableName, columnName),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + tableElement.ColumnDefinition().GetStart().GetLine()),
			})
		}
	}
}

func (r *ColumnSetDefaultForNotNullRule) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	if !r.canNull(ctx) && !r.hasDefault(ctx) && r.columnNeedDefault(ctx) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NotNullColumnWithNoDefault.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", tableName, columnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (*ColumnSetDefaultForNotNullRule) canNull(ctx mysql.IFieldDefinitionContext) bool {
	for _, attribute := range ctx.AllColumnAttribute() {
		switch {
		case attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil:
			return false
		case attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() == nil:
			return true
		default:
		}
	}
	return true
}

func (*ColumnSetDefaultForNotNullRule) isPrimaryKey(ctx mysql.IFieldDefinitionContext) bool {
	for _, attribute := range ctx.AllColumnAttribute() {
		if attribute.PRIMARY_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (*ColumnSetDefaultForNotNullRule) hasDefault(ctx mysql.IFieldDefinitionContext) bool {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.DEFAULT_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (*ColumnSetDefaultForNotNullRule) columnNeedDefault(ctx mysql.IFieldDefinitionContext) bool {
	if ctx.DataType() == nil {
		return true
	}

	// AUTO_INCREMENT columns don't need DEFAULT.
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.AUTO_INCREMENT_SYMBOL() != nil {
			return false
		}
	}

	// Check data types that don't need defaults
	if ctx.DataType().GetType_() != nil {
		switch ctx.DataType().GetType_().GetTokenType() {
		case mysql.MySQLParserTIMESTAMP_SYMBOL, mysql.MySQLParserDATETIME_SYMBOL:
			return false
		// BLOB and TEXT types don't need defaults
		case mysql.MySQLParserTINYBLOB_SYMBOL, mysql.MySQLParserBLOB_SYMBOL,
			mysql.MySQLParserMEDIUMBLOB_SYMBOL, mysql.MySQLParserLONGBLOB_SYMBOL,
			mysql.MySQLParserTINYTEXT_SYMBOL, mysql.MySQLParserTEXT_SYMBOL,
			mysql.MySQLParserMEDIUMTEXT_SYMBOL, mysql.MySQLParserLONGTEXT_SYMBOL:
			return false
		// JSON type doesn't need defaults
		case mysql.MySQLParserJSON_SYMBOL:
			return false
		// SERIAL type doesn't need defaults
		case mysql.MySQLParserSERIAL_SYMBOL:
			return false
		default:
		}
	}

	// Check for LONG VARBINARY and LONG VARCHAR (these are special compound types)
	dataTypeText := ctx.DataType().GetParser().GetTokenStream().GetTextFromRuleContext(ctx.DataType())
	if dataTypeText == "long varbinary" || dataTypeText == "long varchar" {
		return false
	}

	return true
}

func (r *ColumnSetDefaultForNotNullRule) checkAlterTable(ctx *mysql.AlterTableContext) {
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
			if item.FieldDefinition() == nil {
				continue
			}
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			if item.FieldDefinition() == nil {
				continue
			}
			// only care new column name.
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			r.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		default:
		}
	}
}
