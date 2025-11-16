package pg

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementInsertRowLimit, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for to limit INSERT rows.
type InsertRowLimitAdvisor struct {
}

// Check checks for the INSERT row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := &insertRowLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		maxRow:                   payload.Number,
		driver:                   checkCtx.Driver,
		ctx:                      ctx,
		statementsText:           checkCtx.Statements,
		UsePostgresDatabaseOwner: checkCtx.UsePostgresDatabaseOwner,
	}

	if payload.Number > 0 {
		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)
		return checker.GetAdviceList(), nil
	}

	return nil, nil
}

type insertRowLimitRule struct {
	BaseRule

	maxRow                   int
	driver                   *sql.DB
	ctx                      context.Context
	statementsText           string
	explainCount             int
	setRoles                 []string
	UsePostgresDatabaseOwner bool
}

func (*insertRowLimitRule) Name() string {
	return "insert_row_limit"
}

func (r *insertRowLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Variablesetstmt":
		if c, ok := ctx.(*parser.VariablesetstmtContext); ok {
			r.handleVariablesetstmt(c)
		}
	case "Insertstmt":
		if c, ok := ctx.(*parser.InsertstmtContext); ok {
			r.handleInsertstmt(c)
		}
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*insertRowLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *insertRowLimitRule) handleVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Track SET ROLE statements
	if ctx.Set_rest() != nil && ctx.Set_rest().Set_rest_more() != nil {
		setRestMore := ctx.Set_rest().Set_rest_more()
		// Check if this is SET ROLE
		if setRestMore.ROLE() != nil {
			r.setRoles = append(r.setRoles, ctx.GetText())
		}
	}
}

func (r *insertRowLimitRule) handleInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	code := advisorcode.Ok
	rows := int64(0)

	// Check if this is INSERT ... VALUES or INSERT ... SELECT
	if ctx.Insert_rest() != nil && ctx.Insert_rest().Selectstmt() != nil {
		// Count the number of value lists if this is VALUES
		rowCount := countValueLists(ctx.Insert_rest().Selectstmt())
		if rowCount > 0 {
			// This is INSERT ... VALUES
			if rowCount > r.maxRow {
				code = advisorcode.InsertTooManyRows
				rows = int64(rowCount)
			}
		} else if r.driver != nil {
			// For INSERT ... SELECT, use EXPLAIN
			if r.explainCount >= common.MaximumLintExplainSize {
				return
			}
			r.explainCount++

			stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
			res, err := advisor.Query(r.ctx, advisor.QueryContext{
				UsePostgresDatabaseOwner: r.UsePostgresDatabaseOwner,
				PreExecutions:            r.setRoles,
			}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", stmtText))

			if err != nil {
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    advisorcode.InsertTooManyRows.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("\"%s\" dry runs failed: %s", stmtText, err.Error()),
					StartPosition: &storepb.Position{
						Line:   int32(ctx.GetStart().GetLine()),
						Column: 0,
					},
				})
				return
			}

			rowCount, err := getAffectedRows(res)
			if err != nil {
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    advisorcode.Internal.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("failed to get row count for \"%s\": %s", stmtText, err.Error()),
					StartPosition: &storepb.Position{
						Line:   int32(ctx.GetStart().GetLine()),
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
	}

	if code != advisorcode.Ok {
		stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The statement \"%s\" inserts %d rows. The count exceeds %d.", stmtText, rows, r.maxRow),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// countValueLists counts the number of value lists in an INSERT ... VALUES statement
// Returns 0 if this is not a VALUES statement (e.g., INSERT ... SELECT)
func countValueLists(selectStmt parser.ISelectstmtContext) int {
	if selectStmt == nil {
		return 0
	}

	// Navigate to the values_clause
	// SELECT can be select_no_parens or select_with_parens
	if selectStmt.Select_no_parens() != nil {
		return countValuesInSelectNoParens(selectStmt.Select_no_parens())
	}

	if selectStmt.Select_with_parens() != nil {
		return countValuesInSelectWithParens(selectStmt.Select_with_parens())
	}

	return 0
}

// countValuesInSelectNoParens counts VALUES rows in a select_no_parens
func countValuesInSelectNoParens(selectCtx parser.ISelect_no_parensContext) int {
	if selectCtx == nil || selectCtx.Select_clause() == nil {
		return 0
	}

	// Check if this is a values_clause
	return countValuesInSelectClause(selectCtx.Select_clause())
}

// countValuesInSelectWithParens counts VALUES rows in a select_with_parens
func countValuesInSelectWithParens(selectCtx parser.ISelect_with_parensContext) int {
	if selectCtx == nil {
		return 0
	}

	if selectCtx.Select_no_parens() != nil {
		return countValuesInSelectNoParens(selectCtx.Select_no_parens())
	}

	if selectCtx.Select_with_parens() != nil {
		return countValuesInSelectWithParens(selectCtx.Select_with_parens())
	}

	return 0
}

// countValuesInSelectClause counts VALUES rows in a select_clause
func countValuesInSelectClause(selectClause parser.ISelect_clauseContext) int {
	if selectClause == nil {
		return 0
	}

	// select_clause has AllSimple_select_intersect
	allIntersect := selectClause.AllSimple_select_intersect()
	if len(allIntersect) == 0 {
		return 0
	}

	// Check the first one for values_clause
	return countValuesInSimpleSelectIntersect(allIntersect[0])
}

// countValuesInSimpleSelectIntersect counts VALUES rows in simple_select_intersect
func countValuesInSimpleSelectIntersect(intersect parser.ISimple_select_intersectContext) int {
	if intersect == nil {
		return 0
	}

	// Get all simple_select_pramary
	allPrimary := intersect.AllSimple_select_pramary()
	if len(allPrimary) == 0 {
		return 0
	}

	// Check the first one for values_clause
	return countValuesInPrimary(allPrimary[0])
}

// countValuesInPrimary counts VALUES rows in simple_select_pramary
func countValuesInPrimary(primary parser.ISimple_select_pramaryContext) int {
	if primary == nil || primary.Values_clause() == nil {
		return 0
	}

	// values_clause: VALUES (expr_list) (, (expr_list))*
	// Count the number of COMMA tokens + 1
	valuesClause := primary.Values_clause()
	commaCount := len(valuesClause.AllCOMMA())
	return commaCount + 1
}

func getAffectedRows(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	// EXPLAIN output has at least 2 rows
	if len(rowList) < 2 {
		return 0, errors.Errorf("not found any data")
	}
	// We need row 2
	rowTwo, ok := rowList[1].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", rowList[0])
	}
	// PostgreSQL EXPLAIN result has one column
	if len(rowTwo) != 1 {
		return 0, errors.Errorf("expected one but got %d", len(rowTwo))
	}
	// Get the string value
	text, ok := rowTwo[0].(string)
	if !ok {
		return 0, errors.Errorf("expected string but got %t", rowTwo[0])
	}

	rowsRegexp := regexp.MustCompile("rows=([0-9]+)")
	matches := rowsRegexp.FindStringSubmatch(text)
	if len(matches) != 2 {
		return 0, errors.Errorf("failed to find rows in %q", text)
	}
	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, errors.Errorf("failed to get integer from %q", matches[1])
	}
	return value, nil
}
