package tsql

import (
	"context"
	"slices"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MSSQL, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
// Runs on the omni AST extractor.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, schema string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	q := newOmniQuerySpanExtractor(database, schema, gCtx, ignoreCaseSensitive)
	span, err := q.getOmniQuerySpan(ctx, stmt.Text)
	if err != nil {
		return nil, err
	}
	if gCtx.PlanOutput != nil {
		markPlanOutput(span, q.setOption, gCtx.PlanOutput)
	}
	return span, nil
}

var (
	// showplanOptions plan a statement instead of running it.
	showplanOptions = []string{"SHOWPLAN_ALL", "SHOWPLAN_TEXT", "SHOWPLAN_XML"}
	// statisticsOptions run a statement and return its plan as well.
	statisticsOptions = []string{"STATISTICS XML", "STATISTICS PROFILE"}
)

// markPlanOutput records what a SET statement turns on or off in output and
// marks the statements a plan comes back for, since reading a plan needs
// bb.sql.explain. While a SHOWPLAN option is on a read is an Explain; while a
// STATISTICS option is on a statement is marked like EXPLAIN ANALYZE. A SET
// that names one of these options is marked the same way, and any other SET
// follows the batch's options like every other statement.
func markPlanOutput(span *base.QuerySpan, set *ast.SetOptionStmt, output base.PlanOutput) {
	if set != nil {
		// Anything but OFF counts as on, so an unexpected form costs a check
		// rather than skipping one.
		on := true
		if value, ok := set.Value.(*ast.ColumnRef); ok && strings.EqualFold(value.Column, "OFF") {
			on = false
		}
		for _, option := range set.Options {
			name := strings.ToUpper(option)
			switch {
			case slices.Contains(showplanOptions, name):
				output[name] = on
				span.Type = base.Explain
			case slices.Contains(statisticsOptions, name):
				output[name] = on
				span.ExplainAnalyze = true
			default:
			}
		}
	}
	isOn := func(name string) bool { return output[name] }
	if slices.ContainsFunc(showplanOptions, isOn) && span.Type == base.Select {
		span.Type = base.Explain
	}
	if slices.ContainsFunc(statisticsOptions, isOn) {
		span.ExplainAnalyze = true
	}
}
