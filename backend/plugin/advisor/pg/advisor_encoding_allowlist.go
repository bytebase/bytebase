package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*EncodingAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &EncodingAllowlistAdvisor{})
}

// EncodingAllowlistAdvisor is the advisor checking for encoding allowlist.
// The judgment lives in omni (pg/v1review.SystemCharsetAllowlist); this
// adapter passes the rule's payload and renders the findings as advice.
type EncodingAllowlistAdvisor struct {
}

// Check checks for encoding allowlist.
func (*EncodingAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range pgv1review.SystemCharsetAllowlist([]ast.Node{node}, allowList) {
			// V1 printed an empty statement and placed the advice on the
			// statement's base line.
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.DisabledCharset.Int32(),
				Title:   title,
				Content: fmt.Sprintf("\"\" used disabled encoding '%s'", f.Name),
				StartPosition: &storepb.Position{
					Line:   int32(stmt.BaseLine()),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
