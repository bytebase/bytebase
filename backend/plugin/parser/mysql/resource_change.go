package mysql

import (
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MYSQL, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)

	var dmlCount, insertCount int
	var dmlStatements []string

	// addObjectDatabase records a database-only write target for a non-table object DDL
	// (view/function/routine/trigger) ONLY when the object's own name carries an explicit
	// database qualifier (TableRef.Schema). Unqualified names record nothing (request-database
	// fallback). The object's own qualifier is used — never an ON-table reference. SUP-222.
	addObjectDatabase := func(ref *ast.TableRef) {
		if ref != nil && ref.Schema != "" {
			changedResources.AddDatabase(ref.Schema)
		}
	}

	for _, unifiedAST := range asts {
		node, ok := unifiedAST.(*AST)
		if !ok {
			return nil, errors.New("expected AST for MySQL")
		}
		if node.Node == nil {
			continue
		}

		switch n := node.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table != nil {
				db, table := tableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.DropTableStmt:
			for _, ref := range n.Tables {
				db, table := tableRef(ref, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.TruncateStmt:
			for _, ref := range n.Tables {
				db, table := tableRef(ref, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.LoadDataStmt:
			if n.Table != nil {
				db, table := tableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.AlterTableStmt:
			if n.Table != nil {
				db, table := tableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.RenameTableStmt:
			for _, pair := range n.Pairs {
				if pair.Old != nil {
					db, table := tableRef(pair.Old, currentDatabase)
					changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
				}
				if pair.New != nil {
					db, table := tableRef(pair.New, currentDatabase)
					changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
				}
			}

		case *ast.CreateIndexStmt:
			if n.Table != nil {
				db, table := tableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.DropIndexStmt:
			if n.Table != nil {
				db, table := tableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.InsertStmt:
			if n.Table != nil {
				db, table := tableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}
			if len(n.Values) > 0 {
				insertCount += len(n.Values)
				continue
			}
			// INSERT ... SET inserts exactly one row, and EXPLAIN gives no estimate for it.
			if len(n.SetList) > 0 {
				insertCount++
				continue
			}
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(node))

		case *ast.UpdateStmt:
			for _, resource := range extractTableExprs(n.Tables, currentDatabase) {
				changedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, false)
			}
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(node))

		case *ast.DeleteStmt:
			for _, resource := range extractTableExprs(n.Tables, currentDatabase) {
				changedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, false)
			}
			for _, resource := range extractTableExprs(n.Using, currentDatabase) {
				changedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, false)
			}
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(node))

		// Non-table object DDL: gate by the object's own explicit database qualifier only.
		// CREATE TRIGGER is intentionally skipped — its AST name is unqualified (a bare string);
		// inferring the database from the ON table is out of scope. SUP-222 / BYT-9698.
		case *ast.CreateViewStmt:
			addObjectDatabase(n.Name)
		case *ast.AlterViewStmt:
			addObjectDatabase(n.Name)
		case *ast.DropViewStmt:
			for _, ref := range n.Views {
				addObjectDatabase(ref)
			}
		case *ast.CreateFunctionStmt:
			addObjectDatabase(n.Name)
		case *ast.AlterRoutineStmt:
			addObjectDatabase(n.Name)
		case *ast.DropRoutineStmt:
			addObjectDatabase(n.Name)
		case *ast.DropTriggerStmt:
			addObjectDatabase(n.Name)

		default:
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLStatements:    dmlStatements,
		DMLCount:         dmlCount,
		InsertCount:      insertCount,
	}, nil
}

// tableRef extracts database and table name from an omni TableRef.
func tableRef(ref *ast.TableRef, defaultDB string) (string, string) {
	if ref == nil {
		return defaultDB, ""
	}
	db := ref.Schema
	if db == "" {
		db = defaultDB
	}
	return db, ref.Name
}

// extractTableExprs collects schema resources from a slice of TableExpr.
func extractTableExprs(exprs []ast.TableExpr, defaultDB string) []base.SchemaResource {
	var result []base.SchemaResource
	for _, expr := range exprs {
		result = append(result, extractTableExpr(expr, defaultDB)...)
	}
	return result
}

// extractTableExpr recursively extracts schema resources from a single TableExpr.
func extractTableExpr(expr ast.TableExpr, defaultDB string) []base.SchemaResource {
	if expr == nil {
		return nil
	}
	switch n := expr.(type) {
	case *ast.TableRef:
		db, table := tableRef(n, defaultDB)
		return []base.SchemaResource{{Database: db, Table: table}}
	case *ast.JoinClause:
		var res []base.SchemaResource
		res = append(res, extractTableExpr(n.Left, defaultDB)...)
		res = append(res, extractTableExpr(n.Right, defaultDB)...)
		return res
	default:
		return nil
	}
}

// statementText returns the text of a statement, ensuring it ends with a semicolon.
func statementText(a *AST) string {
	text := strings.TrimSpace(a.Text)
	if !strings.HasSuffix(text, ";") {
		text += ";"
	}
	return text
}
