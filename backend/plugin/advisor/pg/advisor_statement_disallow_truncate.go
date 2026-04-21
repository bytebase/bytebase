package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var _ advisor.Advisor = (*StatementDisallowTruncateAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
}

type StatementDisallowTruncateAdvisor struct{}

func (*StatementDisallowTruncateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &statementDisallowTruncateRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowTruncateRule struct {
	OmniBaseRule
}

func (*statementDisallowTruncateRule) Name() string { return "statement_disallow_truncate" }

func (r *statementDisallowTruncateRule) OnStatement(node ast.Node) {
	stmt, ok := node.(*ast.TruncateStmt)
	if !ok || stmt.Relations == nil {
		return
	}
	cascade := stmt.Behavior == ast.DROP_CASCADE
	restartSeqs := stmt.RestartSeqs
	for _, item := range stmt.Relations.Items {
		rv, ok := item.(*ast.RangeVar)
		if !ok {
			continue
		}
		name := rv.Relname
		if rv.Schemaname != "" {
			name = rv.Schemaname + "." + rv.Relname
		}
		suffix := ""
		if cascade {
			suffix += " CASCADE will wipe dependent tables."
		}
		if restartSeqs {
			suffix += " RESTART IDENTITY will reset owned sequences."
		}
		// Point each advice at its own relation in a multi-relation form,
		// not at the TRUNCATE keyword. Loc falls back to the statement
		// start when the parser did not record a byte offset (LocToLine's
		// defined fallback), which preserves the single-relation case.
		r.AddAdvice(&storepb.Advice{
			Status: r.Level,
			Code:   code.StatementDisallowTruncate.Int32(),
			Title:  r.Title,
			Content: fmt.Sprintf(
				`TRUNCATE TABLE %q is not allowed: TRUNCATE bypasses triggers. Unlike Oracle/MySQL/MSSQL, PG TRUNCATE is transactional and rolls back cleanly.%s`,
				name, suffix,
			),
			StartPosition: &storepb.Position{Line: r.LocToLine(rv.Loc)},
		})
	}
}
