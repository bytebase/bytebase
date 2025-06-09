package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
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
	checker := &columnNoNullChecker{
		level:     level,
		title:     string(checkCtx.Rule.Type),
		columnSet: make(map[string]columnName),
		catalog:   checkCtx.Catalog,
	}

	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.generateAdvice(), nil
}

type columnName struct {
	tableName  string
	columnName string
	line       int
}

func (c columnName) name() string {
	return fmt.Sprintf("%s.%s", c.tableName, c.columnName)
}

type columnNoNullChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	columnSet  map[string]columnName
	catalog    *catalog.Finder
}

func (checker *columnNoNullChecker) generateAdvice() []*storepb.Advice {
	var columnList []columnName
	for _, column := range checker.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(a, b columnName) int {
		if a.line != b.line {
			if a.line < b.line {
				return -1
			}
			return 1
		}
		if a.columnName < b.columnName {
			return -1
		}
		if a.columnName > b.columnName {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		col := checker.catalog.Final.FindColumn(&catalog.ColumnFind{
			TableName:  column.tableName,
			ColumnName: column.columnName,
		})
		if col != nil && col.Nullable() {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.ColumnCannotNull.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
	}

	return checker.adviceList
}

func (checker *columnNoNullChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
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
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement == nil {
			continue
		}
		if tableElement.ColumnDefinition() == nil {
			continue
		}

		_, _, column := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		col := columnName{
			tableName:  tableName,
			columnName: column,
			line:       checker.baseLine + tableElement.GetStart().GetLine(),
		}
		if _, exists := checker.columnSet[col.name()]; !exists {
			checker.columnSet[col.name()] = col
		}
	}
}

func (checker *columnNoNullChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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

		var columns []string
		switch {
		// add column
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				column := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				columns = append(columns, column)
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil {
						continue
					}
					_, _, column := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					columns = append(columns, column)
				}
			}
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			// only care new column name.
			column := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			columns = append(columns, column)
		}

		for _, column := range columns {
			col := columnName{
				tableName:  tableName,
				columnName: column,
				line:       checker.baseLine + item.GetStart().GetLine(),
			}
			if _, exists := checker.columnSet[col.name()]; !exists {
				checker.columnSet[col.name()] = col
			}
		}
	}
}
