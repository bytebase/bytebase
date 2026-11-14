package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor is the advisor checking for collation allowlist.
// The judgment lives in omni (mysql/v1review.SystemCollationAllowlist); this
// adapter passes the rule's payload and renders the findings as advice.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	allowList := stringArrayPayload.List

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mysqlv1review.SystemCollationAllowlist([]ast.Node{node}, allowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisabledCollation.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" used disabled collation '%s'", strings.TrimSpace(stmt.Text), f.Name),
				StartPosition: findingPositionBaseLineTwice(stmt, f),
			})
		}
	}
	return adviceList, nil
}
