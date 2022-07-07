package pg

import (
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.Postgres, advisor.PostgreSQLTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (adv *TableRequirePKAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmts, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	_, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableRequirePKChecker{}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	return []advisor.Advice{}, nil
}

// TODO(rebelice): fill the implementation to check table PK.
type tableRequirePKChecker struct {
	// adviceList []advisor.Advice
	// level      advisor.Status
	// title      string
	// tables:  make(tablePK),
	// catalog catalog.Catalog
}

// Visit implements the ast.Visitor interface.
func (checker *tableRequirePKChecker) Visit(node ast.Node) ast.Visitor {
	return checker
}
