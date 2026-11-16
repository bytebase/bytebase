// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*TableRequirePkAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePkAdvisor{})
}

// TableRequirePkAdvisor is the advisor checking for table require primary
// key. The judgment lives in omni (snowflake/v1review.TableRequirePK); this
// adapter renders the findings as advice.
type TableRequirePkAdvisor struct {
}

// Check checks for table require primary key.
func (*TableRequirePkAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := tableStateStatements(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range snowflakev1review.TableRequirePK(nodes) {
		if f.Code != v1review.TablePrimaryKeyMissing {
			continue
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.TableNoPK.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
		})
	}
	return adviceList, nil
}
