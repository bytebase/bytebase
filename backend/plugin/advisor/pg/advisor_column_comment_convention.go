package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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

	checker := &columnCommentConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		payload:                      payload,
		classificationConfig:         checkCtx.ClassificationConfig,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Now validate all collected columns against comments
	return checker.generateAdvice(), nil
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

type columnCommentConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	level                storepb.Advice_Status
	title                string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	columns  []columnInfo
	comments []commentInfo
}

func (c *columnCommentConventionChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	tableName := c.extractTableName(ctx.AllQualified_name())
	if tableName == "" {
		return
	}

	// Extract all columns
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil && elem.ColumnDef().Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(elem.ColumnDef().Colid())
				c.columns = append(c.columns, columnInfo{
					schema: "public", // Default schema
					table:  tableName,
					column: columnName,
					line:   ctx.GetStart().GetLine(),
				})
			}
		}
	}
}

func (c *columnCommentConventionChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
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
				c.columns = append(c.columns, columnInfo{
					schema: "public", // Default schema
					table:  tableName,
					column: columnName,
					line:   ctx.GetStart().GetLine(),
				})
			}
		}
	}
}

func (c *columnCommentConventionChecker) EnterCommentstmt(ctx *parser.CommentstmtContext) {
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

	c.comments = append(c.comments, commentInfo{
		schema:  "public",
		table:   tableName,
		column:  columnName,
		comment: comment,
		line:    ctx.GetStart().GetLine(),
	})
}

func (*columnCommentConventionChecker) extractTableName(qualifiedNames []parser.IQualified_nameContext) string {
	if len(qualifiedNames) == 0 {
		return ""
	}

	// Return the last part (table name) from qualified name
	return extractTableName(qualifiedNames[0])
}

func (c *columnCommentConventionChecker) generateAdvice() []*storepb.Advice {
	var adviceList []*storepb.Advice

	// For each column, find its comment and validate
	for _, col := range c.columns {
		// Find the last matching comment for this column
		var matchedComment *commentInfo
		for i := range c.comments {
			comment := &c.comments[i]
			if comment.schema == col.schema && comment.table == col.table && comment.column == col.column {
				matchedComment = comment
				// Continue to find the last one
			}
		}

		if matchedComment == nil || matchedComment.comment == "" {
			if c.payload.Required {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  c.level,
					Code:    advisor.CommentEmpty.Int32(),
					Title:   c.title,
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
			if c.payload.MaxLength > 0 && len(comment) > c.payload.MaxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  c.level,
					Code:    advisor.CommentTooLong.Int32(),
					Title:   c.title,
					Content: fmt.Sprintf("Column `%s.%s` comment is too long. The length of comment should be within %d characters", col.table, col.column, c.payload.MaxLength),
					StartPosition: &storepb.Position{
						Line:   int32(matchedComment.line),
						Column: 0,
					},
				})
			}

			// Check classification
			if c.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, c.classificationConfig); classification == "" {
					adviceList = append(adviceList, &storepb.Advice{
						Status:  c.level,
						Code:    advisor.CommentMissingClassification.Int32(),
						Title:   c.title,
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
