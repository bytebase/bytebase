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
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for to limit INSERT rows.
type InsertRowLimitAdvisor struct {
}

// Check checks for the INSERT row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 {
		return nil, nil
	}

	rule := &insertRowLimitRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		maxRow:     int(numberPayload.Number),
		driver:     checkCtx.Driver,
		ctx:        ctx,
		TenantMode: checkCtx.TenantMode,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type insertRowLimitRule struct {
	OmniBaseRule

	maxRow        int
	driver        *sql.DB
	ctx           context.Context
	explainCount  int
	preExecutions []string
	TenantMode    bool
}

func (*insertRowLimitRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT)
}

func (r *insertRowLimitRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.VariableSetStmt:
		if omniIsRoleOrSearchPathSet(n) {
			r.preExecutions = append(r.preExecutions, r.TrimmedStmtText())
		}
	case *ast.InsertStmt:
		r.checkInsert(n)
	default:
	}
}

func (r *insertRowLimitRule) checkInsert(ins *ast.InsertStmt) {
	code := advisorcode.Ok
	rows := int64(0)
	statementText := r.TrimmedStmtText()

	// Count VALUES rows if this is INSERT ... VALUES.
	if sel, ok := ins.SelectStmt.(*ast.SelectStmt); ok && sel.ValuesLists != nil {
		rowCount := len(sel.ValuesLists.Items)
		if rowCount > r.maxRow {
			code = advisorcode.InsertTooManyRows
			rows = int64(rowCount)
		}
	} else if ins.SelectStmt != nil && r.driver != nil {
		// For INSERT ... SELECT, use EXPLAIN.
		if r.explainCount >= common.MaximumLintExplainSize {
			return
		}
		r.explainCount++

		res, err := advisor.Query(r.ctx, advisor.QueryContext{
			TenantMode:    r.TenantMode,
			PreExecutions: r.preExecutions,
		}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", statementText))

		if err != nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    advisorcode.InsertTooManyRows.Int32(),
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
				Code:    advisorcode.Internal.Int32(),
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
			code = advisorcode.InsertTooManyRows
			rows = rowCount
		}
	}

	if code != advisorcode.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The statement \"%s\" inserts %d rows. The count exceeds %d.", statementText, rows, r.maxRow),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
