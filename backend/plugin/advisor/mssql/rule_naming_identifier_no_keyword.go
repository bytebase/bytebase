package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier
// naming convention without keyword. The judgment lives in omni
// (mssql/v1review.NamingIdentifierNoKeyword); this adapter renders the
// findings as advice, each on the line where the statement text first holds
// the finding's search text.
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
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mssqlv1review.NamingIdentifierNoKeyword([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NameIsKeywordIdentifier.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Identifier [%s] is a keyword identifier and should be avoided.", f.Name),
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findLineByName(stmt.Text, f.Search)},
			})
		}
	}
	return adviceList, nil
}

// findLineByName searches for an identifier name in text and returns its
// 1-based line number, or the statement's first content line when not found.
func findLineByName(text, name string) int32 {
	if name == "" || text == "" {
		return findingLine(text, v1review.Finding{})
	}
	idx := strings.Index(text, name)
	if idx < 0 {
		return findingLine(text, v1review.Finding{})
	}
	return tsqlparser.ByteOffsetToRunePosition(text, idx).Line
}
