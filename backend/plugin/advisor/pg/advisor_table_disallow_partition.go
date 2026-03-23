package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for partitioned tables.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for partitioned tables.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableDisallowPartitionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowPartitionRule struct {
	OmniBaseRule
}

// Name returns the rule name.
func (*tableDisallowPartitionRule) Name() string {
	return "table.disallow-partition"
}

// OnStatement is called for each top-level statement AST node.
func (r *tableDisallowPartitionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *tableDisallowPartitionRule) handleCreateStmt(n *ast.CreateStmt) {
	if n.Partspec != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.CreateTablePartition.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Table partition is forbidden, but %q creates", r.TrimmedStmtText()),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *tableDisallowPartitionRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AttachPartition {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.CreateTablePartition.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Table partition is forbidden, but %q creates", r.TrimmedStmtText()),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
			return
		}
	}
}
