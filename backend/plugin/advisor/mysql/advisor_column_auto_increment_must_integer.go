package mysql

// Framework code is generated by the generator.

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustIntegerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLAutoIncrementColumnMustInteger, &ColumnAutoIncrementMustIntegerAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLAutoIncrementColumnMustInteger, &ColumnAutoIncrementMustIntegerAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLAutoIncrementColumnMustInteger, &ColumnAutoIncrementMustIntegerAdvisor{})
}

// ColumnAutoIncrementMustIntegerAdvisor is the advisor checking for auto-increment column type.
type ColumnAutoIncrementMustIntegerAdvisor struct {
}

// Check checks for auto-increment column type.
func (*ColumnAutoIncrementMustIntegerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnAutoIncrementMustIntegerChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type columnAutoIncrementMustIntegerChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (checker *columnAutoIncrementMustIntegerChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		checker.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
	}
}

func (checker *columnAutoIncrementMustIntegerChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
	if tableName == "" {
		return
	}
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
				checker.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					checker.checkFieldDefinition(tableName, columnName, tableElement.ColumnDefinition().FieldDefinition())
				}
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			// oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if item.FieldDefinition().DataType() == nil {
				continue
			}
			columnName = mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			checker.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			if item.FieldDefinition().DataType() == nil {
				continue
			}
			columnName = mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			checker.checkFieldDefinition(tableName, columnName, item.FieldDefinition())
		}
	}
}

func (checker *columnAutoIncrementMustIntegerChecker) checkFieldDefinition(tableName, columnName string, ctx mysql.IFieldDefinitionContext) {
	if !checker.isAutoIncrementColumnIsInteger(ctx) {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.AutoIncrementColumnNotInteger.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", tableName, columnName),
			StartPosition: common.ConvertANTLRLineToPosition(checker.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (checker *columnAutoIncrementMustIntegerChecker) isAutoIncrementColumnIsInteger(ctx mysql.IFieldDefinitionContext) bool {
	if checker.isAutoIncrementColumn(ctx) && !checker.isIntegerType(ctx.DataType()) {
		return false
	}
	return true
}

func (*columnAutoIncrementMustIntegerChecker) isAutoIncrementColumn(ctx mysql.IFieldDefinitionContext) bool {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.AUTO_INCREMENT_SYMBOL() != nil {
			return true
		}
	}
	return false
}

func (*columnAutoIncrementMustIntegerChecker) isIntegerType(ctx mysql.IDataTypeContext) bool {
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserINT_SYMBOL, mysql.MySQLParserTINYINT_SYMBOL, mysql.MySQLParserSMALLINT_SYMBOL, mysql.MySQLParserMEDIUMINT_SYMBOL, mysql.MySQLParserBIGINT_SYMBOL:
		return true
	default:
		return false
	}
}
