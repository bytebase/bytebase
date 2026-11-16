package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var _ advisor.Advisor = (*StatementDisallowTruncateAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
}

// StatementDisallowTruncateAdvisor is the advisor disallowing TRUNCATE. The
// judgment lives in omni (pg/v1review.StatementDisallowTruncate); this
// adapter renders its findings as advice.
type StatementDisallowTruncateAdvisor struct{}

func (*StatementDisallowTruncateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		rule := &BaseRule{StmtText: stmt.Text}
		for _, f := range pgv1review.StatementDisallowTruncate([]ast.Node{node}) {
			suffix := ""
			for _, option := range strings.Split(f.Value, ",") {
				switch option {
				case "CASCADE":
					suffix += " CASCADE will wipe dependent tables."
				case "RESTART IDENTITY":
					suffix += " RESTART IDENTITY will reset owned sequences."
				default:
				}
			}
			loc := ast.Loc{Start: f.Start, End: f.End}
			if f.Start == 0 && f.End == 0 {
				loc.Start = -1
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status: level,
				Code:   code.StatementDisallowTruncate.Int32(),
				Title:  title,
				Content: fmt.Sprintf(
					`TRUNCATE TABLE %q is not allowed: TRUNCATE bypasses triggers. Unlike Oracle/MySQL/MSSQL, PG TRUNCATE is transactional and rolls back cleanly.%s`,
					f.Table, suffix,
				),
				StartPosition: &storepb.Position{Line: rule.LocToLine(loc) + int32(stmt.BaseLine())},
			})
		}
	}
	return adviceList, nil
}
