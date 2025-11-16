package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementInsertRowLimit, &InsertRowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementInsertRowLimit, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for insert row limit.
type InsertRowLimitAdvisor struct {
}

// Check checks for insert row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewInsertRowLimitRule(ctx, level, string(checkCtx.Rule.Type), payload.Number, checkCtx.Driver)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		if rule.GetExplainCount() >= common.MaximumLintExplainSize {
			break
		}
	}

	return checker.GetAdviceList(), nil
}

// InsertRowLimitRule checks for insert row limit.
type InsertRowLimitRule struct {
	BaseRule
	text         string
	line         int
	maxRow       int
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

// NewInsertRowLimitRule creates a new InsertRowLimitRule.
func NewInsertRowLimitRule(ctx context.Context, level storepb.Advice_Status, title string, maxRow int, driver *sql.DB) *InsertRowLimitRule {
	return &InsertRowLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maxRow: maxRow,
		driver: driver,
		ctx:    ctx,
	}
}

// Name returns the rule name.
func (*InsertRowLimitRule) Name() string {
	return "InsertRowLimitRule"
}

// OnEnter is called when entering a parse tree node.
func (r *InsertRowLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeInsertStatement:
		r.checkInsertStatement(ctx.(*mysql.InsertStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*InsertRowLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// GetExplainCount returns the explain count.
func (r *InsertRowLimitRule) GetExplainCount() int {
	return r.explainCount
}

func (r *InsertRowLimitRule) checkInsertStatement(ctx *mysql.InsertStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	r.line = r.baseLine + ctx.GetStart().GetLine()
	if ctx.InsertQueryExpression() != nil {
		r.handleInsertQueryExpression(ctx.InsertQueryExpression())
	}
	r.handleNoInsertQueryExpression(ctx)
}

func (r *InsertRowLimitRule) handleInsertQueryExpression(ctx mysql.IInsertQueryExpressionContext) {
	if r.driver == nil || ctx == nil {
		return
	}

	r.explainCount++
	res, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", r.text))
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", r.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(r.line),
		})
		return
	}
	rowCount, err := getInsertRows(res)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.Internal.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", r.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(r.line),
		})
	} else if rowCount > int64(r.maxRow) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", r.text, rowCount, r.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(r.line),
		})
	}
}

func (r *InsertRowLimitRule) handleNoInsertQueryExpression(ctx mysql.IInsertStatementContext) {
	if ctx.InsertFromConstructor() == nil {
		return
	}
	if ctx.InsertFromConstructor().InsertValues() == nil {
		return
	}
	if ctx.InsertFromConstructor().InsertValues().ValueList() == nil {
		return
	}

	allValues := ctx.InsertFromConstructor().InsertValues().ValueList().AllValues()
	if len(allValues) > r.maxRow {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", r.text, len(allValues), r.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(r.line),
		})
	}
}

func getInsertRows(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	columns, ok := res[0].([]string)
	if !ok {
		return 0, errors.Errorf("expected []string but got %t", res[0])
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	if len(rowList) < 1 {
		return 0, errors.Errorf("not found any data")
	}

	// MySQL EXPLAIN statement result has 12 columns.
	// the column 9 is the data 'rows'.
	// the first not-NULL value of column 9 is the affected rows count.
	//
	// mysql> explain delete from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// |  1 | DELETE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | NULL  |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	//
	// mysql> explain insert into td select * from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra           |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// |  1 | INSERT      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL | NULL |     NULL | NULL            |
	// |  1 | SIMPLE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | Using temporary |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+

	rowsIndex, err := getColumnIndex(columns, "rows")
	if err != nil {
		return 0, errors.Errorf("failed to find rows column")
	}

	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any but got %t", row)
		}
		switch col := row[rowsIndex].(type) {
		case int:
			return int64(col), nil
		case int32:
			return int64(col), nil
		case int64:
			return col, nil
		case string:
			v, err := strconv.ParseInt(col, 10, 64)
			if err != nil {
				return 0, errors.Errorf("expected int or int64 but got string(%s)", col)
			}
			return v, nil
		default:
			continue
		}
	}

	return 0, nil
}
