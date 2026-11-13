// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*ColumnAddNotNullColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT, &ColumnAddNotNullColumnRequireDefaultAdvisor{})
}

// ColumnAddNotNullColumnRequireDefaultAdvisor is the advisor checking for adding not null column requires default.
// The judgment lives in omni (oracle/v1review.ColumnAddNotNullRequireDefault);
// this adapter renders the findings as advice.
type ColumnAddNotNullColumnRequireDefaultAdvisor struct {
}

// Check checks for adding not null column requires default.
func (*ColumnAddNotNullColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oraclev1review.ColumnAddNotNullRequireDefault([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NotNullColumnWithNoDefault.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Adding not null column %q requires default.", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
