package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
func (*StatementAffectedRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 || checkCtx.Driver == nil {
		return nil, nil
	}

	rule := &statementAffectedRowLimitRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		maxRow:     int(numberPayload.Number),
		ctx:        ctx,
		driver:     checkCtx.Driver,
		tenantMode: checkCtx.TenantMode,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementAffectedRowLimitRule struct {
	OmniBaseRule
	maxRow        int
	driver        *sql.DB
	ctx           context.Context
	explainCount  int
	preExecutions []string
	tenantMode    bool
}

func (*statementAffectedRowLimitRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT)
}

func (r *statementAffectedRowLimitRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.VariableSetStmt:
		if omniIsRoleOrSearchPathSet(n) {
			r.preExecutions = append(r.preExecutions, r.TrimmedStmtText())
		}
	case *ast.UpdateStmt, *ast.DeleteStmt:
		r.checkAffectedRows()
	default:
	}
}

func (r *statementAffectedRowLimitRule) checkAffectedRows() {
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}

	r.explainCount++

	statementText := r.TrimmedStmtText()

	res, err := advisor.Query(r.ctx, advisor.QueryContext{
		TenantMode:    r.tenantMode,
		PreExecutions: r.preExecutions,
	}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", statementText))

	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.InsertTooManyRows.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", statementText, err.Error()),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
		return
	}

	rowCount, err := getAffectedRows(res)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Internal.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("failed to get row count for \"%s\": %s", statementText, err.Error()),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
		return
	}

	if rowCount > int64(r.maxRow) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.StatementAffectedRowExceedsLimit.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The statement \"%s\" affected %d rows (estimated). The count exceeds %d.", statementText, rowCount, r.maxRow),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
