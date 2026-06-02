// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
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

	rule := NewColumnCommentConventionRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, commentPayload)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ColumnCommentConventionRule is the rule implementation for column comment convention.
type ColumnCommentConventionRule struct {
	BaseRule

	currentDatabase string
	payload         *storepb.SQLReviewRule_CommentConventionRulePayload

	columnNames   []string
	columnComment map[string]string
	columnLine    map[string]int
}

// NewColumnCommentConventionRule creates a new ColumnCommentConventionRule.
func NewColumnCommentConventionRule(level storepb.Advice_Status, title string, currentDatabase string, payload *storepb.SQLReviewRule_CommentConventionRulePayload) *ColumnCommentConventionRule {
	return &ColumnCommentConventionRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		payload:         payload,
		columnNames:     []string{},
		columnComment:   make(map[string]string),
		columnLine:      make(map[string]int),
	}
}

// Name returns the rule name.
func (*ColumnCommentConventionRule) Name() string {
	return "column.comment-convention"
}

// OnStatement records column definitions and COMMENT ON COLUMN statements from omni.
func (r *ColumnCommentConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, col := range omniColumnDefs(n.Columns) {
			columnName := fmt.Sprintf("%s.%s", tableName, col.Name)
			r.columnNames = append(r.columnNames, columnName)
			r.columnLine[columnName] = r.locLine(col.Loc)
		}
	case *ast.AlterTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Action != ast.AT_ADD_COLUMN {
				continue
			}
			for _, col := range append(omniColumnDefs(cmd.ColumnDefs), cmd.ColumnDef) {
				if col == nil {
					continue
				}
				columnName := fmt.Sprintf("%s.%s", tableName, col.Name)
				r.columnNames = append(r.columnNames, columnName)
				r.columnLine[columnName] = r.locLine(col.Loc)
			}
		}
	case *ast.CommentStmt:
		if n.ObjectType != ast.OBJECT_TABLE || n.Column == "" {
			return
		}
		columnName := fmt.Sprintf("%s.%s", omniObjectName(n.Object, r.currentDatabase), n.Column)
		r.columnComment[columnName] = n.Comment
	default:
	}
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
			if r.payload.MaxLength > 0 && int32(len(comment)) > r.payload.MaxLength {
				r.AddAdvice(
					r.level,
					code.CommentTooLong.Int32(),
					fmt.Sprintf("Column %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(columnName), r.payload.MaxLength),
					common.ConvertANTLRLineToPosition(r.columnLine[columnName]),
				)
			}
		}
	}
	return r.BaseRule.GetAdviceList()
}
