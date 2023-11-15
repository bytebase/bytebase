package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
func (*StatementAffectedRowLimitAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &statementAffectedRowLimitChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		maxRow: payload.Number,
		driver: ctx.Driver,
		ctx:    ctx.Context,
	}

	if checker.driver != nil {
		for _, stmt := range stmtList {
			checker.baseLine = stmt.BaseLine
			antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		}
	}

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

type statementAffectedRowLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	maxRow     int
	driver     *sql.DB
	ctx        context.Context
}

func (checker *statementAffectedRowLimitChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *statementAffectedRowLimitChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	checker.handleStmt(ctx.GetStart().GetLine())
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *statementAffectedRowLimitChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	checker.handleStmt(ctx.GetStart().GetLine())
}

func (checker *statementAffectedRowLimitChecker) handleStmt(lineNumber int) {
	lineNumber += checker.baseLine
	res, err := advisor.Query(checker.ctx, checker.driver, fmt.Sprintf("EXPLAIN %s", checker.text))
	if err != nil {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementAffectedRowExceedsLimit,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", checker.text, err.Error()),
			Line:    lineNumber,
		})
	} else {
		rowCount, err := getRows(res)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   checker.title,
				Content: fmt.Sprintf("failed to get row count for \"%s\": %s", checker.text, err.Error()),
				Line:    lineNumber,
			})
		} else if rowCount > int64(checker.maxRow) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementAffectedRowExceedsLimit,
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\" affected %d rows. The count exceeds %d.", checker.text, rowCount, checker.maxRow),
				Line:    lineNumber,
			})
		}
	}
}

func getRows(res []any) (int64, error) {
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
