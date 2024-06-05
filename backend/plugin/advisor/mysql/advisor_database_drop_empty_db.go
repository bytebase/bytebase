package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &allowDropEmptyDBChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		catalog: ctx.Catalog,
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  storepb.Advice_SUCCESS,
			Code:    advisor.Ok.Int32(),
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type allowDropEmptyDBChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	catalog    *catalog.Finder
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (checker *allowDropEmptyDBChecker) EnterDropDatabase(ctx *mysql.DropDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaRef() == nil {
		return
	}

	dbName := mysqlparser.NormalizeMySQLSchemaRef(ctx.SchemaRef())
	if checker.catalog.Origin.DatabaseName() != dbName {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    advisor.NotCurrentDatabase.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", dbName, checker.catalog.Origin.DatabaseName()),
			StartPosition: &storepb.Position{
				Line: int32(checker.baseLine + ctx.GetStart().GetLine()),
			},
		})
	} else if !checker.catalog.Origin.HasNoTable() {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    advisor.DatabaseNotEmpty.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("Database `%s` is not allowed to drop if not empty", dbName),
			StartPosition: &storepb.Position{
				Line: int32(checker.baseLine + ctx.GetStart().GetLine()),
			},
		})
	}
}
