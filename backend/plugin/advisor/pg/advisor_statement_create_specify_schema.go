package pg

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementCreateSpecifySchema)(nil)
	_ ast.Visitor     = (*statementCreateSpecifySchemaChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLStatementCreateSpecifySchema, &StatementCreateSpecifySchema{})
}

type StatementCreateSpecifySchema struct {
}

func (*StatementCreateSpecifySchema) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementCreateSpecifySchemaChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type statementCreateSpecifySchemaChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

func (c *statementCreateSpecifySchemaChecker) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Name.Schema == "" {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName,
				Title:   c.title,
				Content: "Table schema should be specified.",
				Line:    node.LastLine(),
			})
		}
	case *ast.CreateExtensionStmt:
		if n.Schema == "" {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName,
				Title:   c.title,
				Content: "Extension schema should be specified.",
				Line:    node.LastLine(),
			})
		}
	case *ast.CreateFunctionStmt:
		if n.Function.Schema == "" {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName,
				Title:   c.title,
				Content: "Function schema should be specified.",
				Line:    node.LastLine(),
			})
		}
	case *ast.CreateIndexStmt:
		if n.Index.Table.Schema == "" {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName,
				Title:   c.title,
				Content: "Table schema for index should be specified.",
				Line:    node.LastLine(),
			})
		}
	case *ast.CreateSequenceStmt:
		if n.SequenceDef.SequenceName.Schema == "" {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName,
				Title:   c.title,
				Content: "Sequence schema should be specified.",
				Line:    node.LastLine(),
			})
		}
	case *ast.CreateTriggerStmt:
		if n.Trigger.Table.Schema == "" {
			c.adviceList = append(c.adviceList, advisor.Advice{
				Status:  c.level,
				Code:    advisor.StatementCreateWithoutSchemaName,
				Title:   c.title,
				Content: "Table schema for trigger should be specified.",
				Line:    node.LastLine(),
			})
		}
	}
	return c
}
