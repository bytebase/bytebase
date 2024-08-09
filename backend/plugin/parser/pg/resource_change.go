package pg

import (
	"sort"

	"github.com/pkg/errors"

	pgquery "github.com/pganalyze/pg_query_go/v5"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
}

func extractChangedResources(database string, schema string, asts any, statement string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	resourceChangeMap := make(map[string]*base.ResourceChange)
	dmlCount := 0
	insertCount := 0
	var sampleDMLs []string
	for _, node := range nodes {
		// schema is "public" by default.
		err := getResourceChanges(database, schema, node, statement, resourceChangeMap)
		if err != nil {
			return nil, err
		}

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

	var resourceChanges []*base.ResourceChange
	for _, change := range resourceChangeMap {
		resourceChanges = append(resourceChanges, change)
	}
	sort.Slice(resourceChanges, func(i, j int) bool {
		return resourceChanges[i].String() < resourceChanges[j].String()
	})
	return &base.ChangeSummary{
		ResourceChanges: resourceChanges,
		DMLCount:        dmlCount,
		SampleDMLS:      sampleDMLs,
		InsertCount:     insertCount,
	}, nil
}

func getResourceChanges(database, schema string, node ast.Node, statement string, resourceChangeMap map[string]*base.ResourceChange) error {
	switch node := node.(type) {
	case *ast.CreateTableStmt:
		if node.Name.Type == ast.TableTypeBaseTable {
			resource := base.SchemaResource{
				Database: node.Name.Database,
				Schema:   node.Name.Schema,
				Table:    node.Name.Name,
			}
			if resource.Database == "" {
				resource.Database = database
			}
			if resource.Schema == "" {
				resource.Schema = schema
			}

			putResourceChange(resourceChangeMap, &base.ResourceChange{
				Resource: resource,
				Ranges:   []base.Range{base.NewRange(statement, node.Text())},
			})
			return nil
		}
	case *ast.DropTableStmt:
		for _, table := range node.TableList {
			resource := base.SchemaResource{
				Database: table.Database,
				Schema:   table.Schema,
				Table:    table.Name,
			}
			if resource.Database == "" {
				resource.Database = database
			}
			if resource.Schema == "" {
				resource.Schema = schema
			}

			putResourceChange(resourceChangeMap, &base.ResourceChange{
				Resource:    resource,
				Ranges:      []base.Range{base.NewRange(statement, node.Text())},
				AffectTable: true,
			})
		}
		return nil
	case *ast.AlterTableStmt:
		if node.Table.Type == ast.TableTypeBaseTable {
			resource := base.SchemaResource{
				Database: node.Table.Database,
				Schema:   node.Table.Schema,
				Table:    node.Table.Name,
			}
			if resource.Database == "" {
				resource.Database = database
			}
			if resource.Schema == "" {
				resource.Schema = schema
			}

			putResourceChange(resourceChangeMap, &base.ResourceChange{
				Resource:    resource,
				Ranges:      []base.Range{base.NewRange(statement, node.Text())},
				AffectTable: true,
			})

			for _, item := range node.AlterItemList {
				if v, ok := item.(*ast.RenameTableStmt); ok {
					newResource := base.SchemaResource{
						Database: node.Table.Database,
						Schema:   node.Table.Schema,
						Table:    v.NewName,
					}
					if newResource.Database == "" {
						newResource.Database = database
					}
					if newResource.Schema == "" {
						newResource.Schema = schema
					}

					putResourceChange(resourceChangeMap, &base.ResourceChange{
						Resource: newResource,
						Ranges:   []base.Range{base.NewRange(statement, node.Text())},
					})
					break
				}
			}

			return nil
		}
	// Is this used?
	case *ast.RenameTableStmt:
		if node.Table.Type == ast.TableTypeBaseTable {
			resource := base.SchemaResource{
				Database: node.Table.Database,
				Schema:   node.Table.Schema,
				Table:    node.Table.Name,
			}
			if resource.Database == "" {
				resource.Database = database
			}
			if resource.Schema == "" {
				resource.Schema = schema
			}
			newResource := base.SchemaResource{
				Database: resource.Database,
				Schema:   resource.Schema,
				Table:    node.NewName,
			}

			putResourceChange(resourceChangeMap, &base.ResourceChange{
				Resource: resource,
				Ranges:   []base.Range{base.NewRange(statement, node.Text())},
			})
			putResourceChange(resourceChangeMap, &base.ResourceChange{
				Resource: newResource,
				Ranges:   []base.Range{base.NewRange(statement, node.Text())},
			})
			return nil
		}
	case *ast.CommentStmt:
		change, err := postgresExtractResourcesFromCommentStatement(database, schema, node.Text())
		if err != nil {
			return err
		}

		putResourceChange(resourceChangeMap, change)
		return nil
	}

	return nil
}

func putResourceChange(resourceChangeMap map[string]*base.ResourceChange, change *base.ResourceChange) {
	v, ok := resourceChangeMap[change.String()]
	if !ok {
		resourceChangeMap[change.String()] = change
		return
	}

	v.Ranges = append(v.Ranges, change.Ranges...)
	if change.AffectTable {
		v.AffectTable = true
	}
}

func postgresExtractResourcesFromCommentStatement(database, defaultSchema, statement string) (*base.ResourceChange, error) {
	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, err
	}
	if len(res.Stmts) != 1 {
		return nil, errors.New("expect to get one node from parser")
	}
	for _, stmt := range res.Stmts {
		if comment, ok := stmt.Stmt.Node.(*pgquery.Node_CommentStmt); ok {
			switch comment.CommentStmt.Objtype {
			case pgquery.ObjectType_OBJECT_COLUMN:
				switch node := comment.CommentStmt.Object.Node.(type) {
				case *pgquery.Node_List:
					schemaName, tableName, _, err := convertColumnName(node)
					if err != nil {
						return nil, err
					}
					resource := base.SchemaResource{
						Database: database,
						Schema:   schemaName,
						Table:    tableName,
					}
					if resource.Schema == "" {
						resource.Schema = defaultSchema
					}
					return &base.ResourceChange{Resource: resource}, nil
				default:
					return nil, errors.Errorf("expect to get a list node but got %T", node)
				}
			case pgquery.ObjectType_OBJECT_TABCONSTRAINT:
				resource := base.SchemaResource{
					Database: database,
					Schema:   defaultSchema,
				}
				switch node := comment.CommentStmt.Object.Node.(type) {
				case *pgquery.Node_List:
					schemaName, tableName, _, err := convertConstraintName(node)
					if err != nil {
						return nil, err
					}
					if schemaName != "" {
						resource.Schema = schemaName
					}
					resource.Table = tableName
					return &base.ResourceChange{Resource: resource}, nil
				default:
					return nil, errors.Errorf("expect to get a list node but got %T", node)
				}
			case pgquery.ObjectType_OBJECT_TABLE:
				resource := base.SchemaResource{
					Database: database,
					Schema:   defaultSchema,
				}
				switch node := comment.CommentStmt.Object.Node.(type) {
				case *pgquery.Node_List:
					schemaName, tableName, err := convertTableName(node)
					if err != nil {
						return nil, err
					}
					if schemaName != "" {
						resource.Schema = schemaName
					}
					resource.Table = tableName
					return &base.ResourceChange{Resource: resource}, nil
				default:
					return nil, errors.Errorf("expect to get a list node but got %T", node)
				}
			}
		}
	}
	return nil, nil
}

func convertNodeList(node *pgquery.Node_List) ([]string, error) {
	var list []string
	for _, item := range node.List.Items {
		switch s := item.Node.(type) {
		case *pgquery.Node_String_:
			list = append(list, s.String_.Sval)
		default:
			return nil, errors.Errorf("expect to get a string node but got %T", s)
		}
	}
	return list, nil
}

func convertTableName(node *pgquery.Node_List) (string, string, error) {
	list, err := convertNodeList(node)
	if err != nil {
		return "", "", err
	}
	switch len(list) {
	case 2:
		return list[0], list[1], nil
	case 1:
		return "", list[0], nil
	default:
		return "", "", errors.Errorf("expect to get 1 or 2 items but got %d", len(list))
	}
}

func convertConstraintName(node *pgquery.Node_List) (string, string, string, error) {
	list, err := convertNodeList(node)
	if err != nil {
		return "", "", "", err
	}
	switch len(list) {
	case 3:
		return list[0], list[1], list[2], nil
	case 2:
		return "", list[0], list[1], nil
	default:
		return "", "", "", errors.Errorf("expect to get 2 or 3 items but got %d", len(list))
	}
}

func convertColumnName(node *pgquery.Node_List) (string, string, string, error) {
	list, err := convertNodeList(node)
	if err != nil {
		return "", "", "", err
	}
	switch len(list) {
	case 3:
		return list[0], list[1], list[2], nil
	case 2:
		return "", list[0], list[1], nil
	default:
		return "", "", "", errors.Errorf("expect to get 2 or 3 items but got %d", len(list))
	}
}
