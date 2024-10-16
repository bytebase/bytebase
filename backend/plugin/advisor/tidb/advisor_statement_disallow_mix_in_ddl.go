package tidb

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
}

// StatementDisallowMixInDDLAdvisor is the advisor checking for no mixed DDL and DML.
type StatementDisallowMixInDDLAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementDisallowMixInDDLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	switch ctx.ChangeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
	default:
		return nil, nil
	}
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	var adviceList []*storepb.Advice
	for _, stmtNode := range root {
		if _, ok := stmtNode.(ast.DMLNode); ok {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Alter schema can only run DDL, \"%s\" is not DDL", stmtNode.Text()),
				Code:    advisor.StatementDisallowMixDDLDML.Int32(),
				StartPosition: &storepb.Position{
					Line: int32(stmtNode.OriginTextPosition()),
				},
			})
		}
	}

	return adviceList, nil
}
