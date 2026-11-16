package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

var (
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table
// partition. The judgment lives in omni
// (oceanbase/v1review.TableDisallowPartition); this adapter renders the
// findings as advice.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
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
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oceanbasev1review.TableDisallowPartition([]ast.Node{node}) {
			if f.Code != v1review.TablePartitioned {
				continue
			}
			// V1 printed the whole statement and placed it on the
			// statement's first non-blank line, which a zero-range finding
			// gives.
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.CreateTablePartition.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", strings.TrimSpace(stmt.Text)),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, v1review.Finding{}))),
			})
		}
	}
	return adviceList, nil
}
