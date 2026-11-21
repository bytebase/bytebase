package pg

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func pgInsertRowLimit(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return pgRowLimit(
		e,
		pgv1review.StatementInsertRowLimitProbes,
		getInsertedRows,
		pgv1review.StatementInsertRowLimit,
		code.InsertTooManyRows,
		func(text, count string, maxRow int) string {
			return fmt.Sprintf("The statement \"%s\" inserts %s rows. The count exceeds %d.", text, count, maxRow)
		},
	)
}

func pgAffectedRowLimit(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return pgRowLimit(
		e,
		pgv1review.StatementAffectedRowLimitProbes,
		getAffectedRows,
		pgv1review.StatementAffectedRowLimit,
		code.StatementAffectedRowExceedsLimit,
		func(text, count string, maxRow int) string {
			return fmt.Sprintf("The statement \"%s\" affected %s rows (estimated). The count exceeds %d.", text, count, maxRow)
		},
	)
}

func pgRowLimit(
	e *advisor.V1Engine[ast.Node],
	probesFn func([]ast.Node) []int,
	countFn func(db.ProbeResult) (int64, error),
	judgeFn func([]ast.Node, map[int]v1review.Probe, int) []v1review.Finding,
	exceededCode code.Code,
	exceededContent func(text, count string, maxRow int) string,
) advisor.Func {
	return e.RowLimit(advisor.V1RowLimitConfig[ast.Node]{
		Probes:          probesFn,
		Judge:           judgeFn,
		ExceededCode:    exceededCode,
		ExceededContent: exceededContent,
		PositiveOnly:    true,
		StatementText: func(stmt advisor.V1Stmt[ast.Node], node ast.Node) string {
			return probeText(e, stmt, node)
		},
		Estimate: func(ctx context.Context, checkCtx advisor.Context, stmt advisor.V1Stmt[ast.Node], node ast.Node, settings []string) (v1review.Probe, error) {
			return explainRows(ctx, checkCtx, settings, probeText(e, stmt, node), countFn)
		},
		PreExecutions: probeEach,
	})
}

func pgDMLDryRun(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return e.DMLDryRun(advisor.V1DMLDryRunConfig[ast.Node]{
		Probes: pgv1review.StatementDMLDryRunProbes,
		Judge:  pgv1review.StatementDMLDryRun,
		Probe: func(ctx context.Context, prober db.Prober, stmt advisor.V1Stmt[ast.Node], settings []string) v1review.Probe {
			if _, err := prober.Probe(ctx, db.ProbeContext{PreExecutions: settings}, fmt.Sprintf("EXPLAIN %s", probeText(e, stmt, stmt.Node))); err != nil {
				return v1review.Probe{Error: err.Error()}
			}
			return v1review.Probe{}
		},
		FormatContent: func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("\"%s\" dry runs failed: %s", probeText(e, stmt, stmt.Node), f.Value)
		},
		PreExecutions: probeEach,
	})
}

func pgObjectOwnerCheck(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		currentRole := checkCtx.DBSchema.Owner
		if !checkCtx.TenantMode {
			prober, probeErr := advisor.ProbeFor(checkCtx)
			if probeErr != nil {
				return nil, probeErr
			}
			currentRole, err = getCurrentUser(ctx, prober)
			if err != nil {
				slog.Debug("Failed to get current user", log.BBError(err))
				currentRole = checkCtx.DBSchema.Owner
			}
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		schema := e.SchemaOf(checkCtx)
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range pgv1review.StatementObjectOwnerCheck(nodes[i:i+1:i+1], schema, currentRole) {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("%s \"%s\" is owned by \"%s\", but the current role is \"%s\".", f.Name, f.Table, f.Value, currentRole),
					Code:          code.StatementObjectOwnerCheck.Int32(),
					StartPosition: e.PosContentStart(stmt, f),
				})
			}
		}
		return adviceList, nil
	}
}

func getCurrentUser(ctx context.Context, prober db.Prober) (string, error) {
	res, err := prober.Probe(ctx, db.ProbeContext{}, "SELECT CURRENT_USER")
	if err != nil {
		return "", err
	}
	if len(res.Rows) != 1 || len(res.Rows[0]) != 1 {
		return "", errors.Errorf("expected one row with one column but got %d rows", len(res.Rows))
	}
	user, ok := res.Rows[0][0].(string)
	if !ok {
		return "", errors.Errorf("expected a string but got %T", res.Rows[0][0])
	}
	return user, nil
}

func probeText(e *advisor.V1Engine[ast.Node], stmt advisor.V1Stmt[ast.Node], node ast.Node) string {
	_, text := pganalysis.UnwrapExplainAnalyze(node, stmt.Text)
	return e.TrimmedText(text)
}

func probeEach(stmts []advisor.V1Stmt[ast.Node], nodes []ast.Node, want []int, probe func(i int, settings []string)) {
	wanted := make(map[int]bool, len(want))
	for _, i := range want {
		wanted[i] = true
	}
	var settings sessionSettings
	for i, node := range nodes {
		settings.add(node, strings.TrimRight(strings.TrimSpace(stmts[i].Text), ";"))
		if wanted[i] {
			probe(i, settings.statements())
		}
	}
}

func explainRows(ctx context.Context, checkCtx advisor.Context, settings []string, text string, count func(db.ProbeResult) (int64, error)) (v1review.Probe, error) {
	prober, err := advisor.ProbeFor(checkCtx)
	if err != nil {
		return v1review.Probe{}, err
	}
	res, err := prober.Probe(ctx, db.ProbeContext{PreExecutions: settings}, fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", text))
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

func getAffectedRows(res db.ProbeResult) (int64, error) {
	plan, err := res.Plan()
	if err != nil {
		return 0, err
	}
	return pganalysis.EstimatedAffectedRowsFromExplainJSON(plan)
}

func getInsertedRows(res db.ProbeResult) (int64, error) {
	plan, err := res.Plan()
	if err != nil {
		return 0, err
	}
	return pganalysis.EstimatedInsertedRowsFromExplainJSON(plan)
}

type sessionSettings struct {
	settings         []sessionSetting
	transactionStart int
	inTransaction    bool
}

type sessionSetting struct {
	name  string
	text  string
	local bool
}

func (s *sessionSettings) add(node ast.Node, text string) {
	switch n := node.(type) {
	case *ast.VariableSetStmt:
		if isRoleOrSearchPathSet(n) {
			if n.Kind == ast.VAR_SET_CURRENT && !n.IsLocal {
				for i := range s.settings {
					if strings.EqualFold(s.settings[i].name, n.Name) {
						s.settings[i].local = false
					}
				}
			}
			s.settings = append(s.settings, sessionSetting{name: n.Name, text: text, local: n.IsLocal})
		}
	case *ast.DiscardStmt:
		if n.Target == ast.DISCARD_ALL {
			s.settings, s.transactionStart = nil, 0
		}
	case *ast.TransactionStmt:
		switch n.Kind {
		case ast.TRANS_STMT_BEGIN, ast.TRANS_STMT_START:
			if !s.inTransaction {
				s.transactionStart, s.inTransaction = len(s.settings), true
			}
		case ast.TRANS_STMT_COMMIT, ast.TRANS_STMT_ROLLBACK:
			if n.Kind == ast.TRANS_STMT_ROLLBACK && s.inTransaction {
				s.settings = s.settings[:s.transactionStart]
			}
			s.settings = slices.DeleteFunc(s.settings, func(setting sessionSetting) bool { return setting.local })
			s.transactionStart, s.inTransaction = len(s.settings), n.Chain
		default:
		}
	default:
	}
}

func (s *sessionSettings) statements() []string {
	var statements []string
	for _, setting := range s.settings {
		statements = append(statements, setting.text)
	}
	return statements
}

func isRoleOrSearchPathSet(stmt *ast.VariableSetStmt) bool {
	if stmt == nil {
		return false
	}
	name := strings.ToLower(stmt.Name)
	return name == "role" || name == "search_path" || stmt.Kind == ast.VAR_RESET_ALL
}
