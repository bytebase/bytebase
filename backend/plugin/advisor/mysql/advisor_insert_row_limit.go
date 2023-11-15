package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLInsertRowLimit, &InsertRowLimitAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type InsertRowLimitAdvisor struct {
}

// Check checks for table naming convention.
func (*InsertRowLimitAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &insertRowLimitChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		maxRow: payload.Number,
		driver: ctx.Driver,
		ctx:    ctx.Context,
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}
	return checker.generateAdvice()
}

type insertRowLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
	maxRow     int
	driver     *sql.DB
	ctx        context.Context
}

func (checker *insertRowLimitChecker) generateAdvice() ([]advisor.Advice, error) {
	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

func (checker *insertRowLimitChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *insertRowLimitChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	checker.line = checker.baseLine + ctx.GetStart().GetLine()
	if ctx.InsertQueryExpression() != nil {
		checker.handleInsertQueryExpression(ctx.InsertQueryExpression())
	}
	checker.handleNoInsertQueryExpression(ctx)
}

func (checker *insertRowLimitChecker) handleInsertQueryExpression(ctx mysql.IInsertQueryExpressionContext) {
	if checker.driver == nil || ctx == nil {
		return
	}

	res, err := advisor.Query(checker.ctx, checker.driver, fmt.Sprintf("EXPLAIN %s", checker.text))
	if err != nil {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.InsertTooManyRows,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", checker.text, err.Error()),
			Line:    checker.line,
		})
		return
	}
	rowCount, err := getInsertRows(res)
	if err != nil {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.Internal,
			Title:   checker.title,
			Content: fmt.Sprintf("failed to get row count for \"%s\": %s", checker.text, err.Error()),
			Line:    checker.line,
		})
	} else if rowCount > int64(checker.maxRow) {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.InsertTooManyRows,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", checker.text, rowCount, checker.maxRow),
			Line:    checker.line,
		})
	}
}

func (checker *insertRowLimitChecker) handleNoInsertQueryExpression(ctx mysql.IInsertStatementContext) {
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
	if len(allValues) > checker.maxRow {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.InsertTooManyRows,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", checker.text, len(allValues), checker.maxRow),
			Line:    checker.line,
		})
	}
}

func getInsertRows(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
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

	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any but got %t", row)
		}
		if len(row) != 12 {
			return 0, errors.Errorf("expected 12 but got %d", len(row))
		}
		switch col := row[9].(type) {
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

	return 0, errors.Errorf("failed to extract rows from query plan")
}
