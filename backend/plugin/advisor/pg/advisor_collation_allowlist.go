package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor is the advisor checking for collation allowlist.
// The judgment lives in omni (pg/v1review.SystemCollationAllowlist); this
// adapter passes the rule's payload and renders the findings as advice.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	allowList := checkCtx.Rule.GetStringArrayPayload().List

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		line := stmt.BaseLine() + int((&BaseRule{StmtText: stmt.Text}).ContentStartLine())
		for _, f := range pgv1review.SystemCollationAllowlist([]ast.Node{node}, allowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisabledCollation.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Use disabled collation \"%s\", related statement \"%s\"", f.Name, strings.TrimSpace(stmt.Text)),
				StartPosition: base.ConvertANTLRLineToPosition(line),
			})
		}
	}
	return adviceList, nil
}
