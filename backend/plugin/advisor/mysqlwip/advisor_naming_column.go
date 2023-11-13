package mysqlwip

import (
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var _ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLNamingColumnConvention, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingColumnConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
		tables:    make(tableState),
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type namingColumnConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
	maxLength  int
	tables     tableState
}

// EnterCreateTable is called when production createTable is entered.
func (checker *namingColumnConventionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
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
		if tableElement.ColumnDefinition().ColumnName() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		checker.handleColumn(tableName, columnName, tableElement.GetStart().GetLine())
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *namingColumnConventionChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
				checker.handleColumn(tableName, columnName, item.GetStart().GetLine())
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					checker.handleColumn(tableName, columnName, tableElement.GetStart().GetLine())
				}
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			// only focus on new column-name.
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			checker.handleColumn(tableName, columnName, item.GetStart().GetLine())
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			// only focus on new column-name.
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			checker.handleColumn(tableName, columnName, item.GetStart().GetLine())
		default:
			continue
		}
	}
}

func (checker *namingColumnConventionChecker) handleColumn(tableName string, columnName string, lineNumber int) {
	// we need to accumulate line number for each statement and elements of statements.
	lineNumber += checker.baseLine
	if !checker.format.MatchString(columnName) {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NamingColumnConventionMismatch,
			Title:   checker.title,
			Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, columnName, checker.format),
			Line:    lineNumber,
		})
	}
	if checker.maxLength > 0 && len(columnName) > checker.maxLength {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NamingColumnConventionMismatch,
			Title:   checker.title,
			Content: fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, columnName, checker.maxLength),
			Line:    lineNumber,
		})
	}
}
