package taskcheck

import (
	"context"
	"encoding/json"
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewStatementTypeReportExecutor creates a task check statement type report executor.
func NewStatementTypeReportExecutor(store *store.Store) Executor {
	return &StatementTypeReportExecutor{
		store: store,
	}
}

// StatementTypeReportExecutor is the task check statement type report executor. It reports the type of each statement.
type StatementTypeReportExecutor struct {
	store *store.Store
}

// Run will run the task check statement type report executor once.
func (s *StatementTypeReportExecutor) Run(ctx context.Context, _ *store.TaskCheckRunMessage, task *store.TaskMessage) ([]api.TaskCheckResult, error) {
	if !api.IsTaskCheckReportNeededForTaskType(task.Type) {
		return nil, nil
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	if !api.IsTaskCheckReportSupported(instance.Engine) {
		return nil, nil
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", payload.SheetID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", payload.SheetID)
	}
	if sheet.Size > common.MaxSheetSizeForTaskCheck {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.AdvisorNamespace,
				Code:      common.Ok.Int(),
				Title:     "Large SQL review policy is disabled",
				Content:   "",
			},
		}, nil
	}
	statement, err := s.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", payload.SheetID)
	}
	var charset, collation string
	if task.DatabaseID != nil {
		dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
		if err != nil {
			return nil, err
		}
		charset = dbSchema.Metadata.CharacterSet
		collation = dbSchema.Metadata.Collation
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	switch instance.Engine {
	case db.Postgres:
		return reportStatementTypeForPostgres(database.DatabaseName, renderedStatement)
	case db.MySQL, db.MariaDB, db.OceanBase, db.TiDB:
		return reportStatementTypeForMySQL(instance.Engine, database.DatabaseName, renderedStatement, charset, collation)
	case db.Oracle:
		schema := ""
		if instance.Options == nil || !instance.Options.SchemaTenantMode {
			adminSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
			schema = adminSource.Username
		} else {
			schema = database.DatabaseName
		}
		return reportStatementTypeForOracle(database.DatabaseName, schema, renderedStatement)
	default:
		return nil, errors.New("unsupported db type")
	}
}

func reportStatementTypeForOracle(databaseName string, schemaName string, statement string) ([]api.TaskCheckResult, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.Oracle, statement)
	if err != nil {
		// nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	var result []api.TaskCheckResult

	for _, stmt := range singleSQLs {
		if stmt.Empty || stmt.Text == "" {
			continue
		}

		// TODO: support report statement type for oracle
		sqlType := "UNKNOWN"
		changedResources, err := getChangedResourcesForOracle(databaseName, schemaName, stmt.Text)
		if err != nil {
			log.Error("failed to extract changed resources", zap.String("statement", stmt.Text), zap.Error(err))
		}
		result = append(result, api.TaskCheckResult{
			Status:           api.TaskCheckStatusSuccess,
			Namespace:        api.BBNamespace,
			Code:             common.Ok.Int(),
			Title:            "OK",
			Content:          sqlType,
			ChangedResources: string(changedResources),
		})
	}
	return result, nil
}

func getChangedResourcesForOracle(databaseName string, schemaName string, statement string) ([]byte, error) {
	resources, err := parser.ExtractChangedResources(parser.Oracle, databaseName, schemaName, statement)
	if err != nil {
		return nil, err
	}
	return marshalChangedResources(resources)
}

func reportStatementTypeForMySQL(engine db.Type, databaseName, statement, charset, collation string) ([]api.TaskCheckResult, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.MySQL, statement)
	if err != nil {
		// nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	var result []api.TaskCheckResult

	p := tidbparser.New()
	p.EnableWindowFunc(true)

	for _, stmt := range singleSQLs {
		if stmt.Empty {
			continue
		}
		if parser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "OK",
				Content:   "UNKNOWN",
			})
			continue
		}
		root, _, err := p.Parse(stmt.Text, charset, collation)
		if err != nil {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			})
			continue
		}
		if len(root) != 1 {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     "Failed to report statement type",
				Content:   "Expect to get one node from parser",
			})
			continue
		}
		sqlType, resources := getStatementTypeFromTidbAstNode(strings.ToLower(databaseName), root[0])
		var changedResources []byte
		if !isDML(sqlType) {
			if engine != db.TiDB {
				changedResources, err = getStatementChangedResourcesForMySQL(databaseName, stmt.Text)
			} else {
				changedResources, err = marshalChangedResources(resources)
			}
			if err != nil {
				log.Error("failed to get statement changed resources", zap.Error(err))
			}
		}

		result = append(result, api.TaskCheckResult{
			Status:           api.TaskCheckStatusSuccess,
			Namespace:        api.BBNamespace,
			Code:             common.Ok.Int(),
			Title:            "OK",
			Content:          sqlType,
			ChangedResources: string(changedResources),
		})
	}

	return result, nil
}

func isDML(tp string) bool {
	switch tp {
	case "REPLACE", "INSERT", "UPDATE", "DELETE":
		return true
	default:
		return false
	}
}

func marshalChangedResources(resources []parser.SchemaResource) ([]byte, error) {
	meta := storepb.ChangedResources{}
	// resources is ordered by (db, schema, table)
	for _, resource := range resources {
		if len(meta.Databases) == 0 || meta.Databases[len(meta.Databases)-1].Name != resource.Database {
			meta.Databases = append(meta.Databases, &storepb.ChangedResourceDatabase{Name: resource.Database})
		}
		database := meta.Databases[len(meta.Databases)-1]
		if len(database.Schemas) == 0 || database.Schemas[len(database.Schemas)-1].Name != resource.Schema {
			database.Schemas = append(database.Schemas, &storepb.ChangedResourceSchema{Name: resource.Schema})
		}
		schema := database.Schemas[len(database.Schemas)-1]
		schema.Tables = append(schema.Tables, &storepb.ChangedResourceTable{Name: resource.Table})
	}
	return protojson.Marshal(&meta)
}

func getStatementChangedResourcesForMySQL(currentDatabase, statement string) ([]byte, error) {
	resources, err := parser.ExtractChangedResources(parser.MySQL, currentDatabase, "" /* currentSchema */, statement)
	if err != nil {
		return nil, err
	}
	return marshalChangedResources(resources)
}

func reportStatementTypeForPostgres(database, statement string) ([]api.TaskCheckResult, error) {
	stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		// nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	var result []api.TaskCheckResult

	for _, stmt := range stmts {
		sqlType, resources := getStatementTypeAndResourcesFromAstNode(database, "public", stmt)
		if sqlType == "COMMENT" {
			resources, err = postgresExtractResourcesFromCommentStatement(database, "public", stmt.Text())
			if err != nil {
				return []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementSyntaxError.Int(),
						Title:     "Syntax error",
						Content:   err.Error(),
					},
				}, nil
			}
		}
		changedResource, err := marshalChangedResources(resources)
		if err != nil {
			log.Error("failed to marshal changed resources", zap.Error(err))
		}
		result = append(result, api.TaskCheckResult{
			Status:           api.TaskCheckStatusSuccess,
			Namespace:        api.BBNamespace,
			Code:             common.Ok.Int(),
			Title:            "OK",
			Content:          sqlType,
			ChangedResources: string(changedResource),
		})
	}

	return result, nil
}

func postgresExtractResourcesFromCommentStatement(database, defaultSchema, statement string) ([]parser.SchemaResource, error) {
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
					resource := parser.SchemaResource{
						Database: database,
						Schema:   schemaName,
						Table:    tableName,
					}
					if resource.Schema == "" {
						resource.Schema = defaultSchema
					}
					return []parser.SchemaResource{resource}, nil
				default:
					return nil, errors.Errorf("expect to get a list node but got %T", node)
				}
			case pgquery.ObjectType_OBJECT_TABCONSTRAINT:
				resource := parser.SchemaResource{
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
					return []parser.SchemaResource{resource}, nil
				default:
					return nil, errors.Errorf("expect to get a list node but got %T", node)
				}
			case pgquery.ObjectType_OBJECT_TABLE:
				resource := parser.SchemaResource{
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
					return []parser.SchemaResource{resource}, nil
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

func getStatementTypeFromTidbAstNode(database string, node tidbast.StmtNode) (string, []parser.SchemaResource) {
	var result []parser.SchemaResource
	switch n := node.(type) {
	// DDL

	// CREATE
	case *tidbast.CreateDatabaseStmt:
		return "CREATE_DATABASE", result
	case *tidbast.CreateIndexStmt:
		return "CREATE_INDEX", result
	case *tidbast.CreateTableStmt:
		resource := parser.SchemaResource{
			Database: n.Table.Schema.L,
			Table:    n.Table.Name.L,
		}
		if resource.Database == "" {
			resource.Database = database
		}
		result = append(result, resource)
		return "CREATE_TABLE", result
	case *tidbast.CreateViewStmt:
		return "CREATE_VIEW", result
	case *tidbast.CreateSequenceStmt:
		return "CREATE_SEQUENCE", result
	case *tidbast.CreatePlacementPolicyStmt:
		return "CREATE_PLACEMENT_POLICY", result

	// DROP
	case *tidbast.DropIndexStmt:
		return "DROP_INDEX", result
	case *tidbast.DropTableStmt:
		for _, table := range n.Tables {
			resource := parser.SchemaResource{
				Database: table.Schema.L,
				Table:    table.Name.L,
			}
			if resource.Database == "" {
				resource.Database = database
			}
			result = append(result, resource)
		}
		return "DROP_TABLE", result
	case *tidbast.DropSequenceStmt:
		return "DROP_SEQUENCE", result
	case *tidbast.DropPlacementPolicyStmt:
		return "DROP_PLACEMENT_POLICY", result
	case *tidbast.DropDatabaseStmt:
		return "DROP_DATABASE", result

	// ALTER
	case *tidbast.AlterTableStmt:
		resource := parser.SchemaResource{
			Database: n.Table.Schema.L,
			Table:    n.Table.Name.L,
		}
		if resource.Database == "" {
			resource.Database = database
		}
		result = append(result, resource)
		return "ALTER_TABLE", result
	case *tidbast.AlterSequenceStmt:
		return "ALTER_SEQUENCE", result
	case *tidbast.AlterPlacementPolicyStmt:
		return "ALTER_PLACEMENT_POLICY", result

	// TRUNCATE
	case *tidbast.TruncateTableStmt:
		return "TRUNCATE", result

	// RENAME
	case *tidbast.RenameTableStmt:
		for _, pair := range n.TableToTables {
			resource := parser.SchemaResource{
				Database: pair.OldTable.Schema.L,
				Table:    pair.OldTable.Name.L,
			}
			if resource.Database == "" {
				resource.Database = database
			}
			result = append(result, resource)

			newResource := parser.SchemaResource{
				Database: pair.NewTable.Schema.L,
				Table:    pair.NewTable.Name.L,
			}
			if newResource.Database == "" {
				newResource.Database = resource.Database
			}
			result = append(result, newResource)
		}
		return "RENAME_TABLE", result

	// DML

	case *tidbast.InsertStmt:
		if n.IsReplace {
			return "REPLACE", result
		}
		return "INSERT", result
	case *tidbast.DeleteStmt:
		return "DELETE", result
	case *tidbast.UpdateStmt:
		return "UPDATE", result
	}
	return "UNKNOWN", result
}

func getStatementTypeAndResourcesFromAstNode(database, schema string, node ast.Node) (string, []parser.SchemaResource) {
	result := []parser.SchemaResource{}
	switch node := node.(type) {
	// DDL

	// CREATE
	case *ast.CreateIndexStmt:
		return "CREATE_INDEX", result
	case *ast.CreateTableStmt:
		switch node.Name.Type {
		case ast.TableTypeView:
			return "CREATE_VIEW", result
		case ast.TableTypeBaseTable:
			resource := parser.SchemaResource{
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
			result = append(result, resource)
			return "CREATE_TABLE", result
		}
	case *ast.CreateSequenceStmt:
		return "CREATE_SEQUENCE", result
	case *ast.CreateDatabaseStmt:
		return "CREATE_DATABASE", result
	case *ast.CreateSchemaStmt:
		return "CREATE_SCHEMA", result
	case *ast.CreateFunctionStmt:
		return "CREATE_FUNCTION", result
	case *ast.CreateTriggerStmt:
		return "CREATE_TRIGGER", result
	case *ast.CreateTypeStmt:
		return "CREATE_TYPE", result
	case *ast.CreateExtensionStmt:
		return "CREATE_EXTENSION", result

	// DROP
	case *ast.DropColumnStmt:
		return "DROP_COLUMN", result
	case *ast.DropConstraintStmt:
		return "DROP_CONSTRAINT", result
	case *ast.DropDatabaseStmt:
		return "DROP_DATABASE", result
	case *ast.DropDefaultStmt:
		return "DROP_DEFAULT", result
	case *ast.DropExtensionStmt:
		return "DROP_EXTENSION", result
	case *ast.DropFunctionStmt:
		return "DROP_FUNCTION", result
	case *ast.DropIndexStmt:
		return "DROP_INDEX", result
	case *ast.DropNotNullStmt:
		return "DROP_NOT_NULL", result
	case *ast.DropSchemaStmt:
		return "DROP_SCHEMA", result
	case *ast.DropSequenceStmt:
		return "DROP_SEQUENCE", result
	case *ast.DropTableStmt:
		for _, table := range node.TableList {
			resource := parser.SchemaResource{
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
			result = append(result, resource)
		}
		return "DROP_TABLE", result

	case *ast.DropTriggerStmt:
		return "DROP_TRIGGER", result
	case *ast.DropTypeStmt:
		return "DROP_TYPE", result

	// ALTER
	case *ast.AlterColumnTypeStmt:
		return "ALTER_COLUMN_TYPE", result
	case *ast.AlterSequenceStmt:
		return "ALTER_SEQUENCE", result
	case *ast.AlterTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "ALTER_VIEW", result
		case ast.TableTypeBaseTable:
			resource := parser.SchemaResource{
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
			result = append(result, resource)
			return "ALTER_TABLE", result
		}
	case *ast.AlterTypeStmt:
		return "ALTER_TYPE", result

	case *ast.AddColumnListStmt:
		return "ALTER_TABLE_ADD_COLUMN_LIST", result
	case *ast.AddConstraintStmt:
		return "ALTER_TABLE_ADD_CONSTRAINT", result

	// RENAME
	case *ast.RenameColumnStmt:
		return "RENAME_COLUMN", result
	case *ast.RenameConstraintStmt:
		return "RENAME_CONSTRAINT", result
	case *ast.RenameIndexStmt:
		return "RENAME_INDEX", result
	case *ast.RenameSchemaStmt:
		return "RENAME_SCHEMA", result
	case *ast.RenameTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "RENAME_VIEW", result
		case ast.TableTypeBaseTable:
			resource := parser.SchemaResource{
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
			result = append(result, resource)

			newResource := parser.SchemaResource{
				Database: resource.Database,
				Schema:   resource.Schema,
				Table:    node.NewName,
			}
			result = append(result, newResource)
			return "RENAME_TABLE", result
		}

	case *ast.CommentStmt:
		return "COMMENT", result

	// DML

	case *ast.InsertStmt:
		return "INSERT", result
	case *ast.UpdateStmt:
		return "UPDATE", result
	case *ast.DeleteStmt:
		return "DELETE", result
	}

	return "UNKNOWN", result
}
