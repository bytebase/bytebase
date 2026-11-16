package mysql

import (
	"context"
	"fmt"

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
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number
// limit. The judgment lives in omni (mysql/v1review.IndexKeyNumberLimit);
// this adapter converts the rule's payload and renders the findings as
// advice.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	limit := int(numberPayload.Number)

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
		for _, f := range mysqlv1review.IndexKeyNumberLimit([]ast.Node{node}, limit) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexKeyNumberExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", keyName(stmt.Text, f), f.Table, limit),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}

// keyName returns the name the index rules print for a finding's key. The
// omni AST keeps no name written after PRIMARY KEY, so an unnamed primary key
// takes the first one the statement text has.
func keyName(stmtText string, f v1review.Finding) string {
	if f.Name == "" && f.Value == v1review.KeyPrimary {
		return extractPKNameFromText(stmtText)
	}
	return f.Name
}
