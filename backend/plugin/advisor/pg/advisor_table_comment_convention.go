package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment
// convention. The judgment lives in omni (pg/v1review.TableComment); this
// adapter converts the rule's payload into the omni config and renders the
// findings as advice.
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

	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.TableComment(nodes, cfg) {
		// V1 anchored a finding on the first non-whitespace line of the
		// statement it came from.
		stmt := stmts[f.Statement]
		line := (&BaseRule{StmtText: stmt.Text}).ContentStartLine() + int32(stmt.BaseLine())
		advice := &storepb.Advice{
			Status:        level,
			Title:         title,
			StartPosition: &storepb.Position{Line: line},
		}
		switch f.Code {
		case v1review.TableCommentMissing:
			advice.Code = code.CommentEmpty.Int32()
			advice.Content = fmt.Sprintf("Comment is required for table `%s`", f.Name)
		case v1review.TableCommentTooLong:
			advice.Code = code.CommentTooLong.Int32()
			advice.Content = fmt.Sprintf("Table `%s` comment is too long. The length of comment should be within %d characters", f.Name, cfg.MaxLength)
		default:
			continue
		}
		adviceList = append(adviceList, advice)
	}
	return adviceList, nil
}
