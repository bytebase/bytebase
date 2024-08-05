package plancheck

import (
	"context"
	"database/sql"
	"fmt"

	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	pgquery "github.com/pganalyze/pg_query_go/v5"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewStatementReportExecutor creates a statement report executor.
func NewStatementReportExecutor(store *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementReportExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// StatementReportExecutor is the statement report executor.
type StatementReportExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

// Run runs the statement report executor.
func (e *StatementReportExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetUID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetUID)
	}
	if sheet.Size > common.MaxSheetCheckSize {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_WARNING,
				Code:    common.SizeExceeded.Int32(),
				Title:   "Report for large SQL is not supported",
				Content: "",
			},
		}, nil
	}
	statement := sheet.Statement

	isDML := config.ChangeDatabaseType == storepb.PlanCheckRunConfig_DML
	var databases []*store.DatabaseMessage
	if config.DatabaseGroupUid != nil {
		databaseGroup, err := e.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
			UID: config.DatabaseGroupUid,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database group %d", *config.DatabaseGroupUid)
		}
		if databaseGroup == nil {
			return nil, errors.Errorf("database group not found %d", *config.DatabaseGroupUid)
		}
		project, err := e.store.GetProjectV2(ctx, &store.FindProjectMessage{
			UID: &databaseGroup.ProjectUID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get project %d", databaseGroup.ProjectUID)
		}
		if project == nil {
			return nil, errors.Errorf("project not found %d", databaseGroup.ProjectUID)
		}

		allDatabases, err := e.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
		}
		matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get matched and unmatched databases in database group %q", databaseGroup.ResourceID)
		}
		if len(matchedDatabases) == 0 {
			return nil, errors.Errorf("no matched databases found in database group %q", databaseGroup.ResourceID)
		}
		databases = matchedDatabases
	} else {
		instanceUID := int(config.InstanceUid)
		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance not found UID %v", instanceUID)
		}
		database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &config.DatabaseName})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", config.DatabaseName)
		}
		if database == nil {
			return nil, errors.Errorf("database not found %q", config.DatabaseName)
		}
		databases = append(databases, database)
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, database := range databases {
		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance not found %q", database.InstanceID)
		}
		if !common.StatementReportEngines[instance.Engine] {
			return []*storepb.PlanCheckRunResult_Result{
				{
					Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
					Code:    common.Ok.Int32(),
					Title:   fmt.Sprintf("Statement report is not supported for %s", instance.Engine),
					Content: "",
				},
			}, nil
		}

		r, err := e.runReport(ctx, instance, database, statement, isDML)
		if err != nil {
			return nil, err
		}
		results = append(results, r...)
	}

	if len(results) == 0 {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Title:   "OK",
				Content: "",
				Code:    common.Ok.Int32(),
				Report:  nil,
			},
		}, nil
	}
	return results, nil
}

func (e *StatementReportExecutor) runReport(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, statement string, isDML bool) ([]*storepb.PlanCheckRunResult_Result, error) {
	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}
	if dbSchema.GetMetadata() == nil {
		return nil, errors.Errorf("database schema metadata not found: %d", database.UID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB := driver.GetDB()

		return reportForPostgres(ctx, e.sheetManager, sqlDB, database.DatabaseName, renderedStatement, dbSchema)
	case storepb.Engine_MYSQL, storepb.Engine_OCEANBASE:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB := driver.GetDB()

		return reportForMySQL(ctx, e.sheetManager, sqlDB, instance.Engine, database.DatabaseName, renderedStatement, dbSchema, isDML)
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
		return reportForOracle(e.sheetManager, database.DatabaseName, database.DatabaseName, renderedStatement, dbSchema)
	default:
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   "Not available",
				Content: fmt.Sprintf("Report is not supported for %s", instance.Engine),
			},
		}, nil
	}
}

func reportForOracle(sm *sheet.Manager, databaseName string, schemaName string, statement string, dbMetadata *model.DBSchema) ([]*storepb.PlanCheckRunResult_Result, error) {
	asts, advices := sm.GetASTsForChecks(storepb.Engine_ORACLE, statement)
	if len(advices) > 0 {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: advices[0].Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}
	nodes, ok := asts.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	var changedResources []base.SchemaResource
	resources, err := base.ExtractChangedResources(storepb.Engine_ORACLE, databaseName, schemaName, nodes)
	if err != nil {
		slog.Error("failed to extract changed resources", slog.String("statement", statement), log.BBError(err))
	} else {
		changedResources = append(changedResources, resources...)
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   nil,
					AffectedRows:     0,
					ChangedResources: convertToChangedResources(dbMetadata, changedResources),
				},
			},
		},
	}, nil
}

func reportForMySQL(ctx context.Context, sm *sheet.Manager, sqlDB *sql.DB, engine storepb.Engine, databaseName string, statement string, dbMetadata *model.DBSchema, isDML bool) ([]*storepb.PlanCheckRunResult_Result, error) {
	asts, advices := sm.GetASTsForChecks(storepb.Engine_MYSQL, statement)
	if len(advices) > 0 {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: advices[0].Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}
	nodes, ok := asts.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	sqlTypeSet := map[string]struct{}{}
	var totalAffectedRows int64
	var changedResources []base.SchemaResource

	explainCount := 0
	for i, node := range nodes {
		sqlType := mysqlparser.GetStatementType(node)
		sqlTypeSet[sqlType] = struct{}{}
		resources, err := base.ExtractChangedResources(storepb.Engine_MYSQL, databaseName, "" /* currentSchema */, node)
		if err != nil {
			slog.Error("failed to extract changed resources", slog.String("statement", statement), log.BBError(err))
		} else {
			changedResources = append(changedResources, resources...)
		}

		isExplained := false
		affectedRows, err := base.GetAffectedRows(ctx, engine, node, buildGetRowsCountByQueryForMySQL(sqlDB, engine, &isExplained), buildGetTableDataSizeFuncForMySQL(dbMetadata))
		if err != nil {
			slog.Error("failed to get affected rows for mysql", slog.String("database", databaseName), log.BBError(err))
		} else {
			totalAffectedRows += affectedRows
		}
		if isExplained {
			explainCount++
		}
		if isDML && explainCount >= common.MaximumLintExplainSize {
			totalAffectedRows = int64(len(nodes)) * (totalAffectedRows / int64(i+1))
			break
		}
	}

	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     int32(totalAffectedRows),
					ChangedResources: convertToChangedResources(dbMetadata, changedResources),
				},
			},
		},
	}, nil
}

func convertToChangedResources(dbMetadata *model.DBSchema, resources []base.SchemaResource) *storepb.ChangedResources {
	meta := &storepb.ChangedResources{}
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
		var tableRows int64
		if dbMetadata != nil && dbMetadata.GetDatabaseMetadata() != nil && dbMetadata.GetDatabaseMetadata().GetSchema(resource.Schema) != nil && dbMetadata.GetDatabaseMetadata().GetSchema(resource.Schema).GetTable(resource.Table) != nil {
			tableRows = dbMetadata.GetDatabaseMetadata().GetSchema(resource.Schema).GetTable(resource.Table).GetRowCount()
		}
		schema.Tables = append(schema.Tables, &storepb.ChangedResourceTable{
			Name:      resource.Table,
			TableRows: tableRows,
		})
	}
	return meta
}

func reportForPostgres(ctx context.Context, sm *sheet.Manager, sqlDB *sql.DB, database, statement string, dbMetadata *model.DBSchema) ([]*storepb.PlanCheckRunResult_Result, error) {
	asts, advices := sm.GetASTsForChecks(storepb.Engine_POSTGRES, statement)
	if len(advices) > 0 {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: advices[0].Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}
	nodes, ok := asts.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	sqlTypeSet := map[string]struct{}{}
	var totalAffectedRows int64
	var changedResources []base.SchemaResource

	for _, node := range nodes {
		var resources []base.SchemaResource
		sqlType, v := getStatementTypeAndResourcesFromAstNode(database, "public", node)
		if sqlType == "COMMENT" {
			v2, err := postgresExtractResourcesFromCommentStatement(database, "public", node.Text())
			if err != nil {
				slog.Error("failed to extract resources from comment statement", slog.String("statement", node.Text()), log.BBError(err))
			} else {
				resources = v2
			}
		} else {
			resources = v
		}
		sqlTypeSet[sqlType] = struct{}{}
		changedResources = append(changedResources, resources...)

		rowCount, err := getAffectedRowsForPostgres(ctx, sqlDB, dbMetadata.GetMetadata(), node)
		if err != nil {
			slog.Error("failed to get affected rows for postgres", slog.String("database", database), log.BBError(err))
		} else {
			totalAffectedRows += rowCount
		}
	}

	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int32(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     int32(totalAffectedRows),
					ChangedResources: convertToChangedResources(dbMetadata, changedResources),
				},
			},
		},
	}, nil
}

func postgresExtractResourcesFromCommentStatement(database, defaultSchema, statement string) ([]base.SchemaResource, error) {
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
					return []base.SchemaResource{resource}, nil
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
					return []base.SchemaResource{resource}, nil
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
					return []base.SchemaResource{resource}, nil
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

func getStatementTypeAndResourcesFromAstNode(database, schema string, node ast.Node) (string, []base.SchemaResource) {
	result := []base.SchemaResource{}
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
			result = append(result, resource)

			newResource := base.SchemaResource{
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
