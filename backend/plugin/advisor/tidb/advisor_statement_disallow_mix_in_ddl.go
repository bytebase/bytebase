package tidb

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
}

// StatementDisallowMixInDDLAdvisor is the advisor checking for no mixed DDL and DML.
type StatementDisallowMixInDDLAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementDisallowMixInDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
	default:
		return nil, nil
	}
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(checkCtx.Rule.Type)

	var adviceList []*storepb.Advice
	for _, stmtNode := range root {
		if _, ok := stmtNode.(ast.DMLNode); ok {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Alter schema can only run DDL, \"%s\" is not DDL", stmtNode.Text()),
				Code:          code.StatementDisallowMixDDLDML.Int32(),
				StartPosition: common.ConvertANTLRLineToPosition(stmtNode.OriginTextPosition()),
			})
		}
	}

	return adviceList, nil
}
