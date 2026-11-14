package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*IndexTypeAllowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST, &IndexTypeAllowListAdvisor{})
}

// IndexTypeAllowListAdvisor is the advisor checking for index types. The
// judgment lives in omni (mysql/v1review.IndexTypeAllowList); this adapter
// converts the rule's payload and renders the findings as advice.
type IndexTypeAllowListAdvisor struct {
}

func (*IndexTypeAllowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// A rule without the payload panics here, which advisor.Check reports
	// as the rule's error.
	allowList := checkCtx.Rule.GetStringArrayPayload().List

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
		for _, f := range mysqlv1review.IndexTypeAllowList([]ast.Node{node}, allowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexTypeNotAllowed.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Index type `%s` is not allowed", f.Value),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
