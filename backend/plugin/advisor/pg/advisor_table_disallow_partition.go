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
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for partitioned
// tables. The judgment lives in omni (pg/v1review.TableDisallowPartition);
// this adapter renders the findings as advice.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for partitioned tables.
func (*TableDisallowPartitionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		// V1 anchored every finding on the statement text's first
		// non-whitespace line and quoted the trimmed statement text.
		text := &BaseRule{StmtText: stmt.Text}
		for _, f := range pgv1review.TableDisallowPartition([]ast.Node{node}) {
			if f.Code != v1review.TablePartitioned {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.CreateTablePartition.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table partition is forbidden, but %q creates", text.TrimmedStmtText()),
				StartPosition: &storepb.Position{Line: text.ContentStartLine() + int32(stmt.BaseLine())},
			})
		}
	}
	return adviceList, nil
}
