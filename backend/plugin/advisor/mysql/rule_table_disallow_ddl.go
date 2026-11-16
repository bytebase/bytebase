package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableDisallowDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor is the advisor checking for disallow DDL on specific tables.
// The judgment lives in omni (mysql/v1review.TableDisallowDDL).
type TableDisallowDDLAdvisor struct {
}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	// A rule without a string array payload panics here, as V1 did.
	disallowList := checkCtx.Rule.GetStringArrayPayload().List

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
		for _, f := range mysqlv1review.TableDisallowDDL([]ast.Node{node}, disallowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.TableDisallowDDL.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
