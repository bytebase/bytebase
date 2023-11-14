package mysqlwip

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLTableCommentConvention, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

// Check checks for table comment convention.
func (*TableCommentConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalCommentConventionRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &tableCommentConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		required:  payload.Required,
		maxLength: payload.MaxLength,
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

type tableCommentConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	required   bool
	maxLength  int
}

// EnterCreateTable is called when production createTable is entered.
func (checker *tableCommentConventionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())

	comment, exists := checker.handleCreateTableOptions(ctx.CreateTableOptions())

	if checker.required && !exists {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NoTableComment,
			Title:   checker.title,
			Content: fmt.Sprintf("Table `%s` requires comments", tableName),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
	if checker.maxLength >= 0 && len(comment) > checker.maxLength {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.TableCommentTooLong,
			Title:   checker.title,
			Content: fmt.Sprintf("The length of table `%s` comment should be within %d characters", tableName, checker.maxLength),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}

func (*tableCommentConventionChecker) handleCreateTableOptions(ctx mysql.ICreateTableOptionsContext) (string, bool) {
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
