package tidb

import (
	"context"
	"fmt"
	"strings"

	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
// The judgment lives in omni (tidb/v1review.ColumnRequired); this adapter
// passes the required columns and renders one advice per table.
type ColumnRequirementAdvisor struct{}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	findings := tidbv1review.ColumnRequired(stmtNodes(stmts), stringArrayPayload.List)
	var adviceList []*storepb.Advice
	for i := 0; i < len(findings); {
		f := findings[i]
		var missingColumns []string
		for ; i < len(findings) && findings[i].Table == f.Table; i++ {
			missingColumns = append(missingColumns, findings[i].Name)
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.NoRequiredColumn.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       fmt.Sprintf("Table `%s` requires columns: %s", f.Table, strings.Join(missingColumns, ", ")),
			StartPosition: base.ConvertANTLRLineToPosition(stmts[f.Statement].FirstTokenLine()),
		})
	}
	return adviceList, nil
}
