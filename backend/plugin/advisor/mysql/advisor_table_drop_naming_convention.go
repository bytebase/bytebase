package mysql

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

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking the MySQLTableDropNamingConvention rule.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for drop table naming convention.
func (*TableDropNamingConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingDropTableConventionChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		format: format,
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

type namingDropTableConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
}

// EnterDropTable is called when production dropTable is entered.
func (checker *namingDropTableConventionChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		if !checker.format.MatchString(tableName) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.TableDropNamingConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, checker.format),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		}
	}
}
