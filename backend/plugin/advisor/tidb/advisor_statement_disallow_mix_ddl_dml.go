package tidb

import (
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

	var hasDDL, hasDML bool
	for _, stmtNode := range root {
		if _, ok := stmtNode.(ast.DDLNode); ok {
			hasDDL = true
		}
		if _, ok := stmtNode.(ast.DMLNode); ok {
			hasDML = true
		}
		if hasDDL && hasDML {
			break
		}
	}

	if hasDDL && hasDML {
		return []advisor.Advice{
			{
				Status:  level,
				Title:   title,
				Content: "Mixing DDL with DML is not allowed",
				Code:    advisor.StatementDisallowMixDDLDML,
			},
		}, nil
	}

	return []advisor.Advice{
		{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		},
	}, nil
}
