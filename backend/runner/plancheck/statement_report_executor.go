package plancheck

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewStatementReportExecutor creates a statement report executor.
func NewStatementReportExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementReportExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// StatementReportExecutor is the statement report executor.
type StatementReportExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run runs the statement report executor.
func (e *StatementReportExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	if target := planCheckRun.Config.GetDatabaseTarget(); target != nil {
		return e.runForDatabaseTarget(ctx, planCheckRun, target)
	}
	if target := planCheckRun.Config.GetDatabaseGroupTarget(); target != nil {
		return e.runForDatabaseGroupTarget(ctx, planCheckRun, target)
	}
	return nil, errors.New("plan check run target is required")
}

func (e *StatementReportExecutor) runForDatabaseTarget(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, target *storepb.PlanCheckRunConfig_DatabaseTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	instanceUID := int(target.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}

	if !isStatementReportSupported(instance.Engine) {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int64(),
				Title:   fmt.Sprintf("Statement advise is not supported for %s", instance.Engine),
				Content: "",
			},
		}, nil
	}

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &target.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", target.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", target.DatabaseName)
	}

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}
	charset := dbSchema.Metadata.CharacterSet
	collation := dbSchema.Metadata.Collation

	sheetUID := int(planCheckRun.Config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetUID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetUID)
	}
	if sheet.Size > common.MaxSheetSizeForTaskCheck {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int64(),
				Title:   "Report for large SQL is not supported",
				Content: "",
			},
		}, nil
	}

	statement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetUID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	switch instance.Engine {
	case db.Postgres:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB := driver.GetDB()

		return reportForPostgres(ctx, sqlDB, database.DatabaseName, renderedStatement)
	case db.MySQL, db.OceanBase:
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
		if err != nil {
			return nil, err
		}
		defer driver.Close(ctx)
		sqlDB := driver.GetDB()

		return reportForMySQL(ctx, sqlDB, instance.Engine, database.DatabaseName, renderedStatement, charset, collation)
	case db.Oracle:
		schema := ""
		if instance.Options == nil || !instance.Options.SchemaTenantMode {
			adminSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
			schema = adminSource.Username
		} else {
			schema = database.DatabaseName
		}
		return reportForOracle(database.DatabaseName, schema, renderedStatement)
	default:
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int64(),
				Title:   "Not available",
				Content: fmt.Sprintf("Report is not supported for %s", instance.Engine),
			},
		}, nil
	}
}

func (e *StatementReportExecutor) runForDatabaseGroupTarget(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, target *storepb.PlanCheckRunConfig_DatabaseGroupTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	databaseGroup, err := e.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		UID: &target.DatabaseGroupUid,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %d", target.DatabaseGroupUid)
	}
	if databaseGroup == nil {
		return nil, errors.Errorf("database group not found %d", target.DatabaseGroupUid)
	}
	schemaGroups, err := e.store.ListSchemaGroups(ctx, &store.FindSchemaGroupMessage{DatabaseGroupUID: &databaseGroup.UID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list schema groups for database group %q", databaseGroup.UID)
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

	sheetUID := int(planCheckRun.Config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetUID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetUID)
	}
	if sheet.Size > common.MaxSheetSizeForTaskCheck {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int64(),
				Title:   "Large SQL review policy is disabled",
				Content: "",
			},
		}, nil
	}
	sheetStatement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetUID)
	}

	var results []*storepb.PlanCheckRunResult_Result

	for _, database := range matchedDatabases {
		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance %q not found", database.InstanceID)
		}

		environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
		if err != nil {
			return nil, err
		}
		if environment == nil {
			return nil, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
		}

		dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get db schema %q", database.UID)
		}

		schemaGroupsMatchedTables := map[string][]string{}
		for _, schemaGroup := range schemaGroups {
			matches, _, err := utils.GetMatchedAndUnmatchedTablesInSchemaGroup(ctx, dbSchema, schemaGroup)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get matched and unmatched tables in schema group %q", schemaGroup.ResourceID)
			}
			schemaGroupsMatchedTables[schemaGroup.ResourceID] = matches
		}

		parserEngineType, err := utils.ConvertDatabaseToParserEngineType(instance.Engine)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert database engine %q to parser engine type", instance.Engine)
		}

		statements, _, err := utils.GetStatementsAndSchemaGroupsFromSchemaGroups(sheetStatement, parserEngineType, "", schemaGroups, schemaGroupsMatchedTables)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get statements from schema groups")
		}

		for _, statement := range statements {
			materials := utils.GetSecretMapFromDatabaseMessage(database)
			// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
			renderedStatement := utils.RenderStatement(statement, materials)
			stmtResults, err := func() ([]*storepb.PlanCheckRunResult_Result, error) {
				switch instance.Engine {
				case db.Postgres:
					driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
					if err != nil {
						return nil, err
					}
					defer driver.Close(ctx)
					sqlDB := driver.GetDB()

					return reportForPostgres(ctx, sqlDB, database.DatabaseName, renderedStatement)
				case db.MySQL, db.OceanBase:
					driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
					if err != nil {
						return nil, err
					}
					defer driver.Close(ctx)
					sqlDB := driver.GetDB()

					charset := dbSchema.Metadata.CharacterSet
					collation := dbSchema.Metadata.Collation
					return reportForMySQL(ctx, sqlDB, instance.Engine, database.DatabaseName, renderedStatement, charset, collation)
				case db.Oracle:
					schema := ""
					if instance.Options == nil || !instance.Options.SchemaTenantMode {
						adminSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
						schema = adminSource.Username
					} else {
						schema = database.DatabaseName
					}
					return reportForOracle(database.DatabaseName, schema, renderedStatement)
				default:
					return nil, nil
				}
			}()
			if err != nil {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_ERROR,
					Title:   "Failed to run report executor",
					Content: err.Error(),
					Code:    common.Internal.Int64(),
					Report:  nil,
				})
			} else {
				results = append(results, stmtResults...)
			}
		}
	}

	if len(results) == 0 {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Title:   "OK",
				Content: "",
				Code:    common.Ok.Int64(),
				Report:  nil,
			},
		}, nil
	}

	return results, nil
}

func reportForOracle(databaseName string, schemaName string, statement string) ([]*storepb.PlanCheckRunResult_Result, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.Oracle, statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int64(),
					},
				},
			},
		}, nil
	}

	var changedResources []parser.SchemaResource

	for _, stmt := range singleSQLs {
		if stmt.Empty || stmt.Text == "" {
			continue
		}
		resources, err := getChangedResourcesForOracle(databaseName, schemaName, stmt.Text)
		if err != nil {
			log.Error("failed to extract changed resources", zap.String("statement", stmt.Text), zap.Error(err))
		} else {
			changedResources = append(changedResources, resources...)
		}
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int64(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   nil,
					AffectedRows:     0,
					ChangedResources: convertToChangedResources(changedResources),
				},
			},
		},
	}, nil
}

func getChangedResourcesForOracle(databaseName string, schemaName string, statement string) ([]parser.SchemaResource, error) {
	return parser.ExtractChangedResources(parser.Oracle, databaseName, schemaName, statement)
}

func reportForMySQL(ctx context.Context, sqlDB *sql.DB, dbType db.Type, databaseName, statement, charset, collation string) ([]*storepb.PlanCheckRunResult_Result, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.MySQL, statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int64(),
					},
				},
			},
		}, nil
	}

	sqlTypeSet := map[string]struct{}{}
	var totalAffectedRows int64
	var changedResources []parser.SchemaResource

	p := tidbparser.New()
	p.EnableWindowFunc(true)

	for _, stmt := range singleSQLs {
		if stmt.Empty {
			continue
		}
		if parser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			continue
		}
		root, _, err := p.Parse(stmt.Text, charset, collation)
		if err != nil {
			log.Error("failed to parse statement", zap.String("statement", stmt.Text), zap.Error(err))
			continue
		}
		if len(root) != 1 {
			log.Error("failed to parse statement, expect to get one node from parser", zap.String("statement", stmt.Text), zap.Error(err))
			continue
		}
		sqlType, resources := getStatementTypeFromTidbAstNode(strings.ToLower(databaseName), root[0])
		sqlTypeSet[sqlType] = struct{}{}
		if !isDML(sqlType) {
			if dbType != db.TiDB {
				resources, err := getStatementChangedResourcesForMySQL(databaseName, stmt.Text)
				if err != nil {
					log.Error("failed to get statement changed resources", zap.Error(err))
				} else {
					changedResources = append(changedResources, resources...)
				}
			} else {
				changedResources = append(changedResources, resources...)
			}
		}

		affectedRows, err := getAffectedRowsForMysql(ctx, dbType, sqlDB, root[0])
		if err != nil {
			log.Error("failed to get affected rows for mysql", zap.String("database", databaseName), zap.Error(err))
		} else {
			totalAffectedRows += affectedRows
		}
	}

	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:   common.Ok.Int64(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     totalAffectedRows,
					ChangedResources: convertToChangedResources(changedResources),
				},
			},
		},
	}, nil
}

func isDML(tp string) bool {
	switch tp {
	case "REPLACE", "INSERT", "UPDATE", "DELETE":
		return true
	default:
		return false
	}
}

func convertToChangedResources(resources []parser.SchemaResource) *storepb.ChangedResources {
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
		schema.Tables = append(schema.Tables, &storepb.ChangedResourceTable{Name: resource.Table})
	}
	return meta
}

func getStatementChangedResourcesForMySQL(currentDatabase, statement string) ([]parser.SchemaResource, error) {
	return parser.ExtractChangedResources(parser.MySQL, currentDatabase, "" /* currentSchema */, statement)
}

func reportForPostgres(ctx context.Context, sqlDB *sql.DB, database, statement string) ([]*storepb.PlanCheckRunResult_Result, error) {
	stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
					SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
						Code: advisor.StatementSyntaxError.Int64(),
					},
				},
			},
		}, nil
	}

	sqlTypeSet := map[string]struct{}{}
	var totalAffectedRows int64
	var changedResources []parser.SchemaResource

	for _, stmt := range stmts {
		sqlType, resources := getStatementTypeAndResourcesFromAstNode(database, "public", stmt)
		if sqlType == "COMMENT" {
			resources, err = postgresExtractResourcesFromCommentStatement(database, "public", stmt.Text())
			if err != nil {
				log.Error("failed to extract resources from comment statement", zap.String("statement", stmt.Text()), zap.Error(err))
				resources = nil
			}
		}
		sqlTypeSet[sqlType] = struct{}{}
		changedResources = append(changedResources, resources...)

		rowCount, err := getAffectedRowsForPostgres(ctx, sqlDB, stmt)
		if err != nil {
			log.Error("failed to get affected rows for postgres", zap.String("database", database), zap.Error(err))
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
			Code:   common.Ok.Int64(),
			Title:  "OK",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					StatementTypes:   sqlTypes,
					AffectedRows:     totalAffectedRows,
					ChangedResources: convertToChangedResources(changedResources),
				},
			},
		},
	}, nil
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
