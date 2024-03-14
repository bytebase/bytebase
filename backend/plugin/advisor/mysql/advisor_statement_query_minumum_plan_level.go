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
	_ advisor.Advisor = (*StatementQueryMinumumPlanLevelAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementQueryMinumumPlanLevel, &StatementQueryMinumumPlanLevelAdvisor{})
}

type StatementQueryMinumumPlanLevelAdvisor struct {
}

func (*StatementQueryMinumumPlanLevelAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &statementQueryMinumumPlanLevelChecker{
		level:       level,
		title:       string(ctx.Rule.Type),
		driver:      ctx.Driver,
		ctx:         ctx.Context,
		explainType: convertExplainTypeFromString(strings.ToUpper(payload.String)),
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

type statementQueryMinumumPlanLevelChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine    int
	adviceList  []advisor.Advice
	level       advisor.Status
	title       string
	text        string
	driver      *sql.DB
	ctx         context.Context
	explainType ExplainType
}

type ExplainType int

const (
	ExplainTypeAll ExplainType = iota
	ExplainTypeIndex
	ExplainTypeRange
	ExplainTypeRef
	ExplainTypeEqRef
	ExplainTypeConst
)

func (t ExplainType) String() string {
	switch t {
	case ExplainTypeAll:
		return "ALL"
	case ExplainTypeIndex:
		return "INDEX"
	case ExplainTypeRange:
		return "RANGE"
	case ExplainTypeRef:
		return "REF"
	case ExplainTypeEqRef:
		return "EQ_REF"
	case ExplainTypeConst:
		return "CONST"
	default:
		return "UNKNOWN"
	}
}

func convertExplainTypeFromString(explainTypeStr string) ExplainType {
	switch explainTypeStr {
	case "ALL":
		return ExplainTypeAll
	case "INDEX":
		return ExplainTypeIndex
	case "RANGE":
		return ExplainTypeRange
	case "REF":
		return ExplainTypeRef
	case "EQ_REF":
		return ExplainTypeEqRef
	case "CONST":
		return ExplainTypeConst
	default:
		// Default to ALL if we don't recognize the explain type.
		return ExplainTypeAll
	}
}

func (checker *statementQueryMinumumPlanLevelChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *statementQueryMinumumPlanLevelChecker) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	if _, ok := ctx.GetParent().(*mysql.SimpleStatementContext); !ok {
		return
	}

	query := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	res, err := advisor.Query(checker.ctx, checker.driver, fmt.Sprintf("EXPLAIN %s", query))
	if err != nil {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementExplainQueryFailed,
			Title:   checker.title,
			Content: fmt.Sprintf("Failed to explain query: %s, with error: %s", query, err),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	} else {
		explainTypes, err := getQueryExplainTypes(res)
		if err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   checker.title,
				Content: fmt.Sprintf("Failed to check explain type column: %s, with error: %s", query, err),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		} else if len(explainTypes) > 0 {
			overused, overusedType := false, ExplainTypeAll
			for _, explainType := range explainTypes {
				if explainType < checker.explainType {
					overused = true
					overusedType = explainType
					break
				}
			}
			if overused {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.StatementUnwantedQueryPlanLevel,
					Title:   checker.title,
					Content: fmt.Sprintf("Overused query plan level detected %s, minimum plan level: %s", overusedType.String(), checker.explainType.String()),
					Line:    checker.baseLine + ctx.GetStart().GetLine(),
				})
			}
		}
	}
}

func getQueryExplainTypes(res []any) ([]ExplainType, error) {
	if len(res) != 3 {
		return nil, errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return nil, errors.Errorf("expected []any but got %t", res[2])
	}
	if len(rowList) < 1 {
		return nil, errors.Errorf("not found any data")
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

	explainTypes := []ExplainType{}
	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return nil, errors.Errorf("expected []any but got %t", row)
		}
		if len(row) != 12 {
			return nil, errors.Errorf("expected 12 but got %d", len(row))
		}
		explainType, ok := row[4].(string)
		if !ok {
			// Skip the row if the type column is not a string.
			continue
		}
		explainTypes = append(explainTypes, convertExplainTypeFromString(explainType))
	}

	return explainTypes, nil
}
