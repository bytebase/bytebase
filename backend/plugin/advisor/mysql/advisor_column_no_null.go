package mysql

import (
	"fmt"
	"sort"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLColumnNoNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnNoNullChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		columnSet: make(map[string]columnName),
		catalog:   ctx.Catalog,
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
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	columnSet  map[string]columnName
	catalog    *catalog.Finder
}

func (checker *columnNoNullChecker) generateAdvice() []advisor.Advice {
	var columnList []columnName
	for _, column := range checker.columnSet {
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		if columnList[i].line != columnList[j].line {
			return columnList[i].line < columnList[j].line
		}
		return columnList[i].columnName < columnList[j].columnName
	})

	for _, column := range columnList {
		col := checker.catalog.Final.FindColumn(&catalog.ColumnFind{
			TableName:  column.tableName,
			ColumnName: column.columnName,
		})
		if col != nil && col.Nullable() {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.ColumnCannotNull,
				Title:   checker.title,
				Content: fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				Line:    column.line,
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList
}

func (checker *columnNoNullChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
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
