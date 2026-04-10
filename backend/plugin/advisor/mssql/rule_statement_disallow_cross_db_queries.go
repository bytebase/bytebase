package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES, &DisallowCrossDBQueriesAdvisor{})
}

type DisallowCrossDBQueriesAdvisor struct{}

func (*DisallowCrossDBQueriesAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &disallowCrossDBQueriesRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		curDB:        checkCtx.CurrentDatabase,
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type disallowCrossDBQueriesRule struct {
	OmniBaseRule
	curDB string
}

func (*disallowCrossDBQueriesRule) Name() string {
	return "DisallowCrossDBQueriesRule"
}

func (r *disallowCrossDBQueriesRule) OnStatement(node ast.Node) {
	// Check for USE statement to track current database.
	if useStmt, ok := node.(*ast.UseStmt); ok {
		if useStmt.Database != "" {
			r.curDB = strings.ToLower(useStmt.Database)
		}
		return
	}

	ast.Inspect(node, func(n ast.Node) bool {
		ref, ok := n.(*ast.TableRef)
		if !ok || ref == nil {
			return true
		}
		if ref.Database != "" && !strings.EqualFold(ref.Database, r.curDB) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementDisallowCrossDBQueries.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", ref.Database, r.curDB),
				StartPosition: &storepb.Position{Line: r.LocToLine(ref.Loc)},
			})
		}
		return true
	})
}
