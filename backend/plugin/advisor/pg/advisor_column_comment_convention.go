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
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
	_ ast.Visitor     = (*columnCommentConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLColumnCommentConvention, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	checker := &columnCommentConventionChecker{
		level:                level,
		title:                string(checkCtx.Rule.Type),
		payload:              payload,
		classificationConfig: checkCtx.ClassificationConfig,
	}

	for _, stmt := range stmtList {
		ast.Walk(checker, stmt)
	}

	for _, columnName := range checker.columnNameList {
		var commentStmt *ast.CommentStmt
		for _, stmt := range checker.commentStmtList {
			columnNameDef, ok := stmt.Object.(*ast.ColumnNameDef)
			if !ok {
				continue
			}
			if columnNameDef.ColumnName == columnName.ColumnName && columnNameDef.Table.Name == columnName.Table.Name {
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
					Content:       fmt.Sprintf("Comment is required for column `%s`", stringifyColumnNameDef(columnName)),
					StartPosition: common.ConvertPGParserLineToPosition(columnName.LastLine()),
				})
			}
		} else {
			comment := commentStmt.Comment
			if checker.payload.MaxLength > 0 && len(comment) > checker.payload.MaxLength {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          advisor.CommentTooLong.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("Column `%s` comment is too long. The length of comment should be within %d characters", stringifyColumnNameDef(columnName), checker.payload.MaxLength),
					StartPosition: common.ConvertPGParserLineToPosition(commentStmt.LastLine()),
				})
			}
			if checker.payload.RequiredClassification {
				if classification, _ := common.GetClassificationAndUserComment(comment, checker.classificationConfig); classification == "" {
					checker.adviceList = append(checker.adviceList, &storepb.Advice{
						Status:        checker.level,
						Code:          advisor.CommentMissingClassification.Int32(),
						Title:         checker.title,
						Content:       fmt.Sprintf("Column `%s` comment requires classification", stringifyColumnNameDef(columnName)),
						StartPosition: common.ConvertPGParserLineToPosition(commentStmt.LastLine()),
					})
				}
			}
		}
	}

	return checker.adviceList, nil
}

type columnCommentConventionChecker struct {
	adviceList           []*storepb.Advice
	level                storepb.Advice_Status
	title                string
	payload              *advisor.CommentConventionRulePayload
	classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig

	columnNameList  []*ast.ColumnNameDef
	commentStmtList []*ast.CommentStmt
}

func (checker *columnCommentConventionChecker) Visit(node ast.Node) ast.Visitor {
	if createTableStmt, ok := node.(*ast.CreateTableStmt); ok {
		for _, columnDef := range createTableStmt.ColumnList {
			columnName := &ast.ColumnNameDef{
				Table:      createTableStmt.Name,
				ColumnName: columnDef.ColumnName,
			}
			columnName.SetLastLine(createTableStmt.LastLine())
			checker.columnNameList = append(checker.columnNameList, columnName)
		}
	} else if alterTableStmt, ok := node.(*ast.AlterTableStmt); ok {
		for _, alterItem := range alterTableStmt.AlterItemList {
			if addColumnListStmt, ok := alterItem.(*ast.AddColumnListStmt); ok {
				for _, columnDef := range addColumnListStmt.ColumnList {
					columnName := &ast.ColumnNameDef{
						Table:      alterTableStmt.Table,
						ColumnName: columnDef.ColumnName,
					}
					columnName.SetLastLine(alterTableStmt.LastLine())
					checker.columnNameList = append(checker.columnNameList, columnName)
				}
			}
		}
	} else if commentStmt, ok := node.(*ast.CommentStmt); ok && commentStmt.Type == ast.ObjectTypeColumn {
		checker.commentStmtList = append(checker.commentStmtList, commentStmt)
	}
	return checker
}

func stringifyColumnNameDef(columnName *ast.ColumnNameDef) string {
	if columnName == nil {
		return ""
	}
	if columnName.Table == nil {
		return columnName.ColumnName
	}
	return fmt.Sprintf("%s.%s", columnName.Table.Name, columnName.ColumnName)
}
