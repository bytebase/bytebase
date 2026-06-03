package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
}

type StatementDmlDryRunAdvisor struct {
}

func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// BYT-8855: Skip DML dry run if there are DDL statements mixed in, because DML
	// statements often reference objects created by DDL statements, causing false positives.
	if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
		return nil, nil
	}

	rule := NewStatementDmlDryRunRule(ctx, level, checkCtx.Rule.Type.String(), checkCtx.Driver)

	if checkCtx.Driver != nil {
		return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
	}

	return rule.GetAdviceList()
}

// StatementDmlDryRunRule is the rule implementation for DML dry run checks.
type StatementDmlDryRunRule struct {
	BaseRule

	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

// NewStatementDmlDryRunRule creates a new StatementDmlDryRunRule.
func NewStatementDmlDryRunRule(ctx context.Context, level storepb.Advice_Status, title string, driver *sql.DB) *StatementDmlDryRunRule {
	return &StatementDmlDryRunRule{
		BaseRule: NewBaseRule(level, title, 0),
		driver:   driver,
		ctx:      ctx,
	}
}

// Name returns the rule name.
func (*StatementDmlDryRunRule) Name() string {
	return "statement.dml-dry-run"
}

// OnStatement dry-runs top-level DML statements from the omni AST.
func (r *StatementDmlDryRunRule) OnStatement(node ast.Node) {
	switch node.(type) {
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
		r.handleStmt(r.stmtText, r.baseLine+1)
	default:
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.

func (r *StatementDmlDryRunRule) handleStmt(text string, lineNumber int) {
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}
	r.explainCount++
	if _, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_ORACLE, fmt.Sprintf("EXPLAIN PLAN FOR %s", text)); err != nil {
		r.AddAdvice(
			r.level,
			code.StatementDMLDryRunFailed.Int32(),
			fmt.Sprintf("Failed to dry run statement at line %d: %v", lineNumber, err),
			common.ConvertANTLRLineToPosition(lineNumber),
		)
	}
}
