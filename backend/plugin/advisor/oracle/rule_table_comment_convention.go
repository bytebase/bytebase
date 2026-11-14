// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

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
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment
// convention. The judgment lives in omni (oracle/v1review.TableComment);
// this adapter converts the rule's payload into the omni config and renders
// the findings as advice.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	// A nil cfg makes omni panic at the first created table, as V1 did when
	// the rule had no payload.
	var cfg *v1review.TableCommentConfig
	if payload := checkCtx.Rule.GetCommentConventionPayload(); payload != nil {
		cfg = &v1review.TableCommentConfig{Required: payload.Required, MaxLength: int(payload.MaxLength)}
	}

	nodes, stmts := tableRuleStatements(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range oraclev1review.TableComment(nodes, checkCtx.CurrentDatabase, cfg) {
		advice := &storepb.Advice{
			Status:        level,
			Title:         title,
			StartPosition: tableRuleFindingPosition(stmts, f),
		}
		switch f.Code {
		case v1review.TableCommentMissing:
			advice.Code = code.CommentEmpty.Int32()
			advice.Content = fmt.Sprintf("Comment is required for table %s", normalizeIdentifierName(f.Name))
		case v1review.TableCommentTooLong:
			advice.Code = code.CommentTooLong.Int32()
			advice.Content = fmt.Sprintf("Table %s comment is too long. The length of comment should be within %d characters", normalizeIdentifierName(f.Name), cfg.MaxLength)
		default:
			continue
		}
		adviceList = append(adviceList, advice)
	}
	return adviceList, nil
}

// tableRuleStatements returns the statements a table rule judging all
// statements together is given, with the parsed statement of each.
func tableRuleStatements(parsed []base.ParsedStatement) ([]ast.Node, []base.ParsedStatement) {
	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}
	return nodes, stmts
}

// tableRuleFindingPosition is the position of a finding from a table rule
// judging all statements together: the line of its range within the
// statement it names.
func tableRuleFindingPosition(stmts []base.ParsedStatement, f v1review.Finding) *storepb.Position {
	stmt := stmts[f.Statement]
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f))
}
