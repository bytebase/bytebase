package plancheck

import (
	"context"
	"database/sql"
	"fmt"

	"log/slog"

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
	statement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, err
	}

	instanceUID := int(config.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
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

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &config.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", config.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", config.DatabaseName)
	}

	results, err := e.runReport(ctx, instance, database, statement)
	if err != nil {
		return nil, err
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

func (e *StatementReportExecutor) runReport(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, statement string) ([]*storepb.PlanCheckRunResult_Result, error) {
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

		return reportForMySQL(ctx, e.sheetManager, sqlDB, instance.Engine, database.DatabaseName, renderedStatement, dbSchema)
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
		advice := advices[0]
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   advice.Title,
				Content: advice.Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:          advice.GetStartPosition().GetLine(),
						Column:        advice.GetStartPosition().GetColumn(),
						Code:          advice.Code,
						Detail:        advice.Detail,
						StartPosition: advice.StartPosition,
						EndPosition:   advice.EndPosition,
					},
				},
			},
		}, nil
	}

	changeSummary, err := base.ExtractChangedResources(storepb.Engine_ORACLE, databaseName, schemaName, asts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
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
					ChangedResources: convertToChangedResources(dbMetadata, changeSummary.ResourceChanges),
				},
			},
		},
	}, nil
}

func reportForMySQL(ctx context.Context, sm *sheet.Manager, sqlDB *sql.DB, engine storepb.Engine, databaseName string, statement string, dbMetadata *model.DBSchema) ([]*storepb.PlanCheckRunResult_Result, error) {
	asts, advices := sm.GetASTsForChecks(storepb.Engine_MYSQL, statement)
	if len(advices) > 0 {
		advice := advices[0]
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   advice.Title,
				Content: advice.Content,
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:          advice.GetStartPosition().GetLine(),
						Column:        advice.GetStartPosition().GetColumn(),
						Code:          advice.Code,
						Detail:        advice.Detail,
						StartPosition: advice.StartPosition,
						EndPosition:   advice.EndPosition,
					},
				},
			},
		}, nil
	}

	changeSummary, err := base.ExtractChangedResources(storepb.Engine_MYSQL, databaseName, "" /* currentSchema */, asts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract changed resources")
	}

	var totalAffectedRows int64
	// Count DMLs.
	for _, dml := range changeSummary.SampleDMLS {
		switch engine {
		case storepb.Engine_OCEANBASE:
			count, err := getAffectedRowsCount(ctx, sqlDB, fmt.Sprintf("EXPLAIN FORMAT=JSON %s", dml), getAffectedRowsCountForOceanBase)
			if err != nil {
				return nil, err
			}
			totalAffectedRows += count
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
			count, err := getAffectedRowsCount(ctx, sqlDB, fmt.Sprintf("EXPLAIN %s", dml), getAffectedRowsCountForMysql)
			if err != nil {
				return nil, err
			}
			totalAffectedRows += count
		default:
			return nil, errors.Errorf("engine %v is not supported", engine)
		}
	}
	if changeSummary.DMLCount > common.MaximumLintExplainSize {
		totalAffectedRows = int64((float64(totalAffectedRows) / common.MaximumLintExplainSize) * float64(changeSummary.DMLCount))
	}
	// Count affected rows by DDLs.
	for _, change := range changeSummary.ResourceChanges {
		if !change.AffectTable {
			continue
		}
		if dbMetadata == nil {
			continue
		}
		dbMeta := dbMetadata.GetDatabaseMetadata()
		if dbMeta == nil {
			continue
		}
		schemaMeta := dbMeta.GetSchema(change.Resource.Schema)
		if schemaMeta == nil {
			continue
		}
		tableMeta := schemaMeta.GetTable(change.Resource.Table)
		if tableMeta == nil {
			continue
		}
		totalAffectedRows += tableMeta.GetRowCount()
	}

	nodes, ok := asts.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}
	sqlTypeSet := map[string]struct{}{}
	for _, node := range nodes {
		sqlType := mysqlparser.GetStatementType(node)
		sqlTypeSet[sqlType] = struct{}{}
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
					ChangedResources: convertToChangedResources(dbMetadata, changeSummary.ResourceChanges),
				},
			},
		},
	}, nil
}

func convertToChangedResources(dbMetadata *model.DBSchema, resourceChanges []*base.ResourceChange) *storepb.ChangedResources {
	meta := &storepb.ChangedResources{}
	// resourceChange is ordered by (db, schema, table)
	for _, resourceChange := range resourceChanges {
		resource := resourceChange.Resource
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
	var resourceChanges []*base.ResourceChange

	for _, node := range nodes {
		var newChanges []*base.ResourceChange
		sqlType, v := getStatementTypeAndResourcesFromAstNode(database, "public", node)
		if sqlType == "COMMENT" {
			v2, err := postgresExtractResourcesFromCommentStatement(database, "public", node.Text())
			if err != nil {
				slog.Error("failed to extract resources from comment statement", slog.String("statement", node.Text()), log.BBError(err))
			} else {
				newChanges = v2
			}
		} else {
			newChanges = v
		}
		sqlTypeSet[sqlType] = struct{}{}
		resourceChanges = append(resourceChanges, newChanges...)

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
					ChangedResources: convertToChangedResources(dbMetadata, resourceChanges),
				},
			},
		},
	}, nil
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

func getStatementTypeAndResourcesFromAstNode(database, schema string, node ast.Node) (string, []*base.ResourceChange) {
	switch node := node.(type) {
	// DDL

	// CREATE
	case *ast.CreateIndexStmt:
		return "CREATE_INDEX", nil
	case *ast.CreateTableStmt:
		switch node.Name.Type {
		case ast.TableTypeView:
			return "CREATE_VIEW", nil
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
			return "CREATE_TABLE", []*base.ResourceChange{{Resource: resource}}
		}
	case *ast.CreateSequenceStmt:
		return "CREATE_SEQUENCE", nil
	case *ast.CreateDatabaseStmt:
		return "CREATE_DATABASE", nil
	case *ast.CreateSchemaStmt:
		return "CREATE_SCHEMA", nil
	case *ast.CreateFunctionStmt:
		return "CREATE_FUNCTION", nil
	case *ast.CreateTriggerStmt:
		return "CREATE_TRIGGER", nil
	case *ast.CreateTypeStmt:
		return "CREATE_TYPE", nil
	case *ast.CreateExtensionStmt:
		return "CREATE_EXTENSION", nil

	// DROP
	case *ast.DropColumnStmt:
		return "DROP_COLUMN", nil
	case *ast.DropConstraintStmt:
		return "DROP_CONSTRAINT", nil
	case *ast.DropDatabaseStmt:
		return "DROP_DATABASE", nil
	case *ast.DropDefaultStmt:
		return "DROP_DEFAULT", nil
	case *ast.DropExtensionStmt:
		return "DROP_EXTENSION", nil
	case *ast.DropFunctionStmt:
		return "DROP_FUNCTION", nil
	case *ast.DropIndexStmt:
		return "DROP_INDEX", nil
	case *ast.DropNotNullStmt:
		return "DROP_NOT_NULL", nil
	case *ast.DropSchemaStmt:
		return "DROP_SCHEMA", nil
	case *ast.DropSequenceStmt:
		return "DROP_SEQUENCE", nil
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
		return "DROP_TABLE", result

	case *ast.DropTriggerStmt:
		return "DROP_TRIGGER", nil
	case *ast.DropTypeStmt:
		return "DROP_TYPE", nil

	// ALTER
	case *ast.AlterColumnTypeStmt:
		return "ALTER_COLUMN_TYPE", nil
	case *ast.AlterSequenceStmt:
		return "ALTER_SEQUENCE", nil
	case *ast.AlterTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "ALTER_VIEW", nil
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
			return "ALTER_TABLE", []*base.ResourceChange{{Resource: resource}}
		}
	case *ast.AlterTypeStmt:
		return "ALTER_TYPE", nil

	case *ast.AddColumnListStmt:
		return "ALTER_TABLE_ADD_COLUMN_LIST", nil
	case *ast.AddConstraintStmt:
		return "ALTER_TABLE_ADD_CONSTRAINT", nil

	// RENAME
	case *ast.RenameColumnStmt:
		return "RENAME_COLUMN", nil
	case *ast.RenameConstraintStmt:
		return "RENAME_CONSTRAINT", nil
	case *ast.RenameIndexStmt:
		return "RENAME_INDEX", nil
	case *ast.RenameSchemaStmt:
		return "RENAME_SCHEMA", nil
	case *ast.RenameTableStmt:
		switch node.Table.Type {
		case ast.TableTypeView:
			return "RENAME_VIEW", nil
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

			newResource := base.SchemaResource{
				Database: resource.Database,
				Schema:   resource.Schema,
				Table:    node.NewName,
			}
			return "RENAME_TABLE", []*base.ResourceChange{
				{Resource: resource},
				{Resource: newResource},
			}
		}

	case *ast.CommentStmt:
		return "COMMENT", nil

	// DML

	case *ast.InsertStmt:
		return "INSERT", nil
	case *ast.UpdateStmt:
		return "UPDATE", nil
	case *ast.DeleteStmt:
		return "DELETE", nil
	}

	return "UNKNOWN", nil
}
