package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
// The judgment lives in omni (pg/v1review.ColumnRequired); this adapter
// passes the required columns and renders one advice per anchor.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		findings := pgv1review.ColumnRequired([]ast.Node{nodes[i]}, stringArrayPayload.List)
		for j := 0; j < len(findings); {
			f := findings[j]
			var missingColumns []string
			for ; j < len(findings) && findings[j].Start == f.Start && findings[j].End == f.End; j++ {
				missingColumns = append(missingColumns, findings[j].Name)
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.NoRequiredColumn.Int32(),
				Title:   checkCtx.Rule.Type.String(),
				Content: fmt.Sprintf("Table %q requires columns: %s", f.Table, strings.Join(missingColumns, ", ")),
				StartPosition: &storepb.Position{
					Line:   findingLine(stmt, f),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
