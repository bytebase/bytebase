package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*FullyQualifiedObjectNameAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED, &FullyQualifiedObjectNameAdvisor{})
}

// FullyQualifiedObjectNameAdvisor is the advisor checking for fully
// qualified object names. The judgment lives in omni
// (pg/v1review.NamingFullyQualified); this adapter passes it the synced
// schema and renders the findings as advice.
type FullyQualifiedObjectNameAdvisor struct {
}

// Check checks for fully qualified object names.
func (*FullyQualifiedObjectNameAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	schema := v1review.Schema{Metadata: checkCtx.DBSchema}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		// V1 anchored every finding on the statement text's last line.
		line := (&BaseRule{StmtText: stmt.Text}).ContentEndLine() + int32(stmt.BaseLine())
		for _, f := range pgv1review.NamingFullyQualified([]ast.Node{node}, schema) {
			if f.Code != v1review.NameUnqualified {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          int32(code.NamingNotFullyQualifiedName),
				Title:         title,
				Content:       fmt.Sprintf("unqualified object name: '%s'", f.Name),
				StartPosition: &storepb.Position{Line: line},
			})
		}
	}
	return adviceList, nil
}
