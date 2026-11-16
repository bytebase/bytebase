package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

var (
	_ advisor.Advisor = (*StatementDisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
}

// StatementDisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE/DELETE statement.
// The judgment lives in omni (oceanbase/v1review.StatementDisallowLimit); this adapter
// renders its findings as advice.
type StatementDisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE/DELETE statement.
func (*StatementDisallowLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		text := strings.TrimSpace(stmt.Text)
		for _, f := range oceanbasev1review.StatementDisallowLimit([]ast.Node{node}) {
			c := code.InsertUseLimit
			switch f.Name {
			case "UPDATE":
				c = code.UpdateUseLimit
			case "DELETE":
				c = code.DeleteUseLimit
			default:
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          c.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", text),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
