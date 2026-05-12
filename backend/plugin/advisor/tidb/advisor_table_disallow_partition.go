package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table partition.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		advice := checkStmtForPartition(ostmt, level, title)
		if advice != nil {
			adviceList = append(adviceList, advice)
		}
	}

	return adviceList, nil
}

// checkStmtForPartition returns at most ONE advice per top-level
// statement, mirroring pingcap-typed predecessor's break-after-first-
// match cardinality. Pre-omni pingcap matched `spec.Tp ==
// AlterTablePartition` which is the REPARTITION form only (ALTER
// TABLE t PARTITION BY ...) — that maps to omni's `ATPartitionBy`.
// Partition-management forms (ADD PARTITION, DROP PARTITION, etc.)
// are NOT covered by this rule in either era; long-standing
// pre-omni behavior preserved per invariant #10. Mysql analog
// (mysql/rule_table_disallow_partition.go) has identical scope.
func checkStmtForPartition(ostmt OmniStmt, level storepb.Advice_Status, title string) *storepb.Advice {
	text := ostmt.TrimmedText()
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Partitions != nil {
			return buildPartitionAdvice(level, title, text, ostmt.FirstTokenLine())
		}
	case *ast.AlterTableStmt:
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			if cmd.Type == ast.ATPartitionBy && cmd.PartitionBy != nil {
				return buildPartitionAdvice(level, title, text, ostmt.FirstTokenLine())
			}
		}
	default:
	}
	return nil
}

func buildPartitionAdvice(level storepb.Advice_Status, title, text string, line int) *storepb.Advice {
	return &storepb.Advice{
		Status:        level,
		Code:          advisorcode.CreateTablePartition.Int32(),
		Title:         title,
		Content:       fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", text),
		StartPosition: common.ConvertANTLRLineToPosition(line),
	}
}
