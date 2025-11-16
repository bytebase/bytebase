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

	rule := &tableCommentConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		payload:              payload,
		classificationConfig: checkCtx.ClassificationConfig,
		statementsText:       checkCtx.Statements,
		createdTables:        make(map[string]*tableInfo),
		tableComments:        make(map[string]*tableCommentInfo),
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Check each created table for comment requirements
	for tableKey, tableInfo := range rule.createdTables {
		tableCommentInfo, hasComment := rule.tableComments[tableKey]

		if !hasComment || tableCommentInfo.comment == "" {
			if rule.payload.Required {
				rule.AddAdvice(&storepb.Advice{
					Status:  rule.level,
					Code:    code.CommentEmpty.Int32(),
					Title:   rule.title,
					Content: fmt.Sprintf("Comment is required for table `%s`", tableInfo.displayName),
					StartPosition: &storepb.Position{
						Line:   int32(tableInfo.line),
						Column: 0,
					},
				})
			}
		} else {
			comment := tableCommentInfo.comment
			if rule.payload.MaxLength > 0 && len(comment) > rule.payload.MaxLength {
				rule.AddAdvice(&storepb.Advice{
					Status:  rule.level,
					Code:    code.CommentTooLong.Int32(),
					Title:   rule.title,
					Content: fmt.Sprintf("Table `%s` comment is too long. The length of comment should be within %d characters", tableInfo.displayName, rule.payload.MaxLength),
					StartPosition: &storepb.Position{
						Line:   int32(tableCommentInfo.line),
						Column: 0,
					},
				})
			}
			if rule.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, rule.classificationConfig); classification == "" {
					rule.AddAdvice(&storepb.Advice{
						Status:  rule.level,
						Code:    code.CommentMissingClassification.Int32(),
						Title:   rule.title,
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

	return checker.GetAdviceList(), nil
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

type tableCommentConventionRule struct {
	BaseRule

	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig
	statementsText       string

	createdTables map[string]*tableInfo
	tableComments map[string]*tableCommentInfo
}

// Name returns the rule name.
func (*tableCommentConventionRule) Name() string {
	return "table-comment-convention"
}

// OnEnter handles entering a parse tree node.
func (r *tableCommentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx)
	case "Commentstmt":
		r.handleCommentstmt(ctx)
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit handles exiting a parse tree node.
func (*tableCommentConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt collects CREATE TABLE statements
func (r *tableCommentConventionRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createstmtCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(createstmtCtx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := createstmtCtx.AllQualified_name()
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

	r.createdTables[tableKey] = &tableInfo{
		schema:      schemaName,
		tableName:   tableName,
		displayName: displayName,
		line:        createstmtCtx.GetStart().GetLine(),
	}
}

// handleCommentstmt collects COMMENT ON TABLE statements
func (r *tableCommentConventionRule) handleCommentstmt(ctx antlr.ParserRuleContext) {
	commentstmtCtx, ok := ctx.(*parser.CommentstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(commentstmtCtx.GetParent()) {
		return
	}

	// Check if this is COMMENT ON TABLE
	if commentstmtCtx.Object_type_any_name() == nil || commentstmtCtx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Extract table name from Any_name
	if commentstmtCtx.Any_name() == nil {
		return
	}

	parts := pgparser.NormalizePostgreSQLAnyName(commentstmtCtx.Any_name())
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
	if commentstmtCtx.Comment_text() != nil && commentstmtCtx.Comment_text().Sconst() != nil {
		commentText := commentstmtCtx.Comment_text().Sconst().GetText()
		// Remove surrounding quotes
		if len(commentText) >= 2 {
			comment = commentText[1 : len(commentText)-1]
		}
	}

	r.tableComments[tableKey] = &tableCommentInfo{
		comment: comment,
		line:    commentstmtCtx.GetStart().GetLine(),
	}
}
