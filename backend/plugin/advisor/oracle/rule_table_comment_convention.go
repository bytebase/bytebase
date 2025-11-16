// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleTableCommentConvention, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalCommentConventionRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := NewTableCommentConventionRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, payload, checkCtx.ClassificationConfig)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// TableCommentConventionRule is the rule implementation for table comment convention.
type TableCommentConventionRule struct {
	BaseRule

	currentDatabase      string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	tableNames   []string
	tableComment map[string]string
	tableLine    map[string]int
}

// NewTableCommentConventionRule creates a new TableCommentConventionRule.
func NewTableCommentConventionRule(level storepb.Advice_Status, title string, currentDatabase string, payload *advisor.CommentConventionRulePayload, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) *TableCommentConventionRule {
	return &TableCommentConventionRule{
		BaseRule:             NewBaseRule(level, title, 0),
		currentDatabase:      currentDatabase,
		payload:              payload,
		classificationConfig: classificationConfig,
		tableNames:           []string{},
		tableComment:         make(map[string]string),
		tableLine:            make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableCommentConventionRule) Name() string {
	return "table.comment-convention"
}

// OnEnter is called when the parser enters a rule context.
func (r *TableCommentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "Comment_on_table":
		r.handleCommentOnTable(ctx.(*parser.Comment_on_tableContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*TableCommentConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableCommentConventionRule) handleCreateTable(ctx *parser.Create_tableContext) {
	schemaName := r.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), r.currentDatabase)
	}

	tableName := fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), r.currentDatabase))
	r.tableNames = append(r.tableNames, tableName)
	r.tableLine[tableName] = r.baseLine + ctx.GetStart().GetLine()
}

func (r *TableCommentConventionRule) handleCommentOnTable(ctx *parser.Comment_on_tableContext) {
	if ctx.Tableview_name() == nil {
		return
	}

	tableName := normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
	r.tableComment[tableName] = plsqlparser.NormalizeQuotedString(ctx.Quoted_string())
}

// GetAdviceList returns the advice list.
func (r *TableCommentConventionRule) GetAdviceList() ([]*storepb.Advice, error) {
	for _, tableName := range r.tableNames {
		comment, ok := r.tableComment[tableName]
		if !ok || comment == "" {
			if r.payload.Required {
				r.AddAdvice(
					r.level,
					code.CommentEmpty.Int32(),
					fmt.Sprintf("Comment is required for table %s", normalizeIdentifierName(tableName)),
					common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
				)
			}
		} else {
			if r.payload.MaxLength > 0 && len(comment) > r.payload.MaxLength {
				r.AddAdvice(
					r.level,
					code.CommentTooLong.Int32(),
					fmt.Sprintf("Table %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(tableName), r.payload.MaxLength),
					common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
				)
			}
			if r.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, r.classificationConfig); classification == "" {
					r.AddAdvice(
						r.level,
						code.CommentMissingClassification.Int32(),
						fmt.Sprintf("Table %s comment requires classification", normalizeIdentifierName(tableName)),
						common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
					)
				}
			}
		}
	}

	return r.BaseRule.GetAdviceList()
}
