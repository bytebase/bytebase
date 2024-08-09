package pg

import (
	"sort"

	"github.com/pkg/errors"

	pgquery "github.com/pganalyze/pg_query_go/v5"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
}

func extractChangedResources(database string, schema string, asts any, _ string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	resourceChangeMap := make(map[string]*base.ResourceChange)
	for _, node := range nodes {
		// schema is "public" by default.
		changes, err := getResourceChanges(database, schema, node)
		if err != nil {
			return nil, err
		}
		for _, change := range changes {
			resourceChangeMap[change.Resource.String()] = change
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
	}, nil
}

func getResourceChanges(database, schema string, node ast.Node) ([]*base.ResourceChange, error) {
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
			return []*base.ResourceChange{{Resource: resource}}, nil
		}
	case *ast.DropTableStmt:
		var result []*base.ResourceChange
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
			result = append(result, &base.ResourceChange{Resource: resource})
		}
		return result, nil
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
			result := []*base.ResourceChange{{Resource: resource}}

			for _, item := range node.AlterItemList {
				if v, ok := item.(*ast.RenameTableStmt); ok {
					resource := base.SchemaResource{
						Database: node.Table.Database,
						Schema:   node.Table.Schema,
						Table:    v.NewName,
					}
					if resource.Database == "" {
						resource.Database = database
					}
					if resource.Schema == "" {
						resource.Schema = schema
					}
					result = append(result, &base.ResourceChange{Resource: resource})
				}
			}

			return result, nil
		}
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
			return []*base.ResourceChange{
				{Resource: resource},
				{Resource: newResource},
			}, nil
		}
	case *ast.CommentStmt:
		return postgresExtractResourcesFromCommentStatement(database, "public", node.Text())
	}

	return nil, nil
}

func postgresExtractResourcesFromCommentStatement(database, defaultSchema, statement string) ([]*base.ResourceChange, error) {
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
					return []*base.ResourceChange{{Resource: resource}}, nil
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
					return []*base.ResourceChange{{Resource: resource}}, nil
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
					return []*base.ResourceChange{{Resource: resource}}, nil
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
