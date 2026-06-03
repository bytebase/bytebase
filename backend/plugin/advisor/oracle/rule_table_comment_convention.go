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
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	commentPayload := checkCtx.Rule.GetCommentConventionPayload()

	rule := NewTableCommentConventionRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, commentPayload)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// TableCommentConventionRule is the rule implementation for table comment convention.
type TableCommentConventionRule struct {
	BaseRule

	currentDatabase string
	payload         *storepb.SQLReviewRule_CommentConventionRulePayload

	tableNames   []string
	tableComment map[string]string
	tableLine    map[string]int
}

// NewTableCommentConventionRule creates a new TableCommentConventionRule.
func NewTableCommentConventionRule(level storepb.Advice_Status, title string, currentDatabase string, payload *storepb.SQLReviewRule_CommentConventionRulePayload) *TableCommentConventionRule {
	return &TableCommentConventionRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		payload:         payload,
		tableNames:      []string{},
		tableComment:    make(map[string]string),
		tableLine:       make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableCommentConventionRule) Name() string {
	return "table.comment-convention"
}

// OnStatement records table creation and COMMENT ON TABLE statements from omni.
func (r *TableCommentConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		r.tableNames = append(r.tableNames, tableName)
		r.tableLine[tableName] = r.locLine(n.Loc)
	case *ast.CommentStmt:
		if n.ObjectType != ast.OBJECT_TABLE {
			return
		}
		r.tableComment[omniObjectName(n.Object, r.currentDatabase)] = n.Comment
	default:
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.

// GetAdviceList returns the advice list.
func (r *TableCommentConventionRule) GetAdviceList() ([]*storepb.Advice, error) {
	for _, tableName := range r.tableNames {
		comment, ok := r.tableComment[tableName]
		if !ok || comment == "" {
			if r.payload.Required {
				r.AddAdvice(
					r.level,
					code.CommentEmpty.Int32(),
					fmt.Sprintf("Comment is required for table %s", normalizeIdentifierName(tableName)),
					common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
				)
			}
		} else {
			if r.payload.MaxLength > 0 && int32(len(comment)) > r.payload.MaxLength {
				r.AddAdvice(
					r.level,
					code.CommentTooLong.Int32(),
					fmt.Sprintf("Table %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(tableName), r.payload.MaxLength),
					common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
				)
			}
		}
	}

	return r.BaseRule.GetAdviceList()
}
