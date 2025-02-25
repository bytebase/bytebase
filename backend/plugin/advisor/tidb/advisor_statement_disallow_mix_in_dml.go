package tidb

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

// StatementDisallowMixInDMLAdvisor is the advisor checking for no mixed DDL and DML.
type StatementDisallowMixInDMLAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementDisallowMixInDMLAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	switch ctx.ChangeType {
	case storepb.PlanCheckRunConfig_DML:
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
		if _, ok := stmtNode.(ast.DDLNode); ok {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", stmtNode.Text()),
				Code:    advisor.StatementDisallowMixDDLDML.Int32(),
				StartPosition: &storepb.Position{
					Line: int32(stmtNode.OriginTextPosition()),
				},
			})
		}
	}

	return adviceList, nil
}
