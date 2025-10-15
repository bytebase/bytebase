package v1

import (
	"context"
	"maps"
	"slices"

	"github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func CalculateRiskLevelWithOptionalSummaryReport(
	_ context.Context,
	risks []*store.RiskMessage,
	commonArgs map[string]any,
	riskSource store.RiskSource,
	summaryReport *storepb.PlanCheckRunResult_Result_SqlSummaryReport,
) (storepb.RiskLevel, error) {
	if riskSource == store.RiskSourceUnknown {
		return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, nil
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
			return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to create cel environment")
		}
		ast, issues := e.Parse(risk.Expression.Expression)
		if issues != nil && issues.Err() != nil {
			return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Errorf("failed to parse expression: %v", issues.Err())
		}
		prg, err := e.Program(ast, cel.EvalOptions(cel.OptPartialEval))
		if err != nil {
			return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Wrap(err, "failed to create program")
		}

		args := map[string]any{}
		maps.Copy(args, commonArgs)
		vars, err := e.PartialVars(args)
		if err != nil {
			return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to get vars")
		}
		out, _, err := prg.Eval(vars)
		if err != nil {
			return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to eval expression")
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
		args[common.CELAttributeStatementAffectedRows] = summaryReport.AffectedRows
		args[common.CELAttributeStatementTableRows] = tableRows
		var tableNames []string
		for _, db := range summaryReport.GetChangedResources().GetDatabases() {
			for _, schema := range db.GetSchemas() {
				for _, table := range schema.GetTables() {
					tableNames = append(tableNames, table.Name)
				}
			}
		}
		for _, statementType := range summaryReport.StatementTypes {
			args[common.CELAttributeStatementSQLType] = statementType
			for _, tableName := range tableNames {
				args[common.CELAttributeResourceTableName] = tableName
				out, _, err := prg.Eval(args)
				if err != nil {
					return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Wrap(err, "failed to eval expression")
				}
				if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
					return risk.Level, nil
				}
			}
		}
	}

	return storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, nil
}
