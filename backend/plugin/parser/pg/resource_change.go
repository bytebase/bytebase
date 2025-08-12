package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_COCKROACHDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbSchema *model.DatabaseSchema, asts any, statement string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	changedResources := model.NewChangedResources(dbSchema)
	dmlCount := 0
	insertCount := 0
	var sampleDMLs []string
	searchPath := dbSchema.GetDatabaseMetadata().GetSearchPath()
	if len(searchPath) == 0 {
		searchPath = []string{"public"} // default search path for PostgreSQL
	}
	for _, node := range nodes {
		if n, ok := node.(*ast.VariableSetStmt); ok {
			if strings.EqualFold(n.Name, "search_path") {
				var err error
				searchPath, err = getSearchPathFromSQL(n.Text())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get search path from statement %q", n.Text())
				}
				if len(searchPath) == 0 {
					searchPath = []string{"public"} // default search path for PostgreSQL
				}
			}
		}

		// schema is "public" by default.
		collectResourceChanges(database, searchPath, node, statement, changedResources, dbSchema.GetDatabaseMetadata())

		switch node := node.(type) {
		case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
			if node, ok := node.(*ast.InsertStmt); ok && len(node.ValueList) > 0 {
				insertCount += len(node.ValueList)
				continue
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, node.Text())
			}
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         dmlCount,
		SampleDMLS:       sampleDMLs,
		InsertCount:      insertCount,
	}, nil
}

func collectResourceChanges(database string, searchPath []string, node ast.Node, statement string, changedResources *model.ChangedResources, databaseMetadata *model.DatabaseMetadata) {
	switch node := node.(type) {
	case *ast.CreateTableStmt:
		if node.Name.Type == ast.TableTypeBaseTable {
			d, s, table := node.Name.Database, node.Name.Schema, node.Name.Name
			if d == "" {
				d = database
			}
			if s == "" {
				s = searchPath[0] // default schema for PostgreSQL
			}
			changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
				false,
			)
		}
	case *ast.DropTableStmt:
		for _, table := range node.TableList {
			if table.Type == ast.TableTypeView {
				d, s, v := table.Database, table.Schema, table.Name
				if d == "" {
					d = database
				}
				if s == "" {
					schemaName, _ := databaseMetadata.SearchObject(searchPath, v)
					if schemaName == "" {
						s = searchPath[0] // default schema for PostgreSQL
					} else {
						s = schemaName
					}
				}
				changedResources.AddView(
					d,
					s,
					&storepb.ChangedResourceView{
						Name:   v,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
				)
			} else {
				d, s, table := table.Database, table.Schema, table.Name
				if d == "" {
					d = database
				}
				if s == "" {
					schemaName, _ := databaseMetadata.SearchObject(searchPath, table)
					if schemaName == "" {
						s = searchPath[0] // default schema for PostgreSQL
					} else {
						s = schemaName
					}
				}
				changedResources.AddTable(
					d,
					s,
					&storepb.ChangedResourceTable{
						Name:   table,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					true,
				)
			}
		}
	case *ast.AlterTableStmt:
		if node.Table.Type == ast.TableTypeBaseTable {
			d, s, table := node.Table.Database, node.Table.Schema, node.Table.Name
			if d == "" {
				d = database
			}
			if s == "" {
				schemaName, _ := databaseMetadata.SearchObject(searchPath, table)
				if schemaName == "" {
					s = searchPath[0] // default schema for PostgreSQL
				} else {
					s = schemaName
				}
			}
			changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
				true,
			)

			for _, item := range node.AlterItemList {
				if v, ok := item.(*ast.RenameTableStmt); ok {
					d, s, table := node.Table.Database, node.Table.Schema, v.NewName
					if d == "" {
						d = database
					}
					if s == "" {
						schemaName, _ := databaseMetadata.SearchObject(searchPath, table)
						if schemaName == "" {
							s = searchPath[0] // default schema for PostgreSQL
						} else {
							s = schemaName
						}
					}
					changedResources.AddTable(
						d,
						s,
						&storepb.ChangedResourceTable{
							Name:   table,
							Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
						},
						false,
					)
					// Only one rename table statement is allowed in a single alter table statement.
					break
				}
			}
		}
	case *ast.CreateIndexStmt:
		d, s, table := node.Index.Table.Database, node.Index.Table.Schema, node.Index.Table.Name
		if d == "" {
			d = database
		}
		if s == "" {
			schemaName, _ := databaseMetadata.SearchObject(searchPath, table)
			if schemaName == "" {
				s = searchPath[0] // default schema for PostgreSQL
			} else {
				s = schemaName
			}
		}
		changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name:   table,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
			false,
		)
	case *ast.DropIndexStmt:
		for _, index := range node.IndexList {
			if index.Table != nil {
				d, s, table := index.Table.Database, index.Table.Schema, index.Table.Name
				if d == "" {
					d = database
				}
				if s == "" {
					schemaName, _ := databaseMetadata.SearchObject(searchPath, table)
					if schemaName == "" {
						s = searchPath[0] // default schema for PostgreSQL
					} else {
						s = schemaName
					}
				}
				changedResources.AddTable(
					d,
					s,
					&storepb.ChangedResourceTable{
						Name:   table,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					false,
				)
			} else {
				schema, indexMetadata := databaseMetadata.SearchIndex(searchPath, index.Name)
				if indexMetadata == nil {
					continue
				}
				tableMetadata := indexMetadata.GetTableProto()
				if tableMetadata == nil {
					continue
				}
				changedResources.AddTable(
					database,
					schema,
					&storepb.ChangedResourceTable{
						Name:   tableMetadata.GetName(),
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					false,
				)
			}
		}
	case *ast.CreateViewStmt:
		d, s, view := node.Name.Database, node.Name.Schema, node.Name.Name
		if d == "" {
			d = database
		}
		if s == "" {
			s = searchPath[0] // default schema for PostgreSQL
		}
		changedResources.AddView(
			d,
			s,
			&storepb.ChangedResourceView{
				Name:   view,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
		)
	case *ast.CreateFunctionStmt:
		s, f := node.Function.Schema, node.Function.Name
		if s == "" {
			s = searchPath[0] // default schema for PostgreSQL
		}
		changedResources.AddFunction(
			database,
			s,
			&storepb.ChangedResourceFunction{
				Name:   f,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
		)
	case *ast.DropFunctionStmt:
		for _, ref := range node.FunctionList {
			s, f := ref.Schema, ref.Name
			if s == "" {
				schemaName, _ := databaseMetadata.SearchObject(searchPath, f)
				if schemaName == "" {
					s = searchPath[0] // default schema for PostgreSQL
				} else {
					s = schemaName
				}
			}
			changedResources.AddFunction(
				database,
				s,
				&storepb.ChangedResourceFunction{
					Name:   f,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
			)
		}
	case *ast.InsertStmt:
		s := node.Table.Schema
		if s == "" {
			schemaName, _ := databaseMetadata.SearchObject(searchPath, node.Table.Name)
			if schemaName == "" {
				s = searchPath[0] // default schema for PostgreSQL
			} else {
				s = schemaName
			}
		}
		changedResources.AddTable(
			database,
			s,
			&storepb.ChangedResourceTable{
				Name:   node.Table.Name,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
			false, // no need to add all table rows as affected rows for DML
		)
	case *ast.UpdateStmt:
		s := node.Table.Schema
		if s == "" {
			schemaName, _ := databaseMetadata.SearchObject(searchPath, node.Table.Name)
			if schemaName == "" {
				s = searchPath[0] // default schema for PostgreSQL
			} else {
				s = schemaName
			}
		}
		changedResources.AddTable(
			database,
			s,
			&storepb.ChangedResourceTable{
				Name:   node.Table.Name,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
			false, // no need to add all table rows as affected rows for DML
		)
	case *ast.DeleteStmt:
		s := node.Table.Schema
		if s == "" {
			schemaName, _ := databaseMetadata.SearchObject(searchPath, node.Table.Name)
			if schemaName == "" {
				s = searchPath[0] // default schema for PostgreSQL
			} else {
				s = schemaName
			}
		}
		changedResources.AddTable(
			database,
			s,
			&storepb.ChangedResourceTable{
				Name:   node.Table.Name,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
			false, // no need to add all table rows as affected rows for DML
		)
	}
}

func getSearchPathFromSQL(statement string) ([]string, error) {
	parseResult, err := ParsePostgreSQL(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement %q", statement)
	}

	if parseResult == nil {
		return nil, errors.Errorf("parse result is nil for statement %q", statement)
	}

	visitor := &searchPathVisitor{}
	antlr.ParseTreeWalkerDefault.Walk(visitor, parseResult.Tree)
	return visitor.searchPath, nil
}

type searchPathVisitor struct {
	*parser.BasePostgreSQLParserListener
	searchPath []string
}

func (v *searchPathVisitor) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	setRest := ctx.Set_rest()
	if setRest == nil {
		return
	}
	setRestMore := setRest.Set_rest_more()
	if setRestMore == nil {
		return
	}
	genericSet := setRestMore.Generic_set()
	if genericSet == nil {
		return
	}
	varName := genericSet.Var_name()
	if varName == nil {
		return
	}
	if len(varName.AllColid()) != 1 {
		return
	}
	name := NormalizePostgreSQLColid(varName.Colid(0))
	if !strings.EqualFold(name, "search_path") {
		return
	}
	var searchPath []string
	for _, value := range genericSet.Var_list().AllVar_value() {
		valueText := value.GetText()
		if strings.HasPrefix(valueText, "\"") && strings.HasSuffix(valueText, "\"") {
			// Remove the quotes from the schema name.
			valueText = strings.Trim(valueText, "\"")
		} else if strings.HasPrefix(valueText, "'") && strings.HasSuffix(valueText, "'") {
			// Remove the quotes from the schema name.
			valueText = strings.Trim(valueText, "'")
		} else {
			// For non-quoted schema names, we just return the lower string for PostgreSQL.
			valueText = strings.ToLower(valueText)
		}
		path := strings.TrimSpace(valueText)
		if model.IsSystemPath(path) {
			continue
		}
		searchPath = append(searchPath, path)
	}
	v.searchPath = searchPath
}
