package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLInsertRowLimit, &InsertRowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLInsertRowLimit, &InsertRowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLInsertRowLimit, &InsertRowLimitAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type InsertRowLimitAdvisor struct {
}

// Check checks for table naming convention.
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

	checker := &insertRowLimitChecker{
		level:  level,
		title:  string(checkCtx.Rule.Type),
		maxRow: payload.Number,
		driver: checkCtx.Driver,
		ctx:    ctx,
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		if checker.explainCount >= common.MaximumLintExplainSize {
			break
		}
	}
	return checker.generateAdvice()
}

type insertRowLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	text         string
	line         int
	maxRow       int
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

func (checker *insertRowLimitChecker) generateAdvice() ([]*storepb.Advice, error) {
	return checker.adviceList, nil
}

func (checker *insertRowLimitChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *insertRowLimitChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
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

	checker.explainCount++
	res, err := advisor.Query(checker.ctx, advisor.QueryContext{}, checker.driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", checker.text))
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", checker.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
		})
		return
	}
	rowCount, err := getInsertRows(res)
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.Internal.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", checker.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
		})
	} else if rowCount > int64(checker.maxRow) {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", checker.text, rowCount, checker.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
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
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", checker.text, len(allValues), checker.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
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
