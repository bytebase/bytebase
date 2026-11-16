// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"math"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
// The judgment lives in omni (oracle/v1review.ColumnComment); this adapter
// passes the payload and renders the findings as advice.
type ColumnCommentConventionAdvisor struct {
}

func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload := checkCtx.Rule.GetCommentConventionPayload()
	// Without a payload every column is a finding, so commentMaxLength panics
	// on the first column, as V1 did.
	required, maxLength := true, -1
	if payload != nil {
		required, maxLength = payload.Required, int(payload.MaxLength)
		if maxLength <= 0 {
			maxLength = math.MaxInt
		}
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range oraclev1review.ColumnComment(nodes, checkCtx.CurrentDatabase, required, maxLength) {
		limit := commentMaxLength(payload)
		name := normalizeIdentifierName(f.Table + "." + f.Name)
		adviceCode, content := code.CommentEmpty, fmt.Sprintf("Comment is required for column %s", name)
		if f.Code == v1review.ColumnCommentTooLong {
			adviceCode, content = code.CommentTooLong, fmt.Sprintf("Column %s comment is too long. The length of comment should be within %d characters", name, limit)
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       content,
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
		})
	}
	return adviceList, nil
}

// commentMaxLength reads the payload's limit without a nil check, so a
// missing payload panics where V1's did.
func commentMaxLength(payload *storepb.SQLReviewRule_CommentConventionRulePayload) int32 {
	return payload.MaxLength
}

// parsedNodes returns the statements that carry an omni node and their nodes,
// index for index, for a rule that judges the statements together.
func parsedNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}
