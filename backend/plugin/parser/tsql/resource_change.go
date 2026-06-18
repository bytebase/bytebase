package tsql

import (
	"strings"
	"unicode"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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

	var dmlCount, insertCount int
	var sampleDMLs []string

	addDML := func(text string) {
		dmlCount++
		if len(sampleDMLs) < common.MaximumLintExplainSize {
			sampleDMLs = append(sampleDMLs, trimStatement(text))
		}
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
			addDML(omniAST.Text)

		case *ast.UpdateStmt:
			if ref, ok := n.Relation.(*ast.TableRef); ok {
				addTable(ref, false)
			}
			addDML(omniAST.Text)

		case *ast.DeleteStmt:
			if ref, ok := n.Relation.(*ast.TableRef); ok {
				addTable(ref, false)
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
		SampleDMLS:       sampleDMLs,
		DMLCount:         dmlCount,
		InsertCount:      insertCount,
	}, nil
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
