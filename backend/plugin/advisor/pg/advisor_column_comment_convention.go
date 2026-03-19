package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
type ColumnCommentConventionAdvisor struct {
}

func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	commentPayload := checkCtx.Rule.GetCommentConventionPayload()

	rule := &columnCommentConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		payload: commentPayload,
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.generateAdvice(), nil
}

type columnInfo struct {
	schema string
	table  string
	column string
	line   int32
}

type commentInfo struct {
	schema  string
	table   string
	column  string
	comment string
	line    int32
}

type columnCommentConventionRule struct {
	OmniBaseRule

	payload *storepb.SQLReviewRule_CommentConventionRulePayload

	columns  []columnInfo
	comments []commentInfo
}

func (*columnCommentConventionRule) Name() string {
	return "column_comment_convention"
}

func (r *columnCommentConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	case *ast.CommentStmt:
		r.handleCommentStmt(n)
	default:
	}
}

func (r *columnCommentConventionRule) handleCreateStmt(n *ast.CreateStmt) {
	if n.Relation == nil {
		return
	}
	tableName := n.Relation.Relname
	cols, _ := omniTableElements(n)
	for _, col := range cols {
		r.columns = append(r.columns, columnInfo{
			schema: "public",
			table:  tableName,
			column: col.Colname,
			line:   r.absoluteLine(),
		})
	}
}

func (r *columnCommentConventionRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	if n.Relation == nil {
		return
	}
	tableName := n.Relation.Relname
	for _, cmd := range omniAlterTableCmds(n) {
		if cmd.Subtype == int(ast.AT_AddColumn) {
			if colDef, ok := cmd.Def.(*ast.ColumnDef); ok {
				r.columns = append(r.columns, columnInfo{
					schema: "public",
					table:  tableName,
					column: colDef.Colname,
					line:   r.absoluteLine(),
				})
			}
		}
	}
}

func (r *columnCommentConventionRule) handleCommentStmt(n *ast.CommentStmt) {
	if n.Objtype != ast.OBJECT_COLUMN {
		return
	}

	list, ok := n.Object.(*ast.List)
	if !ok || len(list.Items) < 2 {
		return
	}

	var tableName, columnName string
	items := list.Items
	switch len(items) {
	case 2:
		tableName = omniStringVal(items[0])
		columnName = omniStringVal(items[1])
	case 3:
		tableName = omniStringVal(items[1])
		columnName = omniStringVal(items[2])
	default:
		return
	}

	r.comments = append(r.comments, commentInfo{
		schema:  "public",
		table:   tableName,
		column:  columnName,
		comment: n.Comment,
		line:    r.absoluteLine(),
	})
}

// absoluteLine returns the absolute 1-based line number for the current statement.
func (r *columnCommentConventionRule) absoluteLine() int32 {
	return r.ContentStartLine() + int32(r.BaseLine)
}

func (r *columnCommentConventionRule) generateAdvice() []*storepb.Advice {
	var adviceList []*storepb.Advice

	for _, col := range r.columns {
		var matchedComment *commentInfo
		for i := range r.comments {
			comment := &r.comments[i]
			if comment.schema == col.schema && comment.table == col.table && comment.column == col.column {
				matchedComment = comment
			}
		}

		if matchedComment == nil || matchedComment.comment == "" {
			if r.payload.Required {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  r.Level,
					Code:    code.CommentEmpty.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("Comment is required for column `%s.%s`", col.table, col.column),
					StartPosition: &storepb.Position{
						Line:   col.line,
						Column: 0,
					},
				})
			}
		} else {
			if r.payload.MaxLength > 0 && int32(len(matchedComment.comment)) > r.payload.MaxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  r.Level,
					Code:    code.CommentTooLong.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("Column `%s.%s` comment is too long. The length of comment should be within %d characters", col.table, col.column, r.payload.MaxLength),
					StartPosition: &storepb.Position{
						Line:   matchedComment.line,
						Column: 0,
					},
				})
			}
		}
	}

	return adviceList
}

// omniStringVal extracts a string value from a Node.
func omniStringVal(n ast.Node) string {
	if s, ok := n.(*ast.String); ok {
		return s.Str
	}
	return ""
}
