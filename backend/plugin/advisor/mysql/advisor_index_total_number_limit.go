package mysql

// Framework code is generated by the generator.

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number limit.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &indexTotalNumberLimitChecker{
		level:        level,
		title:        string(checkCtx.Rule.Type),
		max:          payload.Number,
		lineForTable: make(map[string]int),
		catalog:      checkCtx.Catalog,
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.generateAdvice(), nil
}

type indexTotalNumberLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	max          int
	lineForTable map[string]int
	catalog      *catalog.Finder
}

func (checker *indexTotalNumberLimitChecker) generateAdvice() []*storepb.Advice {
	type tableName struct {
		name string
		line int
	}
	var tableList []tableName

	for k, v := range checker.lineForTable {
		tableList = append(tableList, tableName{
			name: k,
			line: v,
		})
	}
	slices.SortFunc(tableList, func(i, j tableName) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})

	for _, table := range tableList {
		tableInfo := checker.catalog.Final.FindTable(&catalog.TableFind{TableName: table.name})
		if tableInfo != nil && tableInfo.CountIndex() > checker.max {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.IndexCountExceedsLimit.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %d", table.name, checker.max, tableInfo.CountIndex()),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
	}

	return checker.adviceList
}

// EnterCreateTable is called when production createTable is entered.
func (checker *indexTotalNumberLimitChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	checker.lineForTable[tableName] = checker.baseLine + ctx.GetStart().GetLine()
}

func (checker *indexTotalNumberLimitChecker) EnterCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	checker.lineForTable[tableName] = checker.baseLine + ctx.GetStart().GetLine()
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *indexTotalNumberLimitChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			// add single columns.
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				checker.checkFieldDefinitionContext(tableName, item.FieldDefinition())
			// add multi columns.
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					checker.checkFieldDefinitionContext(tableName, tableElement.ColumnDefinition().FieldDefinition())
				}
				// add constraint.
			case item.TableConstraintDef() != nil:
				checker.checkTableConstraintDef(tableName, item.TableConstraintDef())
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			checker.checkFieldDefinitionContext(tableName, item.FieldDefinition())
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			checker.checkFieldDefinitionContext(tableName, item.FieldDefinition())
		default:
			continue
		}
	}
}

func (checker *indexTotalNumberLimitChecker) checkFieldDefinitionContext(tableName string, ctx mysql.IFieldDefinitionContext) {
	for _, attr := range ctx.AllColumnAttribute() {
		if attr == nil || attr.GetValue() == nil {
			continue
		}
		switch attr.GetValue().GetTokenType() {
		case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL:
			checker.lineForTable[tableName] = checker.baseLine + ctx.GetStart().GetLine()
		}
	}
}

func (checker *indexTotalNumberLimitChecker) checkTableConstraintDef(tableName string, ctx mysql.ITableConstraintDefContext) {
	if ctx.GetType_() == nil {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserPRIMARY_SYMBOL, mysql.MySQLParserUNIQUE_SYMBOL, mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserINDEX_SYMBOL, mysql.MySQLParserFULLTEXT_SYMBOL:
		checker.lineForTable[tableName] = checker.baseLine + ctx.GetStart().GetLine()
	}
}
