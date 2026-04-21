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
			default:
			}

		case *ast.CreateIndexStmt:
			addTable(n.Table, false)

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
