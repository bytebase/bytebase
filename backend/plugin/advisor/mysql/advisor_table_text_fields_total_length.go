package mysql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableTextFieldsTotalLength, &TableMaximumVarcharLengthAdvisor{})
}

type TableMaximumVarcharLengthAdvisor struct {
}

func (*TableMaximumVarcharLengthAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
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
	checker := &tableFieldsMaximumVarcharLengthChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		catalog: ctx.Catalog,
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

type tableFieldsMaximumVarcharLengthChecker struct {
	*mysql.BaseMySQLParserListener

	catalog    *catalog.Finder
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	baseLine   int
	maximum    int
}

func (checker *tableFieldsMaximumVarcharLengthChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableElementList() == nil || ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}
	tableInfo := checker.catalog.Final.FindTable(&catalog.TableFind{TableName: tableName})
	if tableInfo == nil {
		return
	}
	total := getTotalTextLength(tableInfo)
	if total > int64(checker.maximum) {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.IndexCountExceedsLimit,
			Title:   checker.title,
			Content: fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, checker.maximum),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}

func (checker *tableFieldsMaximumVarcharLengthChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
	tableInfo := checker.catalog.Final.FindTable(&catalog.TableFind{TableName: tableName})
	if tableInfo == nil {
		return
	}
	total := getTotalTextLength(tableInfo)
	if total > int64(checker.maximum) {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.TotalTextLengthExceedsLimit,
			Title:   checker.title,
			Content: fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, checker.maximum),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}

func getTotalTextLength(tableInfo *catalog.TableState) int64 {
	var total int64
	columns := tableInfo.ListColumns()
	for _, column := range columns {
		total += getTextLength(column.Type())
	}
	return total
}

func getTextLength(s string) int64 {
	s = strings.ToLower(s)
	switch s {
	case "char", "binary":
		return 1
	case "tinyblob", "tinytext":
		return 255
	case "blob", "text":
		return 65_535
	case "mediumblob", "mediumtext":
		return 16_777_215
	case "longblob", "longtext":
		return 4_294_967_295
	default:
		re := regexp.MustCompile(`[a-z]+\((\d+)\)`)
		match := re.FindStringSubmatch(s)
		if len(match) >= 2 {
			n, err := strconv.ParseInt(match[1], 10, 64)
			if err == nil {
				return int64(n)
			}
		}
	}
	return 0
}
