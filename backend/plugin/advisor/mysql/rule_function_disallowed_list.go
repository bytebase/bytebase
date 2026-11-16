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
	_ advisor.Advisor = (*FunctionDisallowedListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST, &FunctionDisallowedListAdvisor{})
}

// FunctionDisallowedListAdvisor is the advisor checking for disallowed
// function list. The judgment lives in omni
// (mysql/v1review.SystemFunctionDisallowedList); this adapter passes the
// rule's payload and renders the findings as advice.
type FunctionDisallowedListAdvisor struct {
}

func (*FunctionDisallowedListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	disallowList := stringArrayPayload.List

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
		for _, f := range mysqlv1review.SystemFunctionDisallowedList([]ast.Node{node}, disallowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisabledFunction.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", f.Name, strings.TrimSpace(stmt.Text)),
				StartPosition: findingPositionBaseLineTwice(stmt, f),
			})
		}
	}
	return adviceList, nil
}
