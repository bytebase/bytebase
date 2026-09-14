package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE/MERGE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE/MERGE affected row limit, including data-modifying CTEs.
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

	adviceList := RunRules(checkCtx.ParsedStatements, []OmniRule{rule})
	return rule.explains.AppendSkippedAdvice(adviceList, rule.Title, code.StatementAffectedRowExceedsLimit), nil
}

type statementAffectedRowLimitRule struct {
	OmniBaseRule
	maxRow     int
	driver     *sql.DB
	ctx        context.Context
	explains   advisor.ExplainBudget
	settings   sessionSettings
	tenantMode bool
}

func (*statementAffectedRowLimitRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT)
}

func (r *statementAffectedRowLimitRule) OnStatement(node ast.Node) {
	r.settings.add(node, r.TrimmedStmtText())
	node, text := pgparser.UnwrapExplainAnalyze(node, r.StmtText)
	switch n := node.(type) {
	case *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
		r.checkAffectedRows(text)
	case *ast.SelectStmt:
		if hasDataModifyingCTE(n.WithClause) {
			r.checkAffectedRows(text)
		}
	case *ast.InsertStmt:
		if hasDataModifyingCTE(n.WithClause) {
			r.checkAffectedRows(text)
		}
	case *ast.CreateTableAsStmt:
		if query, ok := n.Query.(*ast.SelectStmt); ok && hasDataModifyingCTE(query.WithClause) {
			r.checkAffectedRows(text)
		}
	default:
	}
}

func (r *statementAffectedRowLimitRule) checkAffectedRows(text string) {
	if !r.explains.Spend(&storepb.Position{Line: r.ContentStartLine() + int32(r.BaseLine)}) {
		return
	}

	statementText := strings.TrimRight(strings.TrimSpace(text), ";")

	res, err := advisor.Query(r.ctx, advisor.QueryContext{
		TenantMode:    r.tenantMode,
		PreExecutions: r.settings.statements(),
	}, r.driver, storepb.Engine_POSTGRES, getExplainSQL(statementText))

	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.StatementAffectedRowExceedsLimit.Int32(),
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
