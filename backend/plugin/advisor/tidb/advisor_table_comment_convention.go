package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor checks the comment convention for tables.
// The judgment lives in omni (tidb/v1review.TableComment); this adapter
// converts the rule's payload into the omni config and renders the findings
// as advice.
type TableCommentConventionAdvisor struct {
}

// Check checks for table comment convention.
func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	// A nil cfg makes omni panic at the first judged statement, as V1 did
	// when the rule had no payload.
	var cfg *v1review.TableCommentConfig
	if payload := checkCtx.Rule.GetCommentConventionPayload(); payload != nil {
		cfg = &v1review.TableCommentConfig{Required: payload.Required, MaxLength: int(payload.MaxLength)}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.TableComment([]ast.Node{stmt.Node}, cfg) {
			advice := &storepb.Advice{
				Status:        level,
				Title:         title,
				StartPosition: base.ConvertANTLRLineToPosition(stmt.FirstTokenLine()),
			}
			switch f.Code {
			case v1review.TableCommentMissing:
				advice.Code = code.CommentEmpty.Int32()
				advice.Content = fmt.Sprintf("Table `%s` requires comments", f.Name)
			case v1review.TableCommentTooLong:
				advice.Code = code.CommentTooLong.Int32()
				advice.Content = fmt.Sprintf("The length of table `%s` comment should be within %d characters", f.Name, cfg.MaxLength)
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
	}
	return adviceList, nil
}
