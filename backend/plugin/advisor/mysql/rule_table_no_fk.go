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
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
// The judgment lives in omni (mysql/v1review.TableNoForeignKey).
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

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
		for _, f := range mysqlv1review.TableNoForeignKey([]ast.Node{node}) {
			// V1 added the statement's base line twice: once itself and
			// once more in BaseRule.AddAdvice.
			position := base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f)))
			position.Line += int32(stmt.BaseLine())
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.TableHasFK.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", f.Name),
				StartPosition: position,
			})
		}
	}
	return adviceList, nil
}
