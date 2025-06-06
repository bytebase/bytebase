package v1

import (
	"context"
	"maps"
	"slices"

	"github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func CalculateRiskLevelWithSummaryReport(
	_ context.Context,
	risks []*store.RiskMessage,
	commonArgs map[string]any,
	riskSource store.RiskSource,
	summaryReport *storepb.PlanCheckRunResult_Result_SqlSummaryReport,
) (int32, error) {
	if riskSource == store.RiskSourceUnknown {
		return 0, nil
	}

	// Sort by level DESC, higher risks go first.
	slices.SortFunc(risks, func(a, b *store.RiskMessage) int {
		if a.Level > b.Level {
			return -1
		} else if a.Level < b.Level {
			return 1
		}
		return 0
	})

	for _, risk := range risks {
		if !risk.Active {
			continue
		}
		if risk.Source != riskSource {
			continue
		}
		if risk.Expression == nil || risk.Expression.Expression == "" {
			continue
		}
		e, err := cel.NewEnv(common.RiskFactors...)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to create cel environment")
		}
		ast, issues := e.Parse(risk.Expression.Expression)
		if issues != nil && issues.Err() != nil {
			return 0, errors.Errorf("failed to parse expression: %v", issues.Err())
		}
		prg, err := e.Program(ast, cel.EvalOptions(cel.OptPartialEval))
		if err != nil {
			return 0, errors.Wrap(err, "failed to create program")
		}

		args := map[string]any{}
		maps.Copy(args, commonArgs)
		vars, err := e.PartialVars(args)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get vars")
		}
		out, _, err := prg.Eval(vars)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to eval expression")
		}
		if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
			return risk.Level, nil
		}

		if summaryReport == nil {
			continue
		}
		var tableRows int64
		for _, db := range summaryReport.GetChangedResources().GetDatabases() {
			for _, sc := range db.GetSchemas() {
				for _, tb := range sc.GetTables() {
					tableRows += tb.GetTableRows()
				}
			}
		}
		args["affected_rows"] = summaryReport.AffectedRows
		args["table_rows"] = tableRows
		var tableNames []string
		for _, db := range summaryReport.GetChangedResources().GetDatabases() {
			for _, schema := range db.GetSchemas() {
				for _, table := range schema.GetTables() {
					tableNames = append(tableNames, table.Name)
				}
			}
		}
		for _, statementType := range summaryReport.StatementTypes {
			args["sql_type"] = statementType
			for _, tableName := range tableNames {
				args["table_name"] = tableName
				out, _, err := prg.Eval(args)
				if err != nil {
					return 0, errors.Wrap(err, "failed to eval expression")
				}
				if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
					return risk.Level, nil
				}
			}
		}
	}

	return 0, nil
}
