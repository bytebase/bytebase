package pg

import (
	"context"
	"strings"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
	return contentStartLine(l.stmts[i].Text)
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
		settings.add(node, trimmedStmtText(l.stmts[i].Text))
		if wanted[i] {
			probe(i, settings.statements())
		}
	}
}

// explainRows EXPLAINs text with the given session settings and reads its
// row count with count. The row limit rules ask for it only where they must
// estimate, so an absent prober is the statement's problem, not the review's.
func explainRows(ctx context.Context, checkCtx advisor.Context, settings []string, text string, count func(db.ProbeResult) (int64, error)) (v1review.Probe, error) {
	prober, err := advisor.ProbeFor(checkCtx)
	if err != nil {
		return v1review.Probe{}, err
	}
	res, err := prober.Probe(ctx, db.ProbeContext{PreExecutions: settings}, getExplainSQL(text))
	if err != nil {
		//nolint:nilerr // a refused statement is the probe finding, reported as advice.
		return v1review.Probe{Error: err.Error()}, nil
	}
	rows, err := count(res)
	if err != nil {
		//nolint:nilerr // an unreadable plan is the probe finding, reported as advice.
		return v1review.Probe{CountError: err.Error()}, nil
	}
	return v1review.Probe{Rows: rows}, nil
}
