package cockroachdb

import (
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_COCKROACHDB, extractChangedResources)
}

// extractChangedResources resolves an unqualified table to the current database and to currentSchema,
// or to "public" when currentSchema is empty.
func extractChangedResources(database string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	if currentSchema == "" {
		currentSchema = "public"
	}
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	addTable := func(name *tree.TableName, affectedTable bool) {
		db, schema, table := resolveTableName(name, database, currentSchema)
		summary.ChangedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, affectedTable)
	}
	// addMutations records the target of each mutation in the statement and returns their number.
	addMutations := func(stmt tree.Statement) int {
		targets := getMutationTargets(stmt)
		for _, target := range targets {
			if name, ok := getTableName(target); ok {
				addTable(name, false)
			}
		}
		return len(targets)
	}
	addDML := func(crdbAST *AST) {
		summary.DMLCount++
		summary.DMLStatements = append(summary.DMLStatements, strings.TrimSpace(crdbAST.Stmt.SQL))
	}

	for _, ast := range asts {
		crdbAST, ok := ast.(*AST)
		if !ok {
			return nil, errors.New("expected CockroachDB AST")
		}
		switch n := crdbAST.Stmt.AST.(type) {
		case *tree.Insert:
			mutationCount := addMutations(n)
			// Nested mutations make the whole statement a sample, whose estimate includes the VALUES rows.
			if rows, ok := getInsertValuesRowCount(n); ok && mutationCount == 1 {
				summary.InsertCount += rows
			} else {
				addDML(crdbAST)
			}
		case *tree.Update, *tree.Delete:
			addMutations(n)
			addDML(crdbAST)
		case *tree.Select:
			if addMutations(n) > 0 {
				addDML(crdbAST)
			}
		case *tree.CreateTable:
			addTable(&n.Table, false)
			if n.AsSource != nil && addMutations(n.AsSource) > 0 {
				addDML(crdbAST)
			}
		case *tree.AlterTable:
			name := n.Table.ToTableName()
			addTable(&name, true)
		case *tree.DropTable:
			for i := range n.Names {
				addTable(&n.Names[i], true)
			}
		case *tree.Truncate:
			for i := range n.Tables {
				addTable(&n.Tables[i], true)
			}
		case *tree.RenameTable:
			if n.IsSequence || (n.IsView && !n.IsMaterialized) {
				continue
			}
			name := n.Name.ToTableName()
			db, schema, table := resolveTableName(&name, database, currentSchema)
			summary.ChangedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, true)
			// An unqualified new name stays in the schema of the renamed table.
			newName := n.NewName.ToTableName()
			newDB, newSchema, newTable := resolveTableName(&newName, db, schema)
			summary.ChangedResources.AddTable(newDB, newSchema, &storepb.ChangedResourceTable{Name: newTable}, false)
		case *tree.CreateIndex:
			addTable(&n.Table, false)
		default:
		}
	}
	return summary, nil
}

func resolveTableName(name *tree.TableName, defaultDatabase string, defaultSchema string) (string, string, string) {
	database, schema := defaultDatabase, defaultSchema
	if name.ExplicitCatalog {
		database = name.Catalog()
	}
	if name.ExplicitSchema {
		schema = name.Schema()
	}
	return database, schema, name.Table()
}

// getTableName returns the table name of a mutation target, which is false for a numeric table
// reference.
func getTableName(target tree.TableExpr) (*tree.TableName, bool) {
	if aliased, ok := target.(*tree.AliasedTableExpr); ok {
		target = aliased.Expr
	}
	name, ok := target.(*tree.TableName)
	return name, ok
}

// getInsertValuesRowCount returns the rows of INSERT ... VALUES and INSERT ... DEFAULT VALUES; ok is
// false when a query supplies the rows.
func getInsertValuesRowCount(insert *tree.Insert) (int, bool) {
	if insert.Rows == nil || insert.Rows.Select == nil {
		return 1, true
	}
	rows := insert.Rows.Select
	for {
		paren, ok := rows.(*tree.ParenSelect)
		if !ok || paren.Select == nil {
			break
		}
		rows = paren.Select.Select
	}
	if values, ok := rows.(*tree.ValuesClause); ok {
		return len(values.Rows), true
	}
	return 0, false
}

// getMutationTargets returns the target of every INSERT, UPSERT, UPDATE, and DELETE in stmt: stmt
// itself, its CTEs, and statement sources such as [DELETE FROM t RETURNING id] at any depth.
func getMutationTargets(stmt tree.Statement) []tree.TableExpr {
	c := &mutationCollector{}
	c.statement(stmt)
	return c.targets
}

// getMutationTypes returns the statement type of each mutation getMutationTargets finds, in order.
func getMutationTypes(stmt tree.Statement) []storepb.StatementType {
	c := &mutationCollector{}
	c.statement(stmt)
	return c.types
}

type mutationCollector struct {
	targets []tree.TableExpr
	types   []storepb.StatementType
}

func (c *mutationCollector) statement(stmt tree.Statement) {
	switch n := stmt.(type) {
	case *tree.Insert:
		c.targets = append(c.targets, n.Table)
		c.types = append(c.types, storepb.StatementType_INSERT)
		c.with(n.With)
		c.selectQuery(n.Rows)
		if n.OnConflict != nil {
			c.updateExprs(n.OnConflict.Exprs)
			c.where(n.OnConflict.Where)
			c.expr(n.OnConflict.ArbiterPredicate)
		}
		c.returning(n.Returning)
	case *tree.Update:
		c.targets = append(c.targets, n.Table)
		c.types = append(c.types, storepb.StatementType_UPDATE)
		c.with(n.With)
		c.updateExprs(n.Exprs)
		c.tableExprs(n.From)
		c.where(n.Where)
		c.orderBy(n.OrderBy)
		c.limit(n.Limit)
		c.returning(n.Returning)
	case *tree.Delete:
		c.targets = append(c.targets, n.Table)
		c.types = append(c.types, storepb.StatementType_DELETE)
		c.with(n.With)
		c.tableExprs(n.Using)
		c.where(n.Where)
		c.orderBy(n.OrderBy)
		c.limit(n.Limit)
		c.returning(n.Returning)
	case *tree.Select:
		c.selectQuery(n)
	case tree.SelectStatement:
		c.selectStatement(n)
	default:
	}
}

func (c *mutationCollector) selectQuery(sel *tree.Select) {
	if sel == nil {
		return
	}
	c.with(sel.With)
	c.selectStatement(sel.Select)
	c.orderBy(sel.OrderBy)
	c.limit(sel.Limit)
}

func (c *mutationCollector) selectStatement(stmt tree.SelectStatement) {
	switch n := stmt.(type) {
	case *tree.ParenSelect:
		c.selectQuery(n.Select)
	case *tree.SelectClause:
		for _, selectExpr := range n.Exprs {
			c.expr(selectExpr.Expr)
		}
		c.tableExprs(n.From.Tables)
		c.where(n.Where)
		c.exprs(n.GroupBy)
		c.where(n.Having)
		c.exprs(n.DistinctOn)
		for _, window := range n.Window {
			c.exprs(window.Partitions)
			c.orderBy(window.OrderBy)
		}
	case *tree.UnionClause:
		c.selectQuery(n.Left)
		c.selectQuery(n.Right)
	case *tree.ValuesClause:
		for _, row := range n.Rows {
			c.exprs(row)
		}
	default:
	}
}

func (c *mutationCollector) with(with *tree.With) {
	if with == nil {
		return
	}
	for _, cte := range with.CTEList {
		c.statement(cte.Stmt)
	}
}

func (c *mutationCollector) tableExprs(exprs tree.TableExprs) {
	for _, expr := range exprs {
		c.tableExpr(expr)
	}
}

func (c *mutationCollector) tableExpr(expr tree.TableExpr) {
	switch n := expr.(type) {
	case *tree.AliasedTableExpr:
		c.tableExpr(n.Expr)
	case *tree.ParenTableExpr:
		c.tableExpr(n.Expr)
	case *tree.JoinTableExpr:
		c.tableExpr(n.Left)
		c.tableExpr(n.Right)
		if cond, ok := n.Cond.(*tree.OnJoinCond); ok {
			c.expr(cond.Expr)
		}
	case *tree.RowsFromExpr:
		c.exprs(n.Items)
	case *tree.Subquery:
		c.selectStatement(n.Select)
	case *tree.StatementSource:
		c.statement(n.Statement)
	default:
	}
}

func (c *mutationCollector) expr(expr tree.Expr) {
	if expr == nil {
		return
	}
	// The visit function never fails, so neither does the visit.
	_, _ = tree.SimpleVisit(expr, func(e tree.Expr) (bool, tree.Expr, error) {
		if subquery, ok := e.(*tree.Subquery); ok {
			c.selectStatement(subquery.Select)
			return false, e, nil
		}
		return true, e, nil
	})
}

func (c *mutationCollector) exprs(exprs []tree.Expr) {
	for _, expr := range exprs {
		c.expr(expr)
	}
}

func (c *mutationCollector) where(where *tree.Where) {
	if where != nil {
		c.expr(where.Expr)
	}
}

func (c *mutationCollector) orderBy(orderBy tree.OrderBy) {
	for _, order := range orderBy {
		c.expr(order.Expr)
	}
}

func (c *mutationCollector) limit(limit *tree.Limit) {
	if limit != nil {
		c.expr(limit.Count)
		c.expr(limit.Offset)
	}
}

func (c *mutationCollector) updateExprs(exprs tree.UpdateExprs) {
	for _, updateExpr := range exprs {
		c.expr(updateExpr.Expr)
	}
}

func (c *mutationCollector) returning(returning tree.ReturningClause) {
	if exprs, ok := returning.(*tree.ReturningExprs); ok {
		for _, selectExpr := range *exprs {
			c.expr(selectExpr.Expr)
		}
	}
}
