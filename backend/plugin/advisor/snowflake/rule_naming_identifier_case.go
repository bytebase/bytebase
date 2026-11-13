// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
// The judgment lives in omni (snowflake/v1review.NamingIdentifierCase);
// this adapter renders the findings as advice.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// The Snowflake rule requires upper case whatever the payload says, and
	// accepts a missing payload: omni ignores this config.
	cfg := v1review.CaseConfig{Upper: checkCtx.Rule.GetNamingCasePayload().GetUpper()}

	title := checkCtx.Rule.Type.String()
	adviceList := []*storepb.Advice{}
	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range snowflakev1review.NamingIdentifierCase([]snowflakeast.Node{node}, cfg) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NamingCaseMismatch.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Identifier %q should be upper case", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
