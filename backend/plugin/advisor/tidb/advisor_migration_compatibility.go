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
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward
// compatibility. The judgment lives in omni
// (tidb/v1review.SchemaBackwardCompatibility); this adapter renders the
// findings as advice.
type CompatibilityAdvisor struct {
}

// compatibilityCodes maps each omni finding to the advice code V1 reported.
var compatibilityCodes = map[v1review.Code]advisorcode.Code{
	v1review.IncompatibleDropDatabase:  advisorcode.CompatibilityDropDatabase,
	v1review.IncompatibleRenameTable:   advisorcode.CompatibilityRenameTable,
	v1review.IncompatibleDropTable:     advisorcode.CompatibilityDropTable,
	v1review.IncompatibleRenameColumn:  advisorcode.CompatibilityRenameColumn,
	v1review.IncompatibleDropColumn:    advisorcode.CompatibilityDropColumn,
	v1review.IncompatibleAddPrimaryKey: advisorcode.CompatibilityAddPrimaryKey,
	v1review.IncompatibleAddUniqueKey:  advisorcode.CompatibilityAddUniqueKey,
	v1review.IncompatibleAddForeignKey: advisorcode.CompatibilityAddForeignKey,
	v1review.IncompatibleAddCheck:      advisorcode.CompatibilityAddCheck,
	v1review.IncompatibleAlterCheck:    advisorcode.CompatibilityAlterCheck,
	v1review.IncompatibleAlterColumn:   advisorcode.CompatibilityAlterColumn,
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// The rule judges a statement against the ones before it, so it takes
	// them all at once and names each finding's statement by index.
	nodes := make([]ast.Node, 0, len(stmts))
	for _, stmt := range stmts {
		nodes = append(nodes, stmt.Node)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.SchemaBackwardCompatibility(nodes) {
		adviceCode, ok := compatibilityCodes[f.Code]
		if !ok {
			continue
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", stmt.Text),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
		})
	}
	return adviceList, nil
}
