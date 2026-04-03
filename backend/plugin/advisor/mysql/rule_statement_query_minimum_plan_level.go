package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

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
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_QUERY_MINIMUM_PLAN_LEVEL, &StatementQueryMinumumPlanLevelAdvisor{})
}

type StatementQueryMinumumPlanLevelAdvisor struct {
}

func (*StatementQueryMinumumPlanLevelAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringPayload := checkCtx.Rule.GetStringPayload()
	if stringPayload == nil {
		return nil, errors.New("string_payload is required for this rule")
	}

	driver := checkCtx.Driver
	if driver == nil {
		return nil, nil
	}

	title := checkCtx.Rule.Type.String()
	explainType := convertExplainTypeFromString(strings.ToUpper(stringPayload.Value))

	rule := &statementQueryMinPlanLevelOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: title,
		},
		driver:      driver,
		ctx:         ctx,
		explainType: explainType,
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
		if rule.explainCount >= common.MaximumLintExplainSize {
			break
		}
	}

	return rule.GetAdviceList(), nil
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
		return ExplainTypeAll
	}
}

// statementQueryMinPlanLevelOmniRule checks for query minimum plan level using omni AST.
type statementQueryMinPlanLevelOmniRule struct {
	OmniBaseRule
	driver       *sql.DB
	ctx          context.Context
	explainType  ExplainType
	explainCount int
}

func (*statementQueryMinPlanLevelOmniRule) Name() string {
	return "StatementQueryMinumumPlanLevelRule"
}

func (r *statementQueryMinPlanLevelOmniRule) OnStatement(node ast.Node) {
	if _, ok := node.(*ast.SelectStmt); !ok {
		return
	}

	query := r.TrimmedStmtText()
	line := r.BaseLine + int(r.ContentStartLine())
	r.explainCount++
	res, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", query))
	if err != nil {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.StatementExplainQueryFailed.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Failed to explain query: %s, with error: %s", query, err),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
		return
	}

	explainTypes, err := getQueryExplainTypes(res)
	if err != nil {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.Internal.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Failed to check explain type column: %s, with error: %s", query, err),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
		return
	}

	if len(explainTypes) > 0 {
		overused, overusedType := false, ExplainTypeAll
		for _, et := range explainTypes {
			if et < r.explainType {
				overused = true
				overusedType = et
				break
			}
		}
		if overused {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementUnwantedQueryPlanLevel.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Overused query plan level detected %s, minimum plan level: %s", overusedType.String(), r.explainType.String()),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
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
			continue
		}
		explainTypes = append(explainTypes, convertExplainTypeFromString(explainType))
	}

	return explainTypes, nil
}
