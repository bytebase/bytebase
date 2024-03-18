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
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for max varchar length.
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
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
	checker := &columnMaximumVarcharLengthChecker{
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

type columnMaximumVarcharLengthChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	baseLine   int
	maximum    int
}

func (checker *columnMaximumVarcharLengthChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
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
		length := getVarcharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
		if checker.maximum > 0 && length > checker.maximum {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.VarcharLengthExceedsLimit,
				Title:   checker.title,
				Content: fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", tableName, columnName, checker.maximum),
				Line:    checker.baseLine + tableElement.GetStart().GetLine(),
			})
		}
	}
}

func (checker *columnMaximumVarcharLengthChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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

		var columnList []string
		varcharLengthMap := make(map[string]int)
		switch {
		// add column.
		case item.ADD_SYMBOL() != nil:
			switch {
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				if item.FieldDefinition().DataType() == nil {
					continue
				}

				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				length := getVarcharLength(item.FieldDefinition().DataType())
				varcharLengthMap[columnName] = length
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
					length := getVarcharLength(tableElement.ColumnDefinition().FieldDefinition().DataType())
					varcharLengthMap[columnName] = length
					columnList = append(columnList, columnName)
				}
			}
		// change column.
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil && item.FieldDefinition() != nil:
			columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			length := getVarcharLength(item.FieldDefinition().DataType())
			varcharLengthMap[columnName] = length
			columnList = append(columnList, columnName)
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.FieldDefinition() != nil:
			length := getVarcharLength(item.FieldDefinition().DataType())
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			varcharLengthMap[columnName] = length
			columnList = append(columnList, columnName)
		default:
			continue
		}
		for _, columnName := range columnList {
			if length, ok := varcharLengthMap[columnName]; ok && checker.maximum > 0 && length > checker.maximum {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.VarcharLengthExceedsLimit,
					Title:   checker.title,
					Content: fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", tableName, columnName, checker.maximum),
					Line:    checker.baseLine + ctx.GetStart().GetLine(),
				})
			}
		}
	}
}

func getVarcharLength(ctx mysql.IDataTypeContext) int {
	if ctx.GetType_() == nil {
		return 0
	}

	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserVARCHAR_SYMBOL:
		if ctx.FieldLength() == nil || ctx.FieldLength().Real_ulonglong_number() == nil {
			return 1
		}
		lengthStr := ctx.FieldLength().Real_ulonglong_number().GetText()
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return 0
		}
		return length
	default:
		return 0
	}
}
