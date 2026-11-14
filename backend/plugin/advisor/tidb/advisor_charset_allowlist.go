package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*CharsetAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &CharsetAllowlistAdvisor{})
}

// CharsetAllowlistAdvisor checks for charset allowlist. The judgment lives in
// omni (tidb/v1review.SystemCharsetAllowlist); this adapter passes the rule's
// payload and renders the findings as advice.
type CharsetAllowlistAdvisor struct {
}

// Check checks for charset allowlist.
func (*CharsetAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	allowList := stringArrayPayload.List

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.SystemCharsetAllowlist([]ast.Node{stmt.Node}, allowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.DisabledCharset.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" used disabled charset '%s'", stmt.TrimmedText(), f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(allowlistFindingLine(stmt, f)),
			})
		}
	}
	return adviceList, nil
}

// allowlistFindingLine is the line the V1 charset and collation rules
// reported: the line a CREATE TABLE column starts on for a finding anchoring
// that column, and the statement's first token, comments included, for a
// finding anchoring the whole statement.
func allowlistFindingLine(stmt Stmt, f v1review.Finding) int {
	if n, ok := stmt.Node.(*ast.CreateTableStmt); ok && f.Start != max(n.Loc.Start, 0) {
		return stmt.AbsoluteLine(f.Start)
	}
	return stmt.FirstTokenLine()
}
