package mssql

import (
	"context"
	"slices"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for disallowed types for column.
type ColumnTypeDisallowListAdvisor struct {
}

func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column type disallow list rule")
	}

	disallowTypes := make([]string, 0, len(stringArrayPayload.List))
	for _, tp := range stringArrayPayload.List {
		disallowTypes = append(disallowTypes, strings.ToUpper(tp))
	}

	rule := &columnTypeDisallowListRule{
		OmniBaseRule:  OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		disallowTypes: disallowTypes,
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnTypeDisallowListRule struct {
	OmniBaseRule
	disallowTypes []string
}

func (*columnTypeDisallowListRule) Name() string {
	return "ColumnTypeDisallowListRule"
}

func (r *columnTypeDisallowListRule) OnStatement(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		dt, ok := n.(*ast.DataType)
		if !ok || dt == nil {
			return true
		}
		formattedDataType := strings.ToUpper(dt.Name)
		if slices.Contains(r.disallowTypes, formattedDataType) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.DisabledColumnType.Int32(),
				Title:         r.Title,
				Content:       "Column type " + formattedDataType + " is disallowed",
				StartPosition: &storepb.Position{Line: r.LocToLine(dt.Loc)},
			})
		}
		return true
	})
}
