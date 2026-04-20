package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var _ advisor.Advisor = (*StatementDisallowTruncateAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
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

type statementDisallowTruncateRule struct{ OmniBaseRule }

func (*statementDisallowTruncateRule) Name() string { return "StatementDisallowTruncateRule" }

func (r *statementDisallowTruncateRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.TruncateStmt)
	if !ok || n.Table == nil {
		return
	}
	// Carveout: MSSQL temp tables (local `#t`, global `##t`) — session-scoped
	// so blast radius is zero.
	if strings.HasPrefix(n.Table.Object, "#") {
		return
	}
	name := n.Table.Object
	if n.Table.Schema != "" {
		name = n.Table.Schema + "." + n.Table.Object
	}
	r.AddAdvice(&storepb.Advice{
		Status: r.Level,
		Code:   code.StatementDisallowTruncate.Int32(),
		Title:  r.Title,
		Content: fmt.Sprintf(
			`TRUNCATE TABLE %q is not allowed: TRUNCATE is minimally logged, bypasses DELETE triggers, and requires ALTER TABLE permission. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
			name,
		),
		StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
	})
}
