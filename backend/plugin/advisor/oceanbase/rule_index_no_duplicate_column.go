package oceanbase

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

var (
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor is the advisor checking for no duplicate
// columns in index. The judgment lives in omni
// (oceanbase/v1review.IndexNoDuplicateColumn); this adapter renders the findings
// as advice.
type IndexNoDuplicateColumnAdvisor struct {
}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oceanbasev1review.IndexNoDuplicateColumn([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DuplicateColumnInIndex.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("%s `%s` has duplicate column `%s`.`%s`", f.Value, keyName(stmt.Text, f), f.Table, f.Other),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}

// primaryKeyNameRe extracts the optional constraint name from a PRIMARY KEY clause.
// Matches patterns like: PRIMARY KEY pk_a (...), PRIMARY KEY `pk_a` (...)
var primaryKeyNameRe = regexp.MustCompile(`(?i)PRIMARY\s+KEY\s+` + "`?" + `(\w+)` + "`?" + `\s*\(`)

// extractPKNameFromText extracts the PRIMARY KEY constraint name from the statement text.
// Returns empty string if no name is found.
func extractPKNameFromText(stmtText string) string {
	m := primaryKeyNameRe.FindStringSubmatch(stmtText)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}
