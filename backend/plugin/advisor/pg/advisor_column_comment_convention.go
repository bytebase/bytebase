package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnCommentConvention, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalCommentConventionRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := &columnCommentConventionRule{
		level:                level,
		title:                string(checkCtx.Rule.Type),
		payload:              payload,
		classificationConfig: checkCtx.ClassificationConfig,
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Now validate all collected columns against comments
	return rule.generateAdvice(), nil
}

type columnInfo struct {
	schema string
	table  string
	column string
	line   int
}

type commentInfo struct {
	schema  string
	table   string
	column  string
	comment string
	line    int
}

type columnCommentConventionRule struct {
	BaseRule

	level                storepb.Advice_Status
	title                string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	columns  []columnInfo
	comments []commentInfo
}

func (*columnCommentConventionRule) Name() string {
	return "column_comment_convention"
}

func (r *columnCommentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Commentstmt":
		r.handleCommentstmt(ctx.(*parser.CommentstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*columnCommentConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *columnCommentConventionRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	tableName := r.extractTableName(ctx.AllQualified_name())
	if tableName == "" {
		return
	}

	// Extract all columns
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil && elem.ColumnDef().Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(elem.ColumnDef().Colid())
				r.columns = append(r.columns, columnInfo{
					schema: "public", // Default schema
					table:  tableName,
					column: columnName,
					line:   elem.ColumnDef().GetStart().GetLine(),
				})
			}
		}
	}
}

func (r *columnCommentConventionRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := ctx.Relation_expr().Qualified_name().GetText()
	if tableName == "" {
		return
	}

	// Check ALTER TABLE ADD COLUMN
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil && cmd.ColumnDef().Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(cmd.ColumnDef().Colid())
				r.columns = append(r.columns, columnInfo{
					schema: "public", // Default schema
					table:  tableName,
					column: columnName,
					line:   cmd.ColumnDef().GetStart().GetLine(),
				})
			}
		}
	}
}

func (r *columnCommentConventionRule) handleCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a COMMENT ON COLUMN statement
	if ctx.COLUMN() == nil || ctx.Any_name() == nil {
		return
	}

	// Extract table.column name from any_name
	// any_name is like: table.column or schema.table.column
	anyName := ctx.Any_name()
	parts := pg.NormalizePostgreSQLAnyName(anyName)
	if len(parts) < 2 {
		return
	}

	tableName := parts[len(parts)-2]
	columnName := parts[len(parts)-1]

	// Extract comment text
	comment := ""
	if ctx.Comment_text() != nil && ctx.Comment_text().Sconst() != nil {
		comment = extractStringConstant(ctx.Comment_text().Sconst())
	}

	r.comments = append(r.comments, commentInfo{
		schema:  "public",
		table:   tableName,
		column:  columnName,
		comment: comment,
		line:    ctx.GetStart().GetLine(),
	})
}

func (*columnCommentConventionRule) extractTableName(qualifiedNames []parser.IQualified_nameContext) string {
	if len(qualifiedNames) == 0 {
		return ""
	}

	// Return the last part (table name) from qualified name
	return extractTableName(qualifiedNames[0])
}

func (r *columnCommentConventionRule) generateAdvice() []*storepb.Advice {
	var adviceList []*storepb.Advice

	// For each column, find its comment and validate
	for _, col := range r.columns {
		// Find the last matching comment for this column
		var matchedComment *commentInfo
		for i := range r.comments {
			comment := &r.comments[i]
			if comment.schema == col.schema && comment.table == col.table && comment.column == col.column {
				matchedComment = comment
				// Continue to find the last one
			}
		}

		if matchedComment == nil || matchedComment.comment == "" {
			if r.payload.Required {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  r.level,
					Code:    code.CommentEmpty.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("Comment is required for column `%s.%s`", col.table, col.column),
					StartPosition: &storepb.Position{
						Line:   int32(col.line),
						Column: 0,
					},
				})
			}
		} else {
			comment := matchedComment.comment

			// Check max length
			if r.payload.MaxLength > 0 && len(comment) > r.payload.MaxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  r.level,
					Code:    code.CommentTooLong.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("Column `%s.%s` comment is too long. The length of comment should be within %d characters", col.table, col.column, r.payload.MaxLength),
					StartPosition: &storepb.Position{
						Line:   int32(matchedComment.line),
						Column: 0,
					},
				})
			}

			// Check classification
			if r.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, r.classificationConfig); classification == "" {
					adviceList = append(adviceList, &storepb.Advice{
						Status:  r.level,
						Code:    code.CommentMissingClassification.Int32(),
						Title:   r.title,
						Content: fmt.Sprintf("Column `%s.%s` comment requires classification", col.table, col.column),
						StartPosition: &storepb.Position{
							Line:   int32(matchedComment.line),
							Column: 0,
						},
					})
				}
			}
		}
	}

	return adviceList
}
