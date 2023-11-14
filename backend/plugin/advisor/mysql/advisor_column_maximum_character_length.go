package mysql

import (
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for max character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &columnMaximumCharacterLengthChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		maximum: payload.Number,
	}

	for _, stmt := range stmtList {
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

type columnMaximumCharacterLengthChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	baseLine   int
	maximum    int
}

func (checker *columnMaximumCharacterLengthChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		if tableElement.ColumnDefinition() == nil {
			continue
		}
		if tableElement.ColumnDefinition().FieldDefinition() == nil {
			continue
		}
		if tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
			continue
		}
		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		charLength := checker.getCharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
		if checker.maximum > 0 && charLength > checker.maximum {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.CharLengthExceedsLimit,
				Title:   checker.title,
				Content: fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", tableName, columnName, checker.maximum),
				Line:    checker.baseLine + tableElement.GetStart().GetLine(),
			})
		}
	}
}

func (checker *columnMaximumCharacterLengthChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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

		var columnList []string
		charLengthMap := make(map[string]int)
		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				if item.FieldDefinition().DataType() == nil {
					continue
				}

				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				charLength := checker.getCharLength(item.FieldDefinition().DataType())
				charLengthMap[columnName] = charLength
				columnList = append(columnList, columnName)
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil {
						continue
					}
					if tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					if tableElement.ColumnDefinition().FieldDefinition().DataType() == nil {
						continue
					}

					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					charLength := checker.getCharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
					charLengthMap[columnName] = charLength
					columnList = append(columnList, columnName)
				}
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			// oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			charLength := checker.getCharLength(item.FieldDefinition().DataType())
			charLengthMap[columnName] = charLength
			columnList = append(columnList, columnName)
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			charLength := checker.getCharLength(item.FieldDefinition().DataType())
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			charLengthMap[columnName] = charLength
			columnList = append(columnList, columnName)
		default:
			continue
		}
		for _, columnName := range columnList {
			if charLength, ok := charLengthMap[columnName]; ok && checker.maximum > 0 && charLength > checker.maximum {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.CharLengthExceedsLimit,
					Title:   checker.title,
					Content: fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", tableName, columnName, checker.maximum),
					Line:    checker.baseLine + ctx.GetStart().GetLine(),
				})
			}
		}
	}
}

func (*columnMaximumCharacterLengthChecker) getCharLength(ctx mysql.IDataTypeContext) int {
	if ctx.GetType_() == nil {
		return 0
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserCHAR_SYMBOL:
		charLengthStr := ctx.FieldLength().Real_ulonglong_number().GetText()
		charLengthInt, err := strconv.Atoi(charLengthStr)
		if err != nil {
			return 0
		}
		return charLengthInt
	default:
		return 0
	}
}
