package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
	_ ast.Visitor     = (*tableCommentConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableCommentConvention, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
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

	for _, stmt := range stmtList {
		ast.Walk(checker, stmt)
	}

	for _, createTableStmt := range checker.createdTableStmtList {
		var commentStmt *ast.CommentStmt
		for _, stmt := range checker.commentStmtList {
			tableDef, ok := stmt.Object.(*ast.TableDef)
			if !ok {
				continue
			}
			if tableDef.Name == createTableStmt.Name.Name && tableDef.Schema == createTableStmt.Name.Schema && tableDef.Database == createTableStmt.Name.Database {
				commentStmt = stmt
				// continue and find the last comment statement.
			}
		}
		if commentStmt == nil || commentStmt.Comment == "" {
			if checker.payload.Required {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          advisor.CommentEmpty.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("Comment is required for table `%s`", stringifyTableDef(createTableStmt.Name)),
					StartPosition: common.ConvertPGParserLineToPosition(createTableStmt.LastLine()),
				})
			}
		} else {
			comment := commentStmt.Comment
			if checker.payload.MaxLength > 0 && len(comment) > checker.payload.MaxLength {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          advisor.CommentTooLong.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("Table `%s` comment is too long. The length of comment should be within %d characters", stringifyTableDef(createTableStmt.Name), checker.payload.MaxLength),
					StartPosition: common.ConvertPGParserLineToPosition(commentStmt.LastLine()),
				})
			}
			if checker.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, checker.classificationConfig); classification == "" {
					checker.adviceList = append(checker.adviceList, &storepb.Advice{
						Status:        checker.level,
						Code:          advisor.CommentMissingClassification.Int32(),
						Title:         checker.title,
						Content:       fmt.Sprintf("Table `%s` comment requires classification", stringifyTableDef(createTableStmt.Name)),
						StartPosition: common.ConvertPGParserLineToPosition(commentStmt.LastLine()),
					})
				}
			}
		}
	}

	return checker.adviceList, nil
}

type tableCommentConventionChecker struct {
	adviceList           []*storepb.Advice
	level                storepb.Advice_Status
	title                string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	createdTableStmtList []*ast.CreateTableStmt
	commentStmtList      []*ast.CommentStmt
}

func (checker *tableCommentConventionChecker) Visit(node ast.Node) ast.Visitor {
	if createTableStmt, ok := node.(*ast.CreateTableStmt); ok {
		checker.createdTableStmtList = append(checker.createdTableStmtList, createTableStmt)
	} else if commentStmt, ok := node.(*ast.CommentStmt); ok && commentStmt.Type == ast.ObjectTypeTable {
		checker.commentStmtList = append(checker.commentStmtList, commentStmt)
	}
	return checker
}

func stringifyTableDef(tableDef *ast.TableDef) string {
	if tableDef == nil {
		return ""
	}
	if tableDef.Database == "" && tableDef.Schema == "" {
		return tableDef.Name
	}
	if tableDef.Database == "" {
		return fmt.Sprintf("%s.%s", tableDef.Schema, tableDef.Name)
	}
	return fmt.Sprintf("%s.%s.%s", tableDef.Database, tableDef.Schema, tableDef.Name)
}
