package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementQueryMinumumPlanLevelAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementQueryMinumumPlanLevel, &StatementQueryMinumumPlanLevelAdvisor{})
}

type StatementQueryMinumumPlanLevelAdvisor struct {
}

func (*StatementQueryMinumumPlanLevelAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementQueryMinumumPlanLevelRule(ctx, level, string(checkCtx.Rule.Type), checkCtx.Driver, convertExplainTypeFromString(strings.ToUpper(payload.String)))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	if rule.driver != nil {
		for _, stmt := range stmtList {
			rule.SetBaseLine(stmt.BaseLine)
			checker.SetBaseLine(stmt.BaseLine)
			antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
			if rule.GetExplainCount() >= common.MaximumLintExplainSize {
				break
			}
		}
	}

	return checker.GetAdviceList(), nil
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

// StatementQueryMinumumPlanLevelRule checks for query minimum plan level.
type StatementQueryMinumumPlanLevelRule struct {
	BaseRule
	text         string
	driver       *sql.DB
	ctx          context.Context
	explainType  ExplainType
	explainCount int
}

// NewStatementQueryMinumumPlanLevelRule creates a new StatementQueryMinumumPlanLevelRule.
func NewStatementQueryMinumumPlanLevelRule(ctx context.Context, level storepb.Advice_Status, title string, driver *sql.DB, explainType ExplainType) *StatementQueryMinumumPlanLevelRule {
	return &StatementQueryMinumumPlanLevelRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		driver:      driver,
		ctx:         ctx,
		explainType: explainType,
	}
}

// Name returns the rule name.
func (*StatementQueryMinumumPlanLevelRule) Name() string {
	return "StatementQueryMinumumPlanLevelRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementQueryMinumumPlanLevelRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeSelectStatement:
		r.checkSelectStatement(ctx.(*mysql.SelectStatementContext))
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementQueryMinumumPlanLevelRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// GetExplainCount returns the explain count.
func (r *StatementQueryMinumumPlanLevelRule) GetExplainCount() int {
	return r.explainCount
}

func (r *StatementQueryMinumumPlanLevelRule) checkSelectStatement(ctx *mysql.SelectStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if _, ok := ctx.GetParent().(*mysql.SimpleStatementContext); !ok {
		return
	}

	query := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	r.explainCount++
	res, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", query))
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementExplainQueryFailed.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Failed to explain query: %s, with error: %s", query, err),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	} else {
		explainTypes, err := getQueryExplainTypes(res)
		if err != nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.Internal.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Failed to check explain type column: %s, with error: %s", query, err),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		} else if len(explainTypes) > 0 {
			overused, overusedType := false, ExplainTypeAll
			for _, explainType := range explainTypes {
				if explainType < r.explainType {
					overused = true
					overusedType = explainType
					break
				}
			}
			if overused {
				r.AddAdvice(&storepb.Advice{
					Status:        r.level,
					Code:          code.StatementUnwantedQueryPlanLevel.Int32(),
					Title:         r.title,
					Content:       fmt.Sprintf("Overused query plan level detected %s, minimum plan level: %s", overusedType.String(), r.explainType.String()),
					StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
				})
			}
		}
	}
}

func getQueryExplainTypes(res []any) ([]ExplainType, error) {
	if len(res) != 3 {
		return nil, errors.Errorf("expected 3 but got %d", len(res))
	}
	columns, ok := res[0].([]string)
	if !ok {
		return nil, errors.Errorf("expected []string but got %t", res[0])
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
	typeIndex, err := getColumnIndex(columns, "type")
	if err != nil {
		return nil, errors.Errorf("failed to find rows column")
	}

	explainTypes := []ExplainType{}
	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return nil, errors.Errorf("expected []any but got %t", row)
		}
		explainType, ok := row[typeIndex].(string)
		if !ok {
			// Skip the row if the type column is not a string.
			continue
		}
		explainTypes = append(explainTypes, convertExplainTypeFromString(explainType))
	}

	return explainTypes, nil
}
