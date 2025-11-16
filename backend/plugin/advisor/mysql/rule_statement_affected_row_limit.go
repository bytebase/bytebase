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
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
func (*StatementAffectedRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
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
	rule := NewStatementAffectedRowLimitRule(ctx, level, string(checkCtx.Rule.Type), payload.Number, checkCtx.Driver)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	if checkCtx.Driver != nil {
		for _, stmt := range stmtList {
			rule.SetBaseLine(stmt.BaseLine)
			checker.SetBaseLine(stmt.BaseLine)
			antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
			if rule.explainCount >= common.MaximumLintExplainSize {
				break
			}
		}
	}

	return checker.GetAdviceList(), nil
}

// StatementAffectedRowLimitRule checks for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitRule struct {
	BaseRule
	text         string
	maxRow       int
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

// NewStatementAffectedRowLimitRule creates a new StatementAffectedRowLimitRule.
func NewStatementAffectedRowLimitRule(ctx context.Context, level storepb.Advice_Status, title string, maxRow int, driver *sql.DB) *StatementAffectedRowLimitRule {
	return &StatementAffectedRowLimitRule{
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
func (*StatementAffectedRowLimitRule) Name() string {
	return "StatementAffectedRowLimitRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementAffectedRowLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeUpdateStatement:
		if mysqlparser.IsTopMySQLRule(&ctx.(*mysql.UpdateStatementContext).BaseParserRuleContext) {
			r.handleStmt(ctx.GetStart().GetLine())
		}
	case NodeTypeDeleteStatement:
		if mysqlparser.IsTopMySQLRule(&ctx.(*mysql.DeleteStatementContext).BaseParserRuleContext) {
			r.handleStmt(ctx.GetStart().GetLine())
		}
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementAffectedRowLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementAffectedRowLimitRule) handleStmt(lineNumber int) {
	lineNumber += r.baseLine
	r.explainCount++
	res, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", r.text))
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementAffectedRowExceedsLimit.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", r.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	} else {
		rowCount, err := getRows(res)
		if err != nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.Internal.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", r.text, err.Error()),
				StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
			})
		} else if rowCount > int64(r.maxRow) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("\"%s\" affected %d rows (estimated). The count exceeds %d.", r.text, rowCount, r.maxRow),
				StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
			})
		}
	}
}

func getRows(res []any) (int64, error) {
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
