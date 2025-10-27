package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableCommentConvention, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	checker := &tableCommentConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		payload:                      payload,
		classificationConfig:         checkCtx.ClassificationConfig,
		statementsText:               checkCtx.Statements,
		createdTables:                make(map[string]*tableInfo),
		tableComments:                make(map[string]*tableCommentInfo),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Check each created table for comment requirements
	for tableKey, tableInfo := range checker.createdTables {
		tableCommentInfo, hasComment := checker.tableComments[tableKey]

		if !hasComment || tableCommentInfo.comment == "" {
			if checker.payload.Required {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:  checker.level,
					Code:    advisor.CommentEmpty.Int32(),
					Title:   checker.title,
					Content: fmt.Sprintf("Comment is required for table `%s`", tableInfo.displayName),
					StartPosition: &storepb.Position{
						Line:   int32(tableInfo.line),
						Column: 0,
					},
				})
			}
		} else {
			comment := tableCommentInfo.comment
			if checker.payload.MaxLength > 0 && len(comment) > checker.payload.MaxLength {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:  checker.level,
					Code:    advisor.CommentTooLong.Int32(),
					Title:   checker.title,
					Content: fmt.Sprintf("Table `%s` comment is too long. The length of comment should be within %d characters", tableInfo.displayName, checker.payload.MaxLength),
					StartPosition: &storepb.Position{
						Line:   int32(tableCommentInfo.line),
						Column: 0,
					},
				})
			}
			if checker.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, checker.classificationConfig); classification == "" {
					checker.adviceList = append(checker.adviceList, &storepb.Advice{
						Status:  checker.level,
						Code:    advisor.CommentMissingClassification.Int32(),
						Title:   checker.title,
						Content: fmt.Sprintf("Table `%s` comment requires classification", tableInfo.displayName),
						StartPosition: &storepb.Position{
							Line:   int32(tableCommentInfo.line),
							Column: 0,
						},
					})
				}
			}
		}
	}

	return checker.adviceList, nil
}

type tableInfo struct {
	schema      string
	tableName   string
	displayName string
	line        int
}

type tableCommentInfo struct {
	comment string
	line    int
}

type tableCommentConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList           []*storepb.Advice
	level                storepb.Advice_Status
	title                string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig
	statementsText       string

	createdTables map[string]*tableInfo
	tableComments map[string]*tableCommentInfo
}

// EnterCreatestmt collects CREATE TABLE statements
func (c *tableCommentConventionChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName = extractTableName(allQualifiedNames[0])
		schemaName = extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			schemaName = "public"
		}
	}

	tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)
	// Only include schema in display name if it's not the default "public" schema
	displayName := tableName
	if schemaName != "public" {
		displayName = fmt.Sprintf("%s.%s", schemaName, tableName)
	}

	c.createdTables[tableKey] = &tableInfo{
		schema:      schemaName,
		tableName:   tableName,
		displayName: displayName,
		line:        ctx.GetStart().GetLine(),
	}
}

// EnterCommentstmt collects COMMENT ON TABLE statements
func (c *tableCommentConventionChecker) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is COMMENT ON TABLE
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Extract table name from Any_name
	if ctx.Any_name() == nil {
		return
	}

	parts := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())
	if len(parts) == 0 {
		return
	}

	var schemaName, tableName string
	if len(parts) == 1 {
		schemaName = "public"
		tableName = parts[0]
	} else {
		schemaName = parts[0]
		tableName = parts[1]
	}

	tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)

	// Extract comment text
	comment := ""
	if ctx.Comment_text() != nil && ctx.Comment_text().Sconst() != nil {
		commentText := ctx.Comment_text().Sconst().GetText()
		// Remove surrounding quotes
		if len(commentText) >= 2 {
			comment = commentText[1 : len(commentText)-1]
		}
	}

	c.tableComments[tableKey] = &tableCommentInfo{
		comment: comment,
		line:    ctx.GetStart().GetLine(),
	}
}
