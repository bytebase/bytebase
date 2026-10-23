package tsql

import (
	"strings"
	"unicode"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MSSQL, extractChangedResources)
}

func extractChangedResources(currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)

	// The CTE check in resolveDMLWriteTarget matches names under the metadata's case
	// sensitivity, as prior backup does; nil metadata (tests, unsynced) is case-insensitive.
	caseSensitive := dbMetadata != nil && dbMetadata.GetIsObjectCaseSensitive()

	var dmlCount, insertCount int
	var dmlStatements []string

	addDML := func(text string) {
		dmlCount++
		dmlStatements = append(dmlStatements, trimStatement(text))
	}

	addTable := func(ref *ast.TableRef, affectData bool) {
		d, s, t := omniTableRefTriple(ref, currentDatabase, currentSchema)
		if t == "" {
			return
		}
		changedResources.AddTable(d, s, &storepb.ChangedResourceTable{Name: t}, affectData)
	}

	// addObjectDatabase records a database-only write target for a non-table object DDL
	// (view/procedure/function/trigger) ONLY when the object's own name carries an explicit
	// database qualifier. Unqualified names record nothing (request-database fallback). The
	// object's own qualifier is used — never an ON-table or body reference. SUP-222 / BYT-9698.
	addObjectDatabase := func(ref *ast.TableRef) {
		if ref != nil && ref.Database != "" {
			changedResources.AddDatabase(ref.Database)
		}
	}

	for _, unifiedAST := range asts {
		omniAST, ok := unifiedAST.(*OmniAST)
		if !ok {
			return nil, errors.New("expected OmniAST for MSSQL")
		}
		if omniAST.Node == nil {
			continue
		}

		switch n := omniAST.Node.(type) {
		case *ast.CreateTableStmt:
			addTable(n.Name, false)

		case *ast.AlterTableStmt:
			addTable(n.Name, true)

		case *ast.DropStmt:
			switch n.ObjectType {
			case ast.DropTable:
				if n.Names == nil {
					continue
				}
				for _, item := range n.Names.Items {
					if ref, ok := item.(*ast.TableRef); ok {
						addTable(ref, true)
					}
				}
			case ast.DropIndex:
				if n.OnTables == nil {
					continue
				}
				for _, item := range n.OnTables.Items {
					if ref, ok := item.(*ast.TableRef); ok {
						addTable(ref, false)
					}
				}
			case ast.DropView, ast.DropProcedure, ast.DropFunction, ast.DropTrigger,
				ast.DropSequence, ast.DropSynonym, ast.DropType:
				if n.Names == nil {
					continue
				}
				for _, item := range n.Names.Items {
					if ref, ok := item.(*ast.TableRef); ok {
						addObjectDatabase(ref)
					}
				}
			default:
			}

		case *ast.CreateIndexStmt:
			addTable(n.Table, false)

		case *ast.TruncateStmt:
			addTable(n.Table, true)

		case *ast.InsertStmt:
			if ref, ok := n.Relation.(*ast.TableRef); ok {
				addTable(ref, false)
			}
			if n.DefaultValues {
				insertCount++
				continue
			}
			if vc, ok := n.Source.(*ast.ValuesClause); ok {
				insertCount += vc.Rows.Len()
				continue
			}
			// SHOWPLAN estimates one row for INSERT ... EXEC whatever the procedure returns, so the
			// statement is counted without an estimate.
			if _, ok := n.Source.(*ast.ExecStmt); ok {
				dmlCount++
				continue
			}
			addDML(omniAST.Text)

		case *ast.UpdateStmt:
			if ref, ok := n.Relation.(*ast.TableRef); ok {
				addTable(resolveDMLWriteTarget(ref, n.FromClause, n.WithClause, caseSensitive), false)
			}
			addDML(omniAST.Text)

		case *ast.DeleteStmt:
			if ref, ok := n.Relation.(*ast.TableRef); ok {
				addTable(resolveDMLWriteTarget(ref, n.FromClause, n.WithClause, caseSensitive), false)
			}
			addDML(omniAST.Text)

		case *ast.MergeStmt:
			if ref, ok := n.Target.(*ast.TableRef); ok {
				addTable(ref, false)
			}
			addDML(omniAST.Text)

		case *ast.CreateTableAsSelectStmt:
			addTable(n.Name, false)

		case *ast.SelectStmt:
			// SELECT ... INTO new_table creates a table; the INTO target is the write
			// target (possibly on the first arm of a set operation). INTO #temp is a
			// session-scoped tempdb table, not a database change.
			if target := selectIntoTarget(n); target != nil && !strings.HasPrefix(target.Object, "#") {
				addTable(target, false)
			}

		case *ast.CreateViewStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateProcedureStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateFunctionStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateTriggerStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateSequenceStmt:
			addObjectDatabase(n.Name)
		case *ast.AlterSequenceStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateSynonymStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateTypeStmt:
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

// resolveDMLWriteTarget resolves an UPDATE/DELETE target that names a FROM-clause
// correlation to the aliased physical table, so table-scoped grants and approval rules
// see the real table. Only an unqualified target can be an alias; SQL Server rejects a
// correlation name that shares a qualified target's exposed name. The match is exact:
// synced metadata reports every SQL Server database as case-insensitive, even one with a
// case-sensitive collation, so folding would bind a target to a different table there.
// A differently cased alias (BOT-158) and a CTE or derived-table target (BOT-160) are
// left as written.
func resolveDMLWriteTarget(target *ast.TableRef, fromClause *ast.List, withClause *ast.WithClause, caseSensitive bool) *ast.TableRef {
	if target == nil || target.Schema != "" || target.Database != "" || fromClause == nil {
		return target
	}
	_, physicalRef := findPhysicalTableForAlias(fromClause, &TableReference{Table: target.Object})
	if physicalRef == nil || isCTETarget(physicalRef, withClause, caseSensitive) {
		return target
	}
	return physicalRef
}

func omniTableRefTriple(ref *ast.TableRef, defaultDatabase, defaultSchema string) (string, string, string) {
	if ref == nil {
		return defaultDatabase, defaultSchema, ""
	}
	db := ref.Database
	if db == "" {
		db = defaultDatabase
	}
	schema := ref.Schema
	if schema == "" {
		schema = defaultSchema
	}
	return db, schema, ref.Object
}

func trimStatement(statement string) string {
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace) + ";"
}
