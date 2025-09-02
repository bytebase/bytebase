package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
	_ ast.Visitor     = (*tableRequirePKChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableRequirePKChecker{
		level:   level,
		title:   string(checkCtx.Rule.Type),
		catalog: checkCtx.Catalog,
	}

	for _, stmt := range stmts {
		checker.text = stmt.Text()
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type tableRequirePKChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	catalog    *catalog.Finder
	text       string
}

// Visit implements the ast.Visitor interface.
func (checker *tableRequirePKChecker) Visit(node ast.Node) ast.Visitor {
	var missingPK *ast.TableDef
	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		hasPK := false
		for _, column := range n.ColumnList {
			if containPK(column.ConstraintList) {
				hasPK = true
			}
		}
		if containPK(n.ConstraintList) {
			hasPK = true
		}
		if !hasPK {
			missingPK = n.Name
		}
	// DROP CONSTRAINT
	case *ast.DropConstraintStmt:
		_, index := checker.catalog.Origin.FindIndex(&catalog.IndexFind{
			SchemaName: normalizeSchemaName(n.Table.Schema),
			TableName:  n.Table.Name,
			IndexName:  n.ConstraintName,
		})
		if index != nil && index.Primary() {
			missingPK = n.Table
		}
	// DROP COLUMN
	case *ast.DropColumnStmt:
		pk := checker.catalog.Origin.FindPrimaryKey(&catalog.PrimaryKeyFind{
			SchemaName: normalizeSchemaName(n.Table.Schema),
			TableName:  n.Table.Name,
		})
		if pk != nil {
			for _, column := range pk.ExpressionList() {
				if column == n.ColumnName {
					missingPK = n.Table
				}
			}
		}
	}

	if missingPK != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status: checker.level,
			Code:   advisor.TableNoPK.Int32(),
			Title:  checker.title,
			Content: fmt.Sprintf("Table %q.%q requires PRIMARY KEY, related statement: %q",
				normalizeSchemaName(missingPK.Schema),
				missingPK.Name,
				checker.text,
			),
			StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
		})
	}

	return checker
}

func containPK(list []*ast.ConstraintDef) bool {
	for _, cons := range list {
		if cons.Type == ast.ConstraintTypePrimary || cons.Type == ast.ConstraintTypePrimaryUsingIndex {
			return true
		}
	}
	return false
}
