package mysql

import (
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MYSQL, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MARIADB, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_OCEANBASE, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_STARROCKS, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DORIS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)

	var dmlCount, insertCount int
	var sampleDMLs []string

	for _, unifiedAST := range asts {
		omniAST, ok := unifiedAST.(*OmniAST)
		if !ok {
			return nil, errors.New("expected OmniAST for MySQL")
		}
		if omniAST.Node == nil {
			continue
		}

		switch n := omniAST.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table != nil {
				db, table := omniTableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.DropTableStmt:
			for _, ref := range n.Tables {
				db, table := omniTableRef(ref, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.AlterTableStmt:
			if n.Table != nil {
				db, table := omniTableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.RenameTableStmt:
			for _, pair := range n.Pairs {
				if pair.Old != nil {
					db, table := omniTableRef(pair.Old, currentDatabase)
					changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
				}
				if pair.New != nil {
					db, table := omniTableRef(pair.New, currentDatabase)
					changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
				}
			}

		case *ast.CreateIndexStmt:
			if n.Table != nil {
				db, table := omniTableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.DropIndexStmt:
			if n.Table != nil {
				db, table := omniTableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.InsertStmt:
			if n.Table != nil {
				db, table := omniTableRef(n.Table, currentDatabase)
				changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, false)
			}
			if len(n.Values) > 0 {
				insertCount += len(n.Values)
				continue
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, omniStatementText(omniAST))
			}

		case *ast.UpdateStmt:
			for _, resource := range extractTableExprs(n.Tables, currentDatabase) {
				changedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, false)
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, omniStatementText(omniAST))
			}

		case *ast.DeleteStmt:
			for _, resource := range extractTableExprs(n.Tables, currentDatabase) {
				changedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, false)
			}
			for _, resource := range extractTableExprs(n.Using, currentDatabase) {
				changedResources.AddTable(resource.Database, "", &storepb.ChangedResourceTable{Name: resource.Table}, false)
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, omniStatementText(omniAST))
			}

		default:
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       sampleDMLs,
		DMLCount:         dmlCount,
		InsertCount:      insertCount,
	}, nil
}

// omniTableRef extracts database and table name from an omni TableRef.
func omniTableRef(ref *ast.TableRef, defaultDB string) (string, string) {
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
		db, table := omniTableRef(n, defaultDB)
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

// omniStatementText returns the text of a statement, ensuring it ends with a semicolon.
func omniStatementText(a *OmniAST) string {
	text := strings.TrimSpace(a.Text)
	if !strings.HasSuffix(text, ";") {
		text += ";"
	}
	return text
}
