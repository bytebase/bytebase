package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET, &TableDisallowSetCharsetAdvisor{})
}

// TableDisallowSetCharsetAdvisor is the advisor checking for disallow setting a table charset.
// The judgment lives in omni (mysql/v1review.TableDisallowSetCharset).
type TableDisallowSetCharsetAdvisor struct {
}

func (*TableDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range mysqlv1review.TableDisallowSetCharset([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisallowSetCharset.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", strings.TrimSpace(stmt.Text)),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, v1review.Finding{}))),
			})
		}
	}
	return adviceList, nil
}
