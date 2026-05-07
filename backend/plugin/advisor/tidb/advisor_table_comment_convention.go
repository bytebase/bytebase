package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor checks the comment convention for tables.
type TableCommentConventionAdvisor struct {
}

// Check is Recipe A. Only matches *ast.CreateTableStmt at top level —
// CREATE TABLE doesn't nest, and EXPLAIN-DDL is invalid grammar in both
// pingcap and omni.
func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload := checkCtx.Rule.GetCommentConventionPayload()
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		n, ok := ostmt.Node.(*ast.CreateTableStmt)
		if !ok || n.Table == nil {
			continue
		}
		// omni TableOption.Value is the literal comment string (no quotes,
		// no ESCAPE wrapper); pingcap stored it the same way after restore.
		// The "exists" signal is the option's presence in the slice, not a
		// non-empty value (a deliberate empty COMMENT '' is still present).
		comment, exists := omniTableOptionPresent(n.Options, omniOptionNamesComment)
		line := ostmt.FirstTokenLine()
		if payload.Required && !exists {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.CommentEmpty.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table `%s` requires comments", n.Table.Name),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
		}
		if payload.MaxLength >= 0 && int32(len(comment)) > payload.MaxLength {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.CommentTooLong.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The length of table `%s` comment should be within %d characters", n.Table.Name, payload.MaxLength),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
		}
	}
	return adviceList, nil
}
