package tidb

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixDDLDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLStatementDisallowMixDDLDML, &StatementDisallowMixDDLDMLAdvisor{})
}

// StatementDisallowMixDDLDMLAdvisor is the advisor checking for no mixed DDL and DML.
type StatementDisallowMixDDLDMLAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementDisallowMixDDLDMLAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	var adviceList []advisor.Advice
	var hasDDL, hasDML bool
	for _, stmtNode := range root {
		var isDDL, isDML bool
		if _, ok := stmtNode.(ast.DDLNode); ok {
			isDDL = true
		}
		if _, ok := stmtNode.(ast.DMLNode); ok {
			isDML = true
		}

		switch ctx.ChangeType {
		case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
			if isDML {
				adviceList = append(adviceList, advisor.Advice{
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Alter schema can only run DDL, \"%s\" is not DDL", stmtNode.Text()),
					Code:    advisor.StatementDisallowMixDDLDML,
					Line:    stmtNode.OriginTextPosition(),
				})
			}
		case storepb.PlanCheckRunConfig_DML:
			if isDDL {
				adviceList = append(adviceList, advisor.Advice{
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", stmtNode.Text()),
					Code:    advisor.StatementDisallowMixDDLDML,
					Line:    stmtNode.OriginTextPosition(),
				})
			}
		}
		if isDDL {
			hasDDL = true
		}
		if isDML {
			hasDML = true
		}
		if hasDDL && hasDML {
			break
		}
	}

	if hasDDL && hasDML {
		adviceList = append(adviceList, advisor.Advice{
			Status:  level,
			Title:   title,
			Content: "Mixing DDL with DML is not allowed",
			Code:    advisor.StatementDisallowMixDDLDML,
		})
	}

	if len(adviceList) == 0 {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return adviceList, nil
}
