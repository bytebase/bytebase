package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
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
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &indexKeyNumberLimitChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
		max:   payload.Number,
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type indexKeyNumberLimitChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
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
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.IndexKeyNumberExceedsLimit.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", index.indexName, index.tableName, checker.max),
			StartPosition: common.ConvertPGParserLineToPosition(index.line),
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
