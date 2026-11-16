// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK. The
// judgment lives in omni (oracle/v1review.TableRequirePK); this adapter
// renders the findings as advice.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	nodes, stmts := tableRuleStatements(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range oraclev1review.TableRequirePK(nodes, checkCtx.CurrentDatabase) {
		if f.Code != v1review.TablePrimaryKeyMissing {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.TableNoPK.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", normalizeIdentifierName(f.Name)),
			StartPosition: tableRuleFindingPosition(stmts, f),
		})
	}
	return adviceList, nil
}
