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
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier
// naming convention without keyword. The judgment lives in omni
// (mysql/v1review.NamingIdentifierNoKeyword); this adapter renders the
// findings as advice.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		text := &BaseRule{StmtText: stmt.Text}
		for _, f := range mysqlv1review.NamingIdentifierNoKeyword([]ast.Node{node}) {
			// A finding with search text is on the line where the statement
			// first holds that text; any other is on its range's first line.
			line := findingLine(stmt.Text, f)
			if f.Search != "" {
				line = text.FindLineByName(f.Search)
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NameIsKeywordIdentifier.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Identifier %q is a keyword and should be avoided", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(line)),
			})
		}
	}
	return adviceList, nil
}
