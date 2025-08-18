package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
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

func (*StatementCreateSpecifySchema) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementCreateSpecifySchemaChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type statementCreateSpecifySchemaChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (c *statementCreateSpecifySchemaChecker) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Name.Schema == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:         c.title,
				Content:       "Table schema should be specified.",
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	case *ast.CreateExtensionStmt:
		if n.Schema == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:         c.title,
				Content:       "Extension schema should be specified.",
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	case *ast.CreateFunctionStmt:
		if n.Function.Schema == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:         c.title,
				Content:       "Function schema should be specified.",
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	case *ast.CreateIndexStmt:
		if n.Index.Table.Schema == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:         c.title,
				Content:       "Table schema for index should be specified.",
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	case *ast.CreateSequenceStmt:
		if n.SequenceDef.SequenceName.Schema == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:         c.title,
				Content:       "Sequence schema should be specified.",
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	case *ast.CreateTriggerStmt:
		if n.Trigger.Table.Schema == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          advisor.StatementCreateWithoutSchemaName.Int32(),
				Title:         c.title,
				Content:       "Table schema for trigger should be specified.",
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	}
	return c
}
