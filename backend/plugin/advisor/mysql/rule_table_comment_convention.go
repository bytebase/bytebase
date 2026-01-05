package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

// Check checks for table comment convention.
func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	commentPayload := checkCtx.Rule.GetCommentConventionPayload()

	// Create the rule
	rule := NewTableCommentConventionRule(level, checkCtx.Rule.Type.String(), commentPayload)

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

// TableCommentConventionRule checks for table comment convention.
type TableCommentConventionRule struct {
	BaseRule
	payload *storepb.SQLReviewRule_CommentConventionRulePayload
}

// NewTableCommentConventionRule creates a new TableCommentConventionRule.
func NewTableCommentConventionRule(level storepb.Advice_Status, title string, payload *storepb.SQLReviewRule_CommentConventionRulePayload) *TableCommentConventionRule {
	return &TableCommentConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		payload: payload,
	}
}

// Name returns the rule name.
func (*TableCommentConventionRule) Name() string {
	return "TableCommentConventionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableCommentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateTable {
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableCommentConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableCommentConventionRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())

	comment, exists := r.handleCreateTableOptions(ctx.CreateTableOptions())

	if r.payload.Required && !exists {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.CommentEmpty.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table `%s` requires comments", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
	if r.payload.MaxLength >= 0 && int32(len(comment)) > r.payload.MaxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.CommentTooLong.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("The length of table `%s` comment should be within %d characters", tableName, r.payload.MaxLength),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (*TableCommentConventionRule) handleCreateTableOptions(ctx mysql.ICreateTableOptionsContext) (string, bool) {
	if ctx == nil {
		return "", false
	}
	for _, option := range ctx.AllCreateTableOption() {
		if option.COMMENT_SYMBOL() == nil || option.TextStringLiteral() == nil {
			continue
		}

		comment := mysqlparser.NormalizeMySQLTextStringLiteral(option.TextStringLiteral())
		return comment, true
	}
	return "", false
}
