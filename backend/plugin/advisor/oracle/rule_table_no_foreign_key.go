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
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow
// foreign key. The judgment lives in omni
// (oracle/v1review.TableNoForeignKey); this adapter renders the findings as
// advice.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	nodes, stmts := tableRuleStatements(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range oraclev1review.TableNoForeignKey(nodes, checkCtx.CurrentDatabase) {
		if f.Code != v1review.TableForeignKey {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.TableHasFK.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeIdentifierName(f.Name)),
			StartPosition: tableRuleFindingPosition(stmts, f),
		})
	}
	return adviceList, nil
}
