package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
	_ ast.Visitor     = (*tableRequirePKChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (*TableRequirePKAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableRequirePKChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		catalog: ctx.Catalog,
	}

	for _, stmt := range stmts {
		checker.text = stmt.Text()
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

type tableRequirePKChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
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
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status: checker.level,
			Code:   advisor.TableNoPK,
			Title:  checker.title,
			Content: fmt.Sprintf("Table %q.%q requires PRIMARY KEY, related statement: %q",
				normalizeSchemaName(missingPK.Schema),
				missingPK.Name,
				checker.text,
			),
			Line: node.LastLine(),
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
