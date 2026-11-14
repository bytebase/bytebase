package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableRequirePkAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePkAdvisor{})
}

// TableRequirePkAdvisor is the advisor checking for table require primary
// key. The judgment lives in omni (mssql/v1review.TableRequirePK); this
// adapter renders the findings as advice.
type TableRequirePkAdvisor struct {
}

// Check checks for table require primary key..
func (*TableRequirePkAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := tableStateStatements(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mssqlv1review.TableRequirePK(nodes) {
		if f.Code != v1review.TablePrimaryKeyMissing {
			continue
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.TableNoPK.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", f.Name),
			StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
		})
	}
	return adviceList, nil
}

// tableRefText returns a human-readable string for a TableRef.
func tableRefText(ref *ast.TableRef) string {
	if ref == nil {
		return ""
	}
	result := ref.Object
	if ref.Schema != "" {
		result = ref.Schema + "." + result
	}
	if ref.Database != "" {
		result = ref.Database + "." + result
	}
	return result
}
