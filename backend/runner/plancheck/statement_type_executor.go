package plancheck

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	tidbp "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	runnerutils "github.com/bytebase/bytebase/backend/runner/utils"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var _ Executor = (*StatementTypeExecutor)(nil)

// NewStatementTypeExecutor creates a statement type executor.
func NewStatementTypeExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementTypeExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// StatementTypeExecutor is the statement type executor.
type StatementTypeExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run runs the statement type executor.
func (e *StatementTypeExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	if config.ChangeDatabaseType == storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("change database type is unspecified")
	}

	if config.DatabaseGroupUid != nil {
		return e.runForDatabaseGroupTarget(ctx, config, *config.DatabaseGroupUid)
	}
	return e.runForDatabaseTarget(ctx, config)
}

func (e *StatementTypeExecutor) runForDatabaseTarget(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeType := config.ChangeDatabaseType

	instanceUID := int(config.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}

	if !isStatementTypeCheckSupported(instance.Engine) {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   fmt.Sprintf("Statement advise is not supported for %s", instance.Engine),
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

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}

	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID}, api.SystemBotID)
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
				Title:   "Large SQL review policy is disabled",
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

	var results []*storepb.PlanCheckRunResult_Result
	switch instance.Engine {
	case storepb.Engine_POSTGRES, storepb.Engine_RISINGWAVE:
		checkResults, err := postgresqlStatementTypeCheck(renderedStatement, changeType)
		if err != nil {
			return nil, err
		}
		results = append(results, checkResults...)
	case storepb.Engine_TIDB:
		checkResults, err := tidbStatementTypeCheck(renderedStatement, dbSchema.GetMetadata().CharacterSet, dbSchema.GetMetadata().Collation, changeType)
		if err != nil {
			return nil, err
		}
		results = append(results, checkResults...)
		if changeType == storepb.PlanCheckRunConfig_SDL {
			sdlAdvice, err := e.tidbSDLTypeCheck(ctx, renderedStatement, instance, database)
			if err != nil {
				return nil, err
			}
			results = append(results, sdlAdvice...)
		}
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		checkResults, err := mysqlStatementTypeCheck(renderedStatement, changeType)
		if err != nil {
			return nil, err
		}
		results = append(results, checkResults...)
		if changeType == storepb.PlanCheckRunConfig_SDL {
			sdlAdvice, err := e.mysqlSDLTypeCheck(ctx, renderedStatement, instance, database)
			if err != nil {
				return nil, err
			}
			results = append(results, sdlAdvice...)
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type database type: %s", instance.Engine)
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

func (e *StatementTypeExecutor) runForDatabaseGroupTarget(ctx context.Context, config *storepb.PlanCheckRunConfig, databaseGroupUID int64) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeType := config.ChangeDatabaseType

	databaseGroup, err := e.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		UID: &databaseGroupUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %d", databaseGroupUID)
	}
	if databaseGroup == nil {
		return nil, errors.Errorf("database group not found %d", databaseGroupUID)
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

	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID}, api.SystemBotID)
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
		if database.DatabaseName != config.DatabaseName {
			continue
		}

		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance %q not found", database.InstanceID)
		}
		if !isStatementTypeCheckSupported(instance.Engine) {
			continue
		}
		if instance.UID != int(config.InstanceUid) {
			continue
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
				var results []*storepb.PlanCheckRunResult_Result
				switch instance.Engine {
				case storepb.Engine_POSTGRES, storepb.Engine_RISINGWAVE:
					checkResults, err := postgresqlStatementTypeCheck(renderedStatement, changeType)
					if err != nil {
						return nil, err
					}
					results = append(results, checkResults...)
				case storepb.Engine_TIDB:
					checkResults, err := tidbStatementTypeCheck(renderedStatement, dbSchema.GetMetadata().CharacterSet, dbSchema.GetMetadata().Collation, changeType)
					if err != nil {
						return nil, err
					}
					results = append(results, checkResults...)
					if changeType == storepb.PlanCheckRunConfig_SDL {
						sdlAdvice, err := e.tidbSDLTypeCheck(ctx, renderedStatement, instance, database)
						if err != nil {
							return nil, err
						}
						results = append(results, sdlAdvice...)
					}
				case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
					checkResults, err := mysqlStatementTypeCheck(renderedStatement, changeType)
					if err != nil {
						return nil, err
					}
					results = append(results, checkResults...)
					if changeType == storepb.PlanCheckRunConfig_SDL {
						sdlAdvice, err := e.mysqlSDLTypeCheck(ctx, renderedStatement, instance, database)
						if err != nil {
							return nil, err
						}
						results = append(results, sdlAdvice...)
					}
				default:
					return nil, common.Errorf(common.Invalid, "invalid check statement type database type: %s", instance.Engine)
				}

				return results, nil
			}()
			if err != nil {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_ERROR,
					Title:   "Failed to run statement type check",
					Content: err.Error(),
					Code:    common.Internal.Int32(),
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
				Code:    common.Ok.Int32(),
				Report:  nil,
			},
		}, nil
	}

	return results, nil
}

func (e *StatementTypeExecutor) tidbSDLTypeCheck(ctx context.Context, newSchema string, instance *store.InstanceMessage, database *store.DatabaseMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	ddl, err := runnerutils.ComputeDatabaseSchemaDiff(ctx, instance, database, e.dbFactory, newSchema)
	if err != nil {
		return nil, err
	}

	list, err := base.SplitMultiSQL(storepb.Engine_MYSQL, ddl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split SQL")
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, stmt := range list {
		if tidbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			continue
		}
		nodeList, _, err := tidbp.New().Parse(stmt.Text, "", "")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse schema %q", stmt.Text)
		}
		if len(nodeList) != 1 {
			return nil, errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *tidbast.DropTableStmt:
			for _, table := range node.Tables {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropTable.Int32(),
					Title:   "Plan to drop table",
					Content: fmt.Sprintf("Plan to drop table `%s`", table.Name.O),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(stmt.LastLine),
							Detail: "",
							Code:   0,
						},
					},
				})
			}
		case *tidbast.DropIndexStmt:
			results = append(results, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_WARNING,
				Code:    common.TaskTypeDropIndex.Int32(),
				Title:   "Plan to drop index",
				Content: fmt.Sprintf("Plan to drop index `%s` on table `%s`", node.IndexName, node.Table.Name.O),
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   int32(stmt.LastLine),
						Detail: "",
						Code:   0,
					},
				},
			})
		case *tidbast.AlterTableStmt:
			for _, spec := range node.Specs {
				switch spec.Tp {
				case tidbast.AlterTableDropColumn:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropColumn.Int32(),
						Title:   "Plan to drop column",
						Content: fmt.Sprintf("Plan to drop column `%s` on table `%s`", spec.OldColumnName.Name.O, node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int32(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				case tidbast.AlterTableDropPrimaryKey:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropPrimaryKey.Int32(),
						Title:   "Plan to drop primary key",
						Content: fmt.Sprintf("Plan to drop primary key on table `%s`", node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int32(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				case tidbast.AlterTableDropForeignKey:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropForeignKey.Int32(),
						Title:   "Plan to drop foreign key",
						Content: fmt.Sprintf("Plan to drop foreign key `%s` on table `%s`", spec.Name, node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int32(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				case tidbast.AlterTableDropCheck:
					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  storepb.PlanCheckRunResult_Result_WARNING,
						Code:    common.TaskTypeDropCheck.Int32(),
						Title:   "Plan to drop check constraint",
						Content: fmt.Sprintf("Plan to drop check constraint `%s` on table `%s`", spec.Constraint.Name, node.Table.Name.O),
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int32(stmt.LastLine),
								Detail: "",
								Code:   0,
							},
						},
					})
				}
			}
		}
	}
	return results, nil
}

func tidbCreateAndDropDatabaseCheck(nodeList []tidbast.StmtNode) []*storepb.PlanCheckRunResult_Result {
	var results []*storepb.PlanCheckRunResult_Result
	for _, node := range nodeList {
		switch node.(type) {
		case *tidbast.DropDatabaseStmt:
			results = append(results, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeDropDatabase.Int32(),
				Title:   "Cannot drop database",
				Content: fmt.Sprintf(`The statement "%s" drops database`, node.Text()),
			})
		case *tidbast.CreateDatabaseStmt:
			results = append(results, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeCreateDatabase.Int32(),
				Title:   "Cannot create database",
				Content: fmt.Sprintf(`The statement "%s" creates database`, node.Text()),
			})
		}
	}

	return results
}

func mysqlStatementTypeCheck(statement string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) ([]*storepb.PlanCheckRunResult_Result, error) {
	stmts, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    0,
				Title:   "Syntax error",
				Content: err.Error(),
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}

	var results []*storepb.PlanCheckRunResult_Result

	// Disallow CREATE/DROP DATABASE statements.
	results = append(results, mysqlCreateAndDropDatabaseCheck(stmts)...)

	switch changeType {
	case storepb.PlanCheckRunConfig_DML:
		for _, node := range stmts {
			checker := &mysqlTypeChecker{}
			antlr.ParseTreeWalkerDefault.Walk(checker, node.Tree)
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if checker.isDDL {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Data change can only run DML",
					Content: fmt.Sprintf("\"%s\" is not DML", checker.text),
					Code:    common.TaskTypeNotDML.Int32(),
					Report:  nil,
				})
			}
		}
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL:
		for _, node := range stmts {
			checker := &mysqlTypeChecker{}
			antlr.ParseTreeWalkerDefault.Walk(checker, node.Tree)
			if checker.isDML || checker.isExplain {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Alter schema can only run DDL",
					Content: fmt.Sprintf("\"%s\" is not DDL", checker.text),
					Code:    common.TaskTypeNotDDL.Int32(),
					Report:  nil,
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", changeType)
	}

	return results, nil
}

func mysqlCreateAndDropDatabaseCheck(stmtList []*mysqlparser.ParseResult) []*storepb.PlanCheckRunResult_Result {
	checker := &mysqlCreateAndDropDatabaseChecker{}
	for _, stmt := range stmtList {
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.results
}

type mysqlCreateAndDropDatabaseChecker struct {
	*mysql.BaseMySQLParserListener

	text    string
	results []*storepb.PlanCheckRunResult_Result
}

func (checker *mysqlCreateAndDropDatabaseChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (checker *mysqlCreateAndDropDatabaseChecker) EnterCreateDatabase(ctx *mysql.CreateDatabaseContext) {
	checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_ERROR,
		Code:    common.TaskTypeCreateDatabase.Int32(),
		Title:   "Cannot create database",
		Content: fmt.Sprintf(`The statement "%s" creates database`, checker.text),
	})
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (checker *mysqlCreateAndDropDatabaseChecker) EnterDropDatabase(ctx *mysql.DropDatabaseContext) {
	checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_ERROR,
		Code:    common.TaskTypeDropDatabase.Int32(),
		Title:   "Cannot drop database",
		Content: fmt.Sprintf(`The statement "%s" drops database`, checker.text),
	})
}

type mysqlTypeChecker struct {
	*mysql.BaseMySQLParserListener

	isDDL     bool
	isDML     bool
	isExplain bool
	text      string
}

func (checker *mysqlTypeChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// DDL from MySQLParser.g4
// alterStatement
// createStatement
// dropStatement
// renameTableStatement
// truncateTableStatement
// importStatement
// EnterAlterStatement is called when production alterStatement is entered.
func (checker *mysqlTypeChecker) EnterAlterStatement(ctx *mysql.AlterStatementContext) {
	checker.isDDL = true
}

// EnterCreateStatement is called when production createStatement is entered.
func (checker *mysqlTypeChecker) EnterCreateStatement(ctx *mysql.CreateStatementContext) {
	checker.isDDL = true
}

// EnterDropStatement is called when production dropStatement is entered.
func (checker *mysqlTypeChecker) EnterDropStatement(ctx *mysql.DropStatementContext) {
	checker.isDDL = true
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (checker *mysqlTypeChecker) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	checker.isDDL = true
}

// EnterTruncateTableStatement is called when production truncateTableStatement is entered.
func (checker *mysqlTypeChecker) EnterTruncateTableStatement(ctx *mysql.TruncateTableStatementContext) {
	checker.isDDL = true
}

// EnterImportStatement is called when production importStatement is entered.
func (checker *mysqlTypeChecker) EnterImportStatement(ctx *mysql.ImportStatementContext) {
	checker.isDDL = true
}

// EnterExplainStatement is called when production explainStatement is entered.
func (checker *mysqlTypeChecker) EnterExplainStatement(ctx *mysql.ExplainStatementContext) {
	checker.isExplain = true
}

// DML
// EnterCallStatement is called when production callStatement is entered.
func (checker *mysqlTypeChecker) EnterCallStatement(ctx *mysql.CallStatementContext) {
	checker.isDML = true
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *mysqlTypeChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	checker.isDML = true
}

// EnterDoStatement is called when production doStatement is entered.
func (checker *mysqlTypeChecker) EnterDoStatement(ctx *mysql.DoStatementContext) {
	checker.isDML = true
}

// EnterHandlerStatement is called when production handlerStatement is entered.
func (checker *mysqlTypeChecker) EnterHandlerStatement(ctx *mysql.HandlerStatementContext) {
	checker.isDML = true
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *mysqlTypeChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	checker.isDML = true
}

// EnterLoadDataFileTail is called when production loadDataFileTail is entered.
func (checker *mysqlTypeChecker) EnterLoadDataFileTail(ctx *mysql.LoadDataFileTailContext) {
	checker.isDML = true
}

// EnterReplaceStatement is called when production replaceStatement is entered.
func (checker *mysqlTypeChecker) EnterReplaceStatement(ctx *mysql.ReplaceStatementContext) {
	checker.isDML = true
}

// EnterSelectStatement is called when production selectStatement is entered.
func (checker *mysqlTypeChecker) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	checker.isDML = true
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *mysqlTypeChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	checker.isDML = true
}

// EnterTransactionOrLockingStatement is called when production transactionOrLockingStatement is entered.
func (checker *mysqlTypeChecker) EnterTransactionOrLockingStatement(ctx *mysql.TransactionOrLockingStatementContext) {
	checker.isDML = true
}

// EnterReplicationStatement is called when production replicationStatement is entered.
func (checker *mysqlTypeChecker) EnterReplicationStatement(ctx *mysql.ReplicationStatementContext) {
	checker.isDML = true
}

// EnterPreparedStatement is called when production preparedStatement is entered.
func (checker *mysqlTypeChecker) EnterPreparedStatement(ctx *mysql.PreparedStatementContext) {
	checker.isDML = true
}

func (e *StatementTypeExecutor) mysqlSDLTypeCheck(ctx context.Context, newSchema string, instance *store.InstanceMessage, database *store.DatabaseMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	ddl, err := runnerutils.ComputeDatabaseSchemaDiff(ctx, instance, database, e.dbFactory, newSchema)
	if err != nil {
		return nil, err
	}

	list, err := base.SplitMultiSQL(storepb.Engine_MYSQL, ddl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split SQL")
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, stmt := range list {
		nodeList, err := mysqlparser.ParseMySQL(stmt.Text)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse schema %q", stmt.Text)
		}
		if len(nodeList) != 1 {
			return nil, errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		checker := &tidbSDLTypeChecker{}
		antlr.ParseTreeWalkerDefault.Walk(checker, nodeList[0].Tree)
	}

	return results, nil
}

type tidbSDLTypeChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine int
	results  []*storepb.PlanCheckRunResult_Result
}

// for SDL.
// EnterDropTable is called when production dropTable is entered.
func (checker *tidbSDLTypeChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
			Status:  storepb.PlanCheckRunResult_Result_WARNING,
			Code:    common.TaskTypeDropTable.Int32(),
			Title:   "Plan to drop table",
			Content: fmt.Sprintf("Plan to drop table `%s`", tableName),
			Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
				SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
					Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
					Detail: "",
					Code:   0,
				},
			},
		})
	}
}

// EnterDropIndex is called when production dropIndex is entered.
func (checker *tidbSDLTypeChecker) EnterDropIndex(ctx *mysql.DropIndexContext) {
	if ctx.IndexRef() == nil || ctx.TableRef() == nil {
		return
	}

	_, _, indexName := mysqlparser.NormalizeIndexRef(ctx.IndexRef())
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_WARNING,
		Code:    common.TaskTypeDropIndex.Int32(),
		Title:   "Plan to drop index",
		Content: fmt.Sprintf("Plan to drop index `%s` on table `%s`", indexName, tableName),
		Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
			SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
				Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
				Detail: "",
				Code:   0,
			},
		},
	})
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *tidbSDLTypeChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.TableRef() == nil {
		// todo: maybe need to do error handle.
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())

	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if alterListItem == nil {
			continue
		}

		switch {
		case alterListItem.DROP_SYMBOL() != nil:
			switch {
			// drop column.
			case alterListItem.ColumnInternalRef() != nil:
				columnName := mysqlparser.NormalizeMySQLColumnInternalRef(alterListItem.ColumnInternalRef())
				checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropColumn.Int32(),
					Title:   "Plan to drop column",
					Content: fmt.Sprintf("Plan to drop column `%s` on table `%s`", columnName, tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + alterListItem.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			// drop primary key.
			case alterListItem.PRIMARY_SYMBOL() != nil && alterListItem.KEY_SYMBOL() != nil:
				checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropPrimaryKey.Int32(),
					Title:   "Plan to drop primary key",
					Content: fmt.Sprintf("Plan to drop primary key on table `%s`", tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			// drop foreign key.
			case alterListItem.FOREIGN_SYMBOL() != nil && alterListItem.KEY_SYMBOL() != nil && alterListItem.ColumnInternalRef() != nil:
				name := mysqlparser.NormalizeMySQLColumnInternalRef(alterListItem.ColumnInternalRef())
				checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropForeignKey.Int32(),
					Title:   "Plan to drop foreign key",
					Content: fmt.Sprintf("Plan to drop foreign key `%s` on table `%s`", name, tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			// drop check.
			case alterListItem.CHECK_SYMBOL() != nil && alterListItem.Identifier() != nil:
				constraintName := mysqlparser.NormalizeMySQLIdentifier(alterListItem.Identifier())
				checker.results = append(checker.results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropCheck.Int32(),
					Title:   "Plan to drop check constraint",
					Content: fmt.Sprintf("Plan to drop check constraint `%s` on table `%s`", constraintName, tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			}
		default:
		}
	}
}

func tidbStatementTypeCheck(statement string, charset string, collation string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) ([]*storepb.PlanCheckRunResult_Result, error) {
	// Due to the limitation of TiDB parser, we should split the multi-statement into single statements, and extract
	// the TiDB unsupported statements, otherwise, the parser will panic or return the error.
	unsupportStmt, supportStmt, err := tidbparser.ExtractTiDBUnsupportedStmts(statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}
	// TODO(zp): We regard the DELIMITER statement as a DDL statement here.
	// But we should ban the DELIMITER statement because go-sql-driver doesn't support it.
	hasUnsupportDDL := len(unsupportStmt) > 0

	p := tidbp.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	stmts, _, err := p.Parse(supportStmt, charset, collation)
	if err != nil {
		// nolint: nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "Syntax error",
				Content: err.Error(),
				Code:    0,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}

	var results []*storepb.PlanCheckRunResult_Result

	// Disallow CREATE/DROP DATABASE statements.
	results = append(results, tidbCreateAndDropDatabaseCheck(stmts)...)

	switch changeType {
	case storepb.PlanCheckRunConfig_DML:
		for _, node := range stmts {
			_, isDDL := node.(tidbast.DDLNode)
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL || hasUnsupportDDL {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Data change can only run DML",
					Content: fmt.Sprintf("\"%s\" is not DML", node.Text()),
					Code:    common.TaskTypeNotDML.Int32(),
					Report:  nil,
				})
			}
		}
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL:
		for _, node := range stmts {
			_, isDML := node.(tidbast.DMLNode)
			_, isExplain := node.(*tidbast.ExplainStmt)
			if isDML || isExplain {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Alter schema can only run DDL",
					Content: fmt.Sprintf("\"%s\" is not DDL", node.Text()),
					Code:    common.TaskTypeNotDDL.Int32(),
					Report:  nil,
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", changeType)
	}

	return results, nil
}

func postgresqlCreateAndDropDatabaseCheck(nodeList []ast.Node) []*storepb.PlanCheckRunResult_Result {
	var result []*storepb.PlanCheckRunResult_Result
	for _, node := range nodeList {
		switch node.(type) {
		case *ast.DropDatabaseStmt:
			result = append(result, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeDropDatabase.Int32(),
				Title:   "Cannot drop database",
				Content: fmt.Sprintf(`The statement "%s" drops database`, node.Text()),
				Report:  nil,
			})
		case *ast.CreateDatabaseStmt:
			result = append(result, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.TaskTypeCreateDatabase.Int32(),
				Title:   "Cannot create database",
				Content: fmt.Sprintf(`The statement "%s" creates database`, node.Text()),
				Report:  nil,
			})
		}
	}
	return result
}

func postgresqlStatementTypeCheck(statement string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) ([]*storepb.PlanCheckRunResult_Result, error) {
	stmts, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    0,
				Title:   "Syntax error",
				Content: err.Error(),
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.StatementSyntaxError.Int32(),
					},
				},
			},
		}, nil
	}

	var results []*storepb.PlanCheckRunResult_Result

	// Disallow CREATE/DROP DATABASE statements.
	results = append(results, postgresqlCreateAndDropDatabaseCheck(stmts)...)

	switch changeType {
	case storepb.PlanCheckRunConfig_DML:
		for _, node := range stmts {
			_, isDDL := node.(ast.DDLNode)
			// We only want to disallow DDL statements in CHANGE DATA.
			// We need to run some common statements, e.g. COMMIT.
			if isDDL {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Data change can only run DML",
					Content: fmt.Sprintf("\"%s\" is not DML", node.Text()),
					Code:    common.TaskTypeNotDML.Int32(),
					Report:  nil,
				})
			}
		}
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL:
		for _, node := range stmts {
			_, isDML := node.(ast.DMLNode)
			_, isSelect := node.(*ast.SelectStmt)
			_, isExplain := node.(*ast.ExplainStmt)
			if isDML || isSelect || isExplain {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Title:   "Alter schema can only run DDL",
					Content: fmt.Sprintf("\"%s\" is not DDL", node.Text()),
					Code:    common.TaskTypeNotDDL.Int32(),
					Report:  nil,
				})
			}
		}
	default:
		return nil, common.Errorf(common.Invalid, "invalid check statement type task type: %s", changeType)
	}

	return results, nil
}
