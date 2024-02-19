package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementSelectFullTableScanAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementSelectFullTableScan, &StatementSelectFullTableScanAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLStatementSelectFullTableScan, &StatementSelectFullTableScanAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLStatementSelectFullTableScan, &StatementSelectFullTableScanAdvisor{})
}

type StatementSelectFullTableScanAdvisor struct {
}

func (*StatementSelectFullTableScanAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementSelectFullTableScanChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		driver: ctx.Driver,
		ctx:    ctx.Context,
	}

	if checker.driver != nil {
		for _, stmt := range stmtList {
			checker.baseLine = stmt.BaseLine
			antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		}
	}

	return checker.adviceList, nil
}

type statementSelectFullTableScanChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	driver     *sql.DB
	ctx        context.Context
}

func (checker *statementSelectFullTableScanChecker) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	if _, ok := ctx.GetParent().(*mysql.SimpleStatementContext); !ok {
		return
	}

	query := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	res, err := advisor.Query(checker.ctx, checker.driver, fmt.Sprintf("EXPLAIN %s", query))
	if err != nil {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementCheckSelectFullTableScanFailed,
			Title:   checker.title,
			Content: fmt.Sprintf("Failed to check full table scan: %s, with error: %s", query, err),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	} else {
		hasFullScan, tables, err := hasTableFullScan(res)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   checker.title,
				Content: fmt.Sprintf("Failed to check full table scan: %s, with error: %s", query, err),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		} else if hasFullScan {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementHasTableFullScan,
				Title:   checker.title,
				Content: fmt.Sprintf("Full table scan detected on table(s): %s", tables),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		}
	}
}

func hasTableFullScan(res []any) (bool, string, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return false, "", errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return false, "", errors.Errorf("expected []any but got %t", res[2])
	}
	if len(rowList) < 1 {
		return false, "", errors.Errorf("not found any data")
	}

	// MySQL EXPLAIN statement result has 12 columns.
	// 1. the column 4 is the data 'type'.
	// 	  We check all rows of the result to see if any of them has 'ALL' or 'index' in the 'type' column.
	// 2. the column 11 is the 'Extra' column.
	//    If the 'Extra' column dose not contain
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

	var tables []string

	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return false, "", errors.Errorf("expected []any but got %t", row)
		}
		if len(row) != 12 {
			return false, "", errors.Errorf("expected 12 but got %d", len(row))
		}
		if row[4] == "ALL" {
			tables = append(tables, row[2].(string))
			continue
		}
		if row[4] == "index" {
			if strings.Contains(row[11].(string), "Using where") || strings.Contains(row[11].(string), "Using index condition") {
				continue
			}
			tables = append(tables, row[2].(string))
			continue
		}
	}

	if len(tables) == 0 {
		return false, "", nil
	}

	return true, strings.Join(tables, ", "), nil
}
