package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewDatabaseDropEmptyDBRule(level, checkCtx.Rule.Type.String(), checkCtx.OriginalMetadata)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// DatabaseDropEmptyDBRule checks for drop database only if empty.
type DatabaseDropEmptyDBRule struct {
	BaseRule
	originMetadata *model.DatabaseMetadata
}

// NewDatabaseDropEmptyDBRule creates a new DatabaseDropEmptyDBRule.
func NewDatabaseDropEmptyDBRule(level storepb.Advice_Status, title string, originMetadata *model.DatabaseMetadata) *DatabaseDropEmptyDBRule {
	return &DatabaseDropEmptyDBRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		originMetadata: originMetadata,
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
	if r.originMetadata.DatabaseName() != dbName {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", dbName, r.originMetadata.DatabaseName()),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	} else if !r.originMetadata.HasNoTable() {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DatabaseNotEmpty.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Database `%s` is not allowed to drop if not empty", dbName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
