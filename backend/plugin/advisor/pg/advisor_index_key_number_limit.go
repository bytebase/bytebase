package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
	_ ast.Visitor     = (*indexKeyNumberLimitChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLIndexKeyNumberLimit, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number limit.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &indexKeyNumberLimitChecker{
		level: level,
		title: string(ctx.Rule.Type),
		max:   payload.Number,
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

type indexKeyNumberLimitChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	max        int
}

func (checker *indexKeyNumberLimitChecker) getMetaDataList(in ast.Node) []*indexMetaData {
	var res []*indexMetaData

	switch node := in.(type) {
	case *ast.CreateIndexStmt:
		if checker.max > 0 && len(node.Index.KeyList) > checker.max {
			res = append(res, &indexMetaData{
				indexName: node.Index.Name,
				tableName: node.Index.Table.Name,
				line:      node.LastLine(),
			})
		}
	case *ast.CreateTableStmt:
		for _, constraint := range node.ConstraintList {
			if checker.max > 0 && indexKeyNumber(constraint) > checker.max {
				res = append(res, &indexMetaData{
					indexName: constraint.Name,
					tableName: node.Name.Name,
					line:      constraint.LastLine(),
				})
			}
		}
	case *ast.AddConstraintStmt:
		if checker.max > 0 && indexKeyNumber(node.Constraint) > checker.max {
			res = append(res, &indexMetaData{
				indexName: node.Constraint.Name,
				tableName: node.Table.Name,
				line:      node.LastLine(),
			})
		}
	}

	return res
}

func (checker *indexKeyNumberLimitChecker) Visit(node ast.Node) ast.Visitor {
	indexDataList := checker.getMetaDataList(node)

	for _, index := range indexDataList {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.IndexKeyNumberExceedsLimit,
			Title:   checker.title,
			Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", index.indexName, index.tableName, checker.max),
			Line:    index.line,
		})
	}
	return checker
}

func indexKeyNumber(constraint *ast.ConstraintDef) int {
	switch constraint.Type {
	case ast.ConstraintTypePrimary,
		ast.ConstraintTypeUnique,
		ast.ConstraintTypeForeign:
		return len(constraint.KeyList)
	default:
		return 0
	}
}
