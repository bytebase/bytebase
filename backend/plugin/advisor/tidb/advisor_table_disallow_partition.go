package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table
// partition. The judgment lives in omni (tidb/v1review.TableDisallowPartition);
// this adapter renders the findings as advice.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.TableDisallowPartition([]ast.Node{stmt.Node}) {
			if f.Code != v1review.TablePartitioned {
				continue
			}
			// V1 printed the whole statement and placed it on the
			// statement's first token.
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.CreateTablePartition.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", stmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.FirstTokenLine()),
			})
		}
	}

	return adviceList, nil
}
