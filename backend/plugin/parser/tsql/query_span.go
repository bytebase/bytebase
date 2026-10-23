package tsql

import (
	"context"
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

// markPlanOutput records what a SET statement turns on in output and marks the
// statements a plan comes back for, since reading a plan needs bb.sql.explain.
// Under SHOWPLAN a read is planned instead of run, so it is an Explain. Under
// STATISTICS XML or PROFILE a statement runs and returns its plan too, like
// EXPLAIN ANALYZE. A SET that names either option is marked the same way.
func markPlanOutput(span *base.QuerySpan, set *ast.SetOptionStmt, output *base.PlanOutput) {
	if set != nil {
		on := false
		if value, ok := set.Value.(*ast.ColumnRef); ok {
			on = strings.EqualFold(value.Column, "ON")
		}
		for _, option := range set.Options {
			switch strings.ToUpper(option) {
			case "SHOWPLAN_ALL", "SHOWPLAN_TEXT", "SHOWPLAN_XML":
				output.Showplan = on
				span.Type = base.Explain
			case "STATISTICS XML", "STATISTICS PROFILE":
				output.Statistics = on
				span.ExplainAnalyze = true
			default:
			}
		}
		return
	}
	if output.Showplan && span.Type == base.Select {
		span.Type = base.Explain
	}
	if output.Statistics {
		span.ExplainAnalyze = true
	}
}
