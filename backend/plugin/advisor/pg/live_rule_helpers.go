package pg

import (
	"context"
	"database/sql"
	"strings"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// liveStatements are the statements that parsed to an omni node, with their
// nodes, index for index.
type liveStatements struct {
	nodes []ast.Node
	stmts []base.ParsedStatement
}

func newLiveStatements(parsed []base.ParsedStatement) liveStatements {
	var l liveStatements
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		l.nodes = append(l.nodes, node)
		l.stmts = append(l.stmts, stmt)
	}
	return l
}

// probeText is the text a live rule EXPLAINs and prints for statement i: the
// statement an EXPLAIN ANALYZE runs, else the statement, without surrounding
// white space and trailing semicolons.
func (l liveStatements) probeText(i int) string {
	_, text := pganalysis.UnwrapExplainAnalyze(l.nodes[i], l.stmts[i].Text)
	return strings.TrimRight(strings.TrimSpace(text), ";")
}

// startLine is statement i's first content line within its own text.
func (l liveStatements) startLine(i int) int32 {
	return (&BaseRule{StmtText: l.stmts[i].Text}).ContentStartLine()
}

// probeEach walks the statements in order, recording each one's session
// settings, and calls probe for the statements in want with the settings
// the statements before them made.
func (l liveStatements) probeEach(want []int, probe func(i int, settings []string)) {
	wanted := make(map[int]bool, len(want))
	for _, i := range want {
		wanted[i] = true
	}
	var settings sessionSettings
	for i, node := range l.nodes {
		settings.add(node, (&BaseRule{StmtText: l.stmts[i].Text}).TrimmedStmtText())
		if wanted[i] {
			probe(i, settings.statements())
		}
	}
}

// explainRows EXPLAINs text with the given session settings and reads its
// row count with count.
func explainRows(ctx context.Context, driver *sql.DB, tenantMode bool, settings []string, text string, count func([]any) (int64, error)) v1review.Probe {
	res, err := advisor.Query(ctx, advisor.QueryContext{
		TenantMode:    tenantMode,
		PreExecutions: settings,
	}, driver, storepb.Engine_POSTGRES, getExplainSQL(text))
	if err != nil {
		return v1review.Probe{Error: err.Error()}
	}
	rows, err := count(res)
	if err != nil {
		return v1review.Probe{CountError: err.Error()}
	}
	return v1review.Probe{Rows: rows}
}
