package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

// Check checks for column comment convention.
func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	commentPayload := checkCtx.Rule.GetCommentConventionPayload()

	rule := &columnCommentConventionOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		payload: commentPayload,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnCommentConventionOmniRule struct {
	OmniBaseRule
	payload *storepb.SQLReviewRule_CommentConventionRulePayload
}

func (*columnCommentConventionOmniRule) Name() string {
	return "ColumnCommentConventionRule"
}

func (r *columnCommentConventionOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnCommentConventionOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}

	for _, col := range n.Columns {
		r.checkColumn(tableName, col)
	}
}

func (r *columnCommentConventionOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}

	for _, cmd := range n.Commands {
		switch cmd.Type {
		case ast.ATAddColumn:
			if cmd.Column != nil {
				r.checkColumn(tableName, cmd.Column)
			}
			for _, col := range cmd.Columns {
				r.checkColumn(tableName, col)
			}
		case ast.ATModifyColumn, ast.ATChangeColumn:
			if cmd.Column != nil {
				r.checkColumn(tableName, cmd.Column)
			}
		default:
		}
	}
}

func (r *columnCommentConventionOmniRule) checkColumn(tableName string, col *ast.ColumnDef) {
	comment := omniColumnComment(col)

	if comment != "" && r.payload.MaxLength >= 0 && int32(len(comment)) > r.payload.MaxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.CommentTooLong.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", tableName, col.Name, r.payload.MaxLength),
			StartPosition: &storepb.Position{
				Line:   r.LocToLine(col.Loc),
				Column: 0,
			},
		})
	}

	if comment == "" && r.payload.Required {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.CommentEmpty.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Column `%s`.`%s` requires comments", tableName, col.Name),
			StartPosition: &storepb.Position{
				Line:   r.LocToLine(col.Loc),
				Column: 0,
			},
		})
	}
}
