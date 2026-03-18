package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_COCKROACHDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	searchPath := dbMetadata.GetSearchPath()
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}

	if len(asts) == 0 {
		return &base.ChangeSummary{
			ChangedResources: changedResources,
			DMLCount:         0,
			SampleDMLS:       []string{},
			InsertCount:      0,
		}, nil
	}

	var dmlCount, insertCount int
	var sampleDMLs []string

	for _, unifiedAST := range asts {
		omniAST, ok := unifiedAST.(*OmniAST)
		if !ok {
			return nil, errors.New("expected OmniAST for PostgreSQL")
		}
		if omniAST.Node == nil {
			continue
		}

		switch n := omniAST.Node.(type) {
		case *ast.VariableSetStmt:
			if strings.EqualFold(n.Name, "search_path") && n.Args != nil {
				var newSearchPath []string
				for _, arg := range n.Args.Items {
					if ac, ok := arg.(*ast.A_Const); ok {
						if s, ok := ac.Val.(*ast.String); ok {
							newSearchPath = append(newSearchPath, s.Str)
						}
					}
				}
				if len(newSearchPath) > 0 {
					searchPath = newSearchPath
				}
			}

		case *ast.CreateStmt:
			if n.Relation != nil {
				db, schema, table := extractRangeVarNames(n.Relation, database, searchPath)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.DropStmt:
			objType := ast.ObjectType(n.RemoveType)
			switch objType {
			case ast.OBJECT_INDEX:
				handleDropIndexOmni(n, database, searchPath, dbMetadata, changedResources)
			case ast.OBJECT_TABLE, ast.OBJECT_MATVIEW:
				handleDropTableOmni(n, database, searchPath, dbMetadata, changedResources)
			default:
			}

		case *ast.AlterTableStmt:
			if ast.ObjectType(n.ObjType) == ast.OBJECT_VIEW {
				continue
			}
			if n.Relation != nil {
				db, schema, table := extractRangeVarNames(n.Relation, database, searchPath)
				if schema == "" {
					schemaName, _ := dbMetadata.SearchObject(searchPath, table)
					if schemaName != "" {
						schema = schemaName
					}
				}
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.RenameStmt:
			if n.Relation != nil {
				db, schema, oldTableName := extractRangeVarNames(n.Relation, database, searchPath)
				if schema == "" {
					schemaName, _ := dbMetadata.SearchObject(searchPath, oldTableName)
					if schemaName != "" {
						schema = schemaName
					}
				}
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: oldTableName}, true)
				if n.Newname != "" {
					changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: n.Newname}, false)
				}
			}

		case *ast.IndexStmt:
			if n.Relation != nil {
				db, schema, table := extractRangeVarNames(n.Relation, database, searchPath)
				if schema == "" {
					schemaName, _ := dbMetadata.SearchObject(searchPath, table)
					if schemaName != "" {
						schema = schemaName
					}
				}
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.InsertStmt:
			if n.Relation != nil {
				db, schema, table := extractRangeVarNames(n.Relation, database, searchPath)
				if schema == "" {
					schemaName, _ := dbMetadata.SearchObject(searchPath, table)
					if schemaName != "" {
						schema = schemaName
					}
				}
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}
			// Count insert rows from VALUES
			if sel, ok := n.SelectStmt.(*ast.SelectStmt); ok && sel.ValuesLists != nil {
				insertCount += len(sel.ValuesLists.Items)
			}

		case *ast.UpdateStmt:
			if n.Relation != nil {
				db, schema, table := extractRangeVarNames(n.Relation, database, searchPath)
				if schema == "" {
					schemaName, _ := dbMetadata.SearchObject(searchPath, table)
					if schemaName != "" {
						schema = schemaName
					}
				}
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, getOmniStatementText(omniAST))
			}

		case *ast.DeleteStmt:
			if n.Relation != nil {
				db, schema, table := extractRangeVarNames(n.Relation, database, searchPath)
				if schema == "" {
					schemaName, _ := dbMetadata.SearchObject(searchPath, table)
					if schemaName != "" {
						schema = schemaName
					}
				}
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, getOmniStatementText(omniAST))
			}
		default:
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         dmlCount,
		SampleDMLS:       sampleDMLs,
		InsertCount:      insertCount,
	}, nil
}

// extractRangeVarNames extracts database, schema, table from a RangeVar with defaults.
func extractRangeVarNames(rv *ast.RangeVar, defaultDB string, searchPath []string) (string, string, string) {
	db := rv.Catalogname
	schema := rv.Schemaname
	table := rv.Relname
	if db == "" {
		db = defaultDB
	}
	if schema == "" && len(searchPath) > 0 {
		schema = searchPath[0]
	}
	return db, schema, table
}

// handleDropTableOmni handles DROP TABLE/MATERIALIZED VIEW.
func handleDropTableOmni(n *ast.DropStmt, database string, searchPath []string, dbMetadata *model.DatabaseMetadata, changedResources *model.ChangedResources) {
	if n.Objects == nil {
		return
	}
	for _, item := range n.Objects.Items {
		nameList, ok := item.(*ast.List)
		if !ok {
			continue
		}
		db, schema, name := extractNameListParts(nameList, database)
		if schema == "" {
			schemaName, _ := dbMetadata.SearchObject(searchPath, name)
			if schemaName == "" {
				if len(searchPath) > 0 {
					schema = searchPath[0]
				}
			} else {
				schema = schemaName
			}
		}
		changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: name}, true)
	}
}

// handleDropIndexOmni handles DROP INDEX.
func handleDropIndexOmni(n *ast.DropStmt, database string, searchPath []string, dbMetadata *model.DatabaseMetadata, changedResources *model.ChangedResources) {
	if n.Objects == nil {
		return
	}
	for _, item := range n.Objects.Items {
		nameList, ok := item.(*ast.List)
		if !ok {
			continue
		}
		db, schema, indexName := extractNameListParts(nameList, database)

		lookupPath := searchPath
		if schema != "" {
			lookupPath = []string{schema}
		}
		schemaName, indexMetadata := dbMetadata.SearchIndex(lookupPath, indexName)
		if indexMetadata != nil && schemaName != "" {
			tableProto := indexMetadata.GetTableProto()
			if tableProto != nil {
				changedResources.AddTable(db, schemaName, &storepb.ChangedResourceTable{Name: tableProto.GetName()}, false)
			}
		}
	}
}

// extractNameListParts extracts db, schema, name from a list of String nodes.
func extractNameListParts(nameList *ast.List, defaultDB string) (string, string, string) {
	var parts []string
	for _, item := range nameList.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	switch len(parts) {
	case 1:
		return defaultDB, "", parts[0]
	case 2:
		return defaultDB, parts[0], parts[1]
	case 3:
		return parts[0], parts[1], parts[2]
	default:
		return defaultDB, "", ""
	}
}

// getOmniStatementText returns the text of a statement from OmniAST, including semicolon.
func getOmniStatementText(omniAST *OmniAST) string {
	text := strings.TrimSpace(omniAST.Text)
	if !strings.HasSuffix(text, ";") {
		text += ";"
	}
	return text
}
