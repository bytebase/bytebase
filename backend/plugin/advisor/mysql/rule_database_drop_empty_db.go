package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleDropEmptyDatabase, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleDropEmptyDatabase, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleDropEmptyDatabase, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewDatabaseDropEmptyDBRule(level, string(checkCtx.Rule.Type), checkCtx.OriginCatalog)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// DatabaseDropEmptyDBRule checks for drop database only if empty.
type DatabaseDropEmptyDBRule struct {
	BaseRule
	originCatalog *catalog.DatabaseState
}

// NewDatabaseDropEmptyDBRule creates a new DatabaseDropEmptyDBRule.
func NewDatabaseDropEmptyDBRule(level storepb.Advice_Status, title string, originCatalog *catalog.DatabaseState) *DatabaseDropEmptyDBRule {
	return &DatabaseDropEmptyDBRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		originCatalog: originCatalog,
	}
}

// Name returns the rule name.
func (*DatabaseDropEmptyDBRule) Name() string {
	return "DatabaseDropEmptyDBRule"
}

// OnEnter is called when entering a parse tree node.
func (r *DatabaseDropEmptyDBRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeDropDatabase {
		r.checkDropDatabase(ctx.(*mysql.DropDatabaseContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*DatabaseDropEmptyDBRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *DatabaseDropEmptyDBRule) checkDropDatabase(ctx *mysql.DropDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaRef() == nil {
		return
	}

	dbName := mysqlparser.NormalizeMySQLSchemaRef(ctx.SchemaRef())
	if r.originCatalog.DatabaseName() != dbName {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", dbName, r.originCatalog.DatabaseName()),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	} else if !r.originCatalog.HasNoTable() {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DatabaseNotEmpty.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Database `%s` is not allowed to drop if not empty", dbName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
