package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
}

// StatementDisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE statement.
type StatementDisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE statement.
func (*StatementDisallowLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		code := advisorcode.Ok
		switch n := ostmt.Node.(type) {
		case *ast.UpdateStmt:
			if n.Limit != nil {
				code = advisorcode.UpdateUseLimit
			}
		case *ast.DeleteStmt:
			if n.Limit != nil {
				code = advisorcode.DeleteUseLimit
			}
		case *ast.InsertStmt:
			if insertUsesLimit(n) {
				code = advisorcode.InsertUseLimit
			}
		default:
		}

		if code != advisorcode.Ok {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", ostmt.TrimmedText()),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}

	return adviceList, nil
}

// insertUsesLimit reports whether an INSERT ... SELECT carries a LIMIT
// on its inner SELECT. omni unifies the SELECT and UNION/INTERSECT/EXCEPT
// forms under a single SelectStmt with a SetOp field, so checking
// SelectStmt.Limit covers both pingcap's *SelectStmt and *SetOprStmt cases.
//
// Only the outermost LIMIT is checked. A LIMIT nested inside a
// parenthesized UNION arm (e.g.
// "INSERT INTO t (SELECT a FROM x LIMIT 3) UNION SELECT b FROM y") lives
// on Left.Limit, not on the outer .Limit; this matches pingcap parity.
func insertUsesLimit(n *ast.InsertStmt) bool {
	return n.Select != nil && n.Select.Limit != nil
}
