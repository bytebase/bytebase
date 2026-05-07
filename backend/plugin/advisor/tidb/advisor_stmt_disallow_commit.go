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
	_ advisor.Advisor = (*StatementDisallowCommitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
}

// StatementDisallowCommitAdvisor is the advisor checking for disallow COMMIT.
type StatementDisallowCommitAdvisor struct {
}

// Check checks for disallow COMMIT.
func (*StatementDisallowCommitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		if _, ok := ostmt.Node.(*ast.CommitStmt); !ok {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementDisallowCommit.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", ostmt.TrimmedText()),
			StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
		})
	}

	return adviceList, nil
}
