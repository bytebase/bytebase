package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementDMLDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDMLDryRunAdvisor{})
}

// StatementDMLDryRunAdvisor is the advisor checking for DML dry run.
type StatementDMLDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDMLDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Only run EXPLAIN queries if we have a database connection
	if checkCtx.Driver == nil {
		return nil, nil
	}

	// BYT-8855: Skip DML dry run if there are DDL statements mixed in, because DML
	// statements often reference objects created by DDL statements, causing false positives.
	if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
		return nil, nil
	}

	rule := &statementDMLDryRunRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		ctx:        ctx,
		driver:     checkCtx.Driver,
		tenantMode: checkCtx.TenantMode,
	}

	return RunRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDMLDryRunRule struct {
	OmniBaseRule
	driver       *sql.DB
	ctx          context.Context
	explainCount int
	settings     sessionSettings
	tenantMode   bool
}

// Name returns the rule name.
func (*statementDMLDryRunRule) Name() string {
	return "statement.dml-dry-run"
}

func (r *statementDMLDryRunRule) OnStatement(node ast.Node) {
	r.settings.add(node, r.TrimmedStmtText())
	node, text := pgparser.UnwrapExplainAnalyze(node, r.StmtText)
	switch n := node.(type) {
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
		r.checkDMLDryRun(text)
	case *ast.SelectStmt:
		if hasDataModifyingCTE(n.WithClause) {
			r.checkDMLDryRun(text)
		}
	default:
	}
}

func (r *statementDMLDryRunRule) checkDMLDryRun(text string) {
	// Check if we've hit the maximum number of EXPLAIN queries
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}

	r.explainCount++

	statementText := strings.TrimRight(strings.TrimSpace(text), ";")

	// Run EXPLAIN to perform dry run
	_, err := advisor.Query(r.ctx, advisor.QueryContext{
		TenantMode:    r.tenantMode,
		PreExecutions: r.settings.statements(),
	}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", statementText))

	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.StatementDMLDryRunFailed.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", statementText, err.Error()),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
