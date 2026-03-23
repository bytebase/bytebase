package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexConcurrentlyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY, &IndexConcurrentlyAdvisor{})
}

// IndexConcurrentlyAdvisor is the advisor checking for to create index concurrently.
type IndexConcurrentlyAdvisor struct {
}

// Check checks for to create index concurrently.
func (*IndexConcurrentlyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &indexCreateConcurrentlyRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		newlyCreatedTables: make(map[string]bool),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexCreateConcurrentlyRule struct {
	OmniBaseRule

	newlyCreatedTables map[string]bool
}

func (*indexCreateConcurrentlyRule) Name() string {
	return string(storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY)
}

func (r *indexCreateConcurrentlyRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		tableName := omniTableName(n.Relation)
		if tableName != "" {
			r.newlyCreatedTables[tableName] = true
		}
	case *ast.IndexStmt:
		if !n.Concurrent {
			tableName := omniTableName(n.Relation)
			if r.newlyCreatedTables[tableName] {
				return
			}
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.CreateIndexUnconcurrently.Int32(),
				Title:   r.Title,
				Content: "Creating indexes will block writes on the table, unless use CONCURRENTLY",
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	case *ast.DropStmt:
		if n.RemoveType == int(ast.OBJECT_INDEX) && !n.Concurrent {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.DropIndexUnconcurrently.Int32(),
				Title:   r.Title,
				Content: "Droping indexes will block writes on the table, unless use CONCURRENTLY",
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	default:
	}
}
