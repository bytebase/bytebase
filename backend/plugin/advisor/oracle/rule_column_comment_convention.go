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
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := NewColumnCommentConventionRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, payload, checkCtx.ClassificationConfig)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// ColumnCommentConventionRule is the rule implementation for column comment convention.
type ColumnCommentConventionRule struct {
	BaseRule

	currentDatabase      string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	tableName     string
	columnNames   []string
	columnComment map[string]string
	columnLine    map[string]int
}

// NewColumnCommentConventionRule creates a new ColumnCommentConventionRule.
func NewColumnCommentConventionRule(level storepb.Advice_Status, title string, currentDatabase string, payload *advisor.CommentConventionRulePayload, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) *ColumnCommentConventionRule {
	return &ColumnCommentConventionRule{
		BaseRule:             NewBaseRule(level, title, 0),
		currentDatabase:      currentDatabase,
		payload:              payload,
		classificationConfig: classificationConfig,
		columnNames:          []string{},
		columnComment:        make(map[string]string),
		columnLine:           make(map[string]int),
	}
}

// Name returns the rule name.
func (*ColumnCommentConventionRule) Name() string {
	return "column.comment-convention"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnCommentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "Column_definition":
		r.handleColumnDefinition(ctx.(*parser.Column_definitionContext))
	case "Alter_table":
		r.handleAlterTable(ctx.(*parser.Alter_tableContext))
	case "Comment_on_column":
		r.handleCommentOnColumn(ctx.(*parser.Comment_on_columnContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *ColumnCommentConventionRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTableExit()
	case "Add_column_clause":
		r.handleAddColumnClauseExit()
	default:
	}
	return nil
}

func (r *ColumnCommentConventionRule) handleCreateTable(ctx *parser.Create_tableContext) {
	schemaName := r.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), r.currentDatabase)
	}
	r.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

func (r *ColumnCommentConventionRule) handleCreateTableExit() {
	r.tableName = ""
}

func (r *ColumnCommentConventionRule) handleColumnDefinition(ctx *parser.Column_definitionContext) {
	if r.tableName == "" {
		return
	}
	columnName := fmt.Sprintf(`%s.%s`, r.tableName, normalizeIdentifier(ctx.Column_name(), r.currentDatabase))
	r.columnNames = append(r.columnNames, columnName)
	r.columnLine[columnName] = r.baseLine + ctx.GetStart().GetLine()
}

func (r *ColumnCommentConventionRule) handleAlterTable(ctx *parser.Alter_tableContext) {
	r.tableName = normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
}

func (r *ColumnCommentConventionRule) handleAddColumnClauseExit() {
	r.tableName = ""
}

func (r *ColumnCommentConventionRule) handleCommentOnColumn(ctx *parser.Comment_on_columnContext) {
	if ctx.Column_name() == nil {
		return
	}

	columnName := fmt.Sprintf(`%s.%s`, r.currentDatabase, normalizeIdentifier(ctx.Column_name(), ""))
	r.columnComment[columnName] = plsqlparser.NormalizeQuotedString(ctx.Quoted_string())
}

// GetAdviceList returns the advice list.
// We override this to perform final checks after all statements have been processed.
func (r *ColumnCommentConventionRule) GetAdviceList() ([]*storepb.Advice, error) {
	for _, columnName := range r.columnNames {
		comment, ok := r.columnComment[columnName]
		if !ok || comment == "" {
			if r.payload.Required {
				r.AddAdvice(
					r.level,
					code.CommentEmpty.Int32(),
					fmt.Sprintf("Comment is required for column %s", normalizeIdentifierName(columnName)),
					common.ConvertANTLRLineToPosition(r.columnLine[columnName]),
				)
			}
		} else {
			if r.payload.MaxLength > 0 && len(comment) > r.payload.MaxLength {
				r.AddAdvice(
					r.level,
					code.CommentTooLong.Int32(),
					fmt.Sprintf("Column %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(columnName), r.payload.MaxLength),
					common.ConvertANTLRLineToPosition(r.columnLine[columnName]),
				)
			}
			if r.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, r.classificationConfig); classification == "" {
					r.AddAdvice(
						r.level,
						code.CommentMissingClassification.Int32(),
						fmt.Sprintf("Column %s comment requires classification", normalizeIdentifierName(columnName)),
						common.ConvertANTLRLineToPosition(r.columnLine[columnName]),
					)
				}
			}
		}
	}
	return r.BaseRule.GetAdviceList()
}
