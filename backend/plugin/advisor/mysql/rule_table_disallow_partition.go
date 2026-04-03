package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table partition.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableDisallowPartitionOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowPartitionOmniRule struct {
	OmniBaseRule
}

func (*tableDisallowPartitionOmniRule) Name() string {
	return "TableDisallowPartitionRule"
}

func (r *tableDisallowPartitionOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *tableDisallowPartitionOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Partitions != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.CreateTablePartition.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", r.QueryText()),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
		})
	}
}

func (r *tableDisallowPartitionOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		if cmd.Type == ast.ATPartitionBy && cmd.PartitionBy != nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          advisorcode.CreateTablePartition.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", r.QueryText()),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
			return
		}
	}
}
