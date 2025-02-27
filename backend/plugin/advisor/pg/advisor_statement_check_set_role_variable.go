package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementCheckSetRoleVariable)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLStatementCheckSetRoleVariable, &StatementCheckSetRoleVariable{})
}

type StatementCheckSetRoleVariable struct {
}

func (*StatementCheckSetRoleVariable) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	variableSetStmts := []*ast.VariableSetStmt{}
	for _, stmt := range stmts {
		if n, ok := stmt.(*ast.VariableSetStmt); ok {
			variableSetStmts = append(variableSetStmts, n)
		} else {
			break
		}
	}
	hasSetRole := false
	for _, stmt := range variableSetStmts {
		if stmt.Name == "role" {
			hasSetRole = true
		}
	}

	if !hasSetRole {
		return []*storepb.Advice{{
			Status:  level,
			Code:    advisor.StatementCheckSetRoleVariable.Int32(),
			Title:   string(checkCtx.Rule.Type),
			Content: "No SET ROLE statement found.",
			StartPosition: &storepb.Position{
				Line: 1,
			},
		}}, nil
	}

	return nil, nil
}
