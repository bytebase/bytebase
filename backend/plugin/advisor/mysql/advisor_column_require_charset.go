package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnRequireCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLColumnRequireCharset, &ColumnRequireCharsetAdvisor{})
}

// ColumnRequireCharsetAdvisor is the advisor checking for require charset.
type ColumnRequireCharsetAdvisor struct {
}

func (*ColumnRequireCharsetAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnRequireCharsetChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type columnRequireCharsetChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	baseLine   int
}

func (checker *columnRequireCharsetChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil || ctx.TableElementList() == nil {
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
		columnDefinition := tableElement.ColumnDefinition()
		if columnDefinition.FieldDefinition() == nil || columnDefinition.FieldDefinition().DataType() == nil {
			continue
		}

		_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
		dataType := columnDefinition.FieldDefinition().DataType()
		if isCharsetDataType(dataType) {
			if dataType.CharsetWithOptBinary() == nil {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:  checker.level,
					Code:    advisor.NoCharset.Int32(),
					Title:   checker.title,
					Content: fmt.Sprintf("Column %s does not have a character set specified", columnName),
					StartPosition: &storepb.Position{
						Line: int32(checker.baseLine + columnDefinition.GetStart().GetLine()),
					},
				})
			}
		}
	}
}

func (checker *columnRequireCharsetChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil || ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		// Only check ADD COLUMN for now.
		if alterListItem.ADD_SYMBOL() == nil || alterListItem.COLUMN_SYMBOL() == nil || alterListItem.FieldDefinition() == nil {
			continue
		}

		columnName := mysqlparser.NormalizeMySQLIdentifier(alterListItem.Identifier())
		dataType := alterListItem.FieldDefinition().DataType()
		if isCharsetDataType(dataType) {
			if dataType.CharsetWithOptBinary() == nil {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:  checker.level,
					Code:    advisor.NoCharset.Int32(),
					Title:   checker.title,
					Content: fmt.Sprintf("Column %s does not have a character set specified", columnName),
					StartPosition: &storepb.Position{
						Line: int32(checker.baseLine + alterListItem.GetStart().GetLine()),
					},
				})
			}
		}
	}
}

func isCharsetDataType(dataType mysql.IDataTypeContext) bool {
	return dataType != nil && (dataType.CHAR_SYMBOL() != nil ||
		dataType.VARCHAR_SYMBOL() != nil ||
		dataType.VARYING_SYMBOL() != nil ||
		dataType.TINYTEXT_SYMBOL() != nil ||
		dataType.TEXT_SYMBOL() != nil ||
		dataType.MEDIUMTEXT_SYMBOL() != nil ||
		dataType.LONGTEXT_SYMBOL() != nil)
}
