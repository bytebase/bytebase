package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK. The
// judgment lives in omni (tidb/v1review.TableRequirePK); this adapter passes
// the statements and the pre-change schema and renders the findings as advice.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	nodes := make([]ast.Node, len(stmts))
	for i, stmt := range stmts {
		nodes[i] = stmt.Node
	}
	// A nil schema makes omni panic where V1 did with no original metadata.
	var original *v1review.Schema
	if checkCtx.OriginalMetadata != nil {
		original = &v1review.Schema{
			Metadata:            checkCtx.OriginalMetadata.GetProto(),
			ObjectCaseSensitive: checkCtx.OriginalMetadata.GetIsObjectCaseSensitive(),
		}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.TableRequirePK(nodes, original) {
		if f.Code != v1review.TablePrimaryKeyMissing {
			continue
		}
		line := 0
		if f.Statement >= 0 {
			line = stmts[f.Statement].AbsoluteLine(f.Start)
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.TableNoPK.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Table `%s` requires PRIMARY KEY", f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(line),
		})
	}
	return adviceList, nil
}
