package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableCommentConvention, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLTableCommentConvention, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLTableCommentConvention, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

// Check checks for table comment convention.
func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalCommentConventionRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &tableCommentConventionChecker{
		level:                level,
		title:                string(checkCtx.Rule.Type),
		payload:              payload,
		classificationConfig: checkCtx.ClassificationConfig,
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type tableCommentConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine             int
	adviceList           []*storepb.Advice
	level                storepb.Advice_Status
	title                string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig
}

// EnterCreateTable is called when production createTable is entered.
func (checker *tableCommentConventionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())

	comment, exists := checker.handleCreateTableOptions(ctx.CreateTableOptions())

	if checker.payload.Required && !exists {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.CommentEmpty.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("Table `%s` requires comments", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(checker.baseLine + ctx.GetStart().GetLine()),
		})
	}
	if checker.payload.MaxLength >= 0 && len(comment) > checker.payload.MaxLength {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.CommentTooLong.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("The length of table `%s` comment should be within %d characters", tableName, checker.payload.MaxLength),
			StartPosition: common.ConvertANTLRLineToPosition(checker.baseLine + ctx.GetStart().GetLine()),
		})
	}
	if checker.payload.RequiredClassification {
		if classification, _ := common.GetClassificationAndUserComment(comment, checker.classificationConfig); classification == "" {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.CommentMissingClassification.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("Table `%s` comment requires classification", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(checker.baseLine + ctx.GetStart().GetLine()),
			})
		}
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
