package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementWhereMaximumLogicalOperatorCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT, &StatementWhereMaximumLogicalOperatorCountAdvisor{})
}

// StatementWhereMaximumLogicalOperatorCountAdvisor is the advisor checking the number of logical operands in WHERE clauses.
// The judgment lives in omni (mysql/v1review.StatementWhereMaximumLogicalOperatorCount); this
// adapter renders its findings as advice.
type StatementWhereMaximumLogicalOperatorCountAdvisor struct {
}

func (*StatementWhereMaximumLogicalOperatorCountAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	maximum := int(numberPayload.Number)

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mysqlv1review.StatementWhereMaximumLogicalOperatorCount([]ast.Node{node}, maximum) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementWhereMaximumLogicalOperatorCount.Int32(),
				Title:         title,
				Content:       logicalOperatorContent(f, maximum, strings.TrimSpace(stmt.Text)),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}

func logicalOperatorContent(f v1review.Finding, maximum int, text string) string {
	if f.Name == "IN" {
		return fmt.Sprintf("Number of tokens (%s) in IN predicate operation exceeds limit (%d) in statement %q.", f.Value, maximum, text)
	}
	return fmt.Sprintf("Number of tokens (%s) in the OR predicate operation exceeds limit (%d) in statement %q.", f.Value, maximum, text)
}
