package plancheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewStatementAdviseExecutor creates a plan check statement advise executor.
func NewStatementAdviseExecutor(
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	licenseService enterpriseAPI.LicenseService,
) Executor {
	return &StatementAdviseExecutor{
		store:          store,
		dbFactory:      dbFactory,
		licenseService: licenseService,
	}
}

// StatementAdviseExecutor is the plan check statement advise executor.
type StatementAdviseExecutor struct {
	store          *store.Store
	dbFactory      *dbfactory.DBFactory
	licenseService enterpriseAPI.LicenseService
}

// Run will run the plan check statement advise executor once, and run its sub-advisors one-by-one.
func (e *StatementAdviseExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	if planCheckRun.Type != store.PlanCheckDatabaseStatementAdvise {
		return nil, common.Errorf(common.Invalid, "unexpected plan check type in statement advise executor: %v", planCheckRun.Type)
	}

	changeType := planCheckRun.Config.ChangeDatabaseType
	if changeType == storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("change database type is unspecified")
	}

	if planCheckRun.Config.DatabaseGroupUid != nil {
		return e.runForDatabaseGroupTarget(ctx, planCheckRun, *planCheckRun.Config.DatabaseGroupUid)
	} else {
		return e.runForDatabaseTarget(ctx, planCheckRun)
	}
}

func (e *StatementAdviseExecutor) runForDatabaseTarget(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeType := planCheckRun.Config.ChangeDatabaseType
	config := planCheckRun.Config

	instanceUID := int(config.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}

	if !isStatementAdviseSupported(instance.Engine) {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
				Code:    common.Ok.Int64(),
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

	environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
	}

	if err := e.licenseService.IsFeatureEnabledForInstance(api.FeatureSQLReview, instance); err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_WARNING,
				Code:    0,
				Title:   fmt.Sprintf("SQL review disabled for instance %s", instance.ResourceID),
				Content: err.Error(),
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						Line:   0,
						Detail: "",
						Code:   advisor.Unsupported.Int64(),
					},
				},
			},
		}, nil
	}

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
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
	statement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetUID)
	}

	policy, err := e.store.GetSQLReviewPolicy(ctx, environment.UID)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			policy = &advisor.SQLReviewPolicy{
				Name:     "Default",
				RuleList: []*advisor.SQLReviewRule{},
			}
		} else {
			return nil, common.Wrapf(err, common.Internal, "failed to get SQL review policy")
		}
	}

	catalog, err := e.store.NewCatalog(ctx, database.UID, instance.Engine, getSyntaxMode(changeType))
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
	if err != nil {
		return nil, err
	}

	driver, err := e.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)
	adviceList, err := advisor.SQLReviewCheck(renderedStatement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:   dbSchema.Metadata.CharacterSet,
		Collation: dbSchema.Metadata.Collation,
		DbType:    dbType,
		Catalog:   catalog,
		Driver:    connection,
		Context:   ctx,
	})
	if err != nil {
		return nil, err
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, advice := range adviceList {
		status := storepb.PlanCheckRunResult_Result_SUCCESS
		switch advice.Status {
		case advisor.Success:
			continue
		case advisor.Warn:
			status = storepb.PlanCheckRunResult_Result_WARNING
		case advisor.Error:
			status = storepb.PlanCheckRunResult_Result_ERROR
		}

		results = append(results, &storepb.PlanCheckRunResult_Result{
			Status:  status,
			Title:   advice.Title,
			Content: advice.Content,
			Code:    0,
			Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
				SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
					Line:   int64(advice.Line),
					Column: int64(advice.Column),
					Code:   advice.Code.Int64(),
					Detail: advice.Details,
				},
			},
		})
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

func (e *StatementAdviseExecutor) runForDatabaseGroupTarget(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, databaseGroupUID int64) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeType := planCheckRun.Config.ChangeDatabaseType
	config := planCheckRun.Config
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
	for _, db := range matchedDatabases {
		if db.DatabaseName != config.DatabaseName {
			continue
		}

		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &db.InstanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", db.InstanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance %q not found", db.InstanceID)
		}
		if instance.UID != int(config.InstanceUid) {
			continue
		}

		environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &db.EffectiveEnvironmentID})
		if err != nil {
			return nil, err
		}
		if environment == nil {
			return nil, errors.Errorf("environment %q not found", db.EffectiveEnvironmentID)
		}

		if err := e.licenseService.IsFeatureEnabledForInstance(api.FeatureSQLReview, instance); err != nil {
			// nolint:nilerr
			return []*storepb.PlanCheckRunResult_Result{
				{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    0,
					Title:   fmt.Sprintf("SQL review disabled for instance %s", instance.ResourceID),
					Content: err.Error(),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   0,
							Detail: "",
							Code:   advisor.Unsupported.Int64(),
						},
					},
				},
			}, nil
		}

		dbSchema, err := e.store.GetDBSchema(ctx, db.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get db schema %q", db.UID)
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
			policy, err := e.store.GetSQLReviewPolicy(ctx, environment.UID)
			if err != nil {
				if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
					policy = &advisor.SQLReviewPolicy{
						Name:     "Default",
						RuleList: []*advisor.SQLReviewRule{},
					}
				} else {
					return nil, common.Wrapf(err, common.Internal, "failed to get SQL review policy")
				}
			}

			catalog, err := e.store.NewCatalog(ctx, db.UID, instance.Engine, getSyntaxMode(changeType))
			if err != nil {
				return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
			}

			dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
			if err != nil {
				return nil, err
			}

			stmtResults, err := func() ([]*storepb.PlanCheckRunResult_Result, error) {
				driver, err := e.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, db)
				if err != nil {
					return nil, err
				}
				defer driver.Close(ctx)
				connection := driver.GetDB()

				materials := utils.GetSecretMapFromDatabaseMessage(db)
				// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
				renderedStatement := utils.RenderStatement(statement, materials)
				adviceList, err := advisor.SQLReviewCheck(renderedStatement, policy.RuleList, advisor.SQLReviewCheckContext{
					Charset:   dbSchema.Metadata.CharacterSet,
					Collation: dbSchema.Metadata.Collation,
					DbType:    dbType,
					Catalog:   catalog,
					Driver:    connection,
					Context:   ctx,
				})
				if err != nil {
					return nil, err
				}

				var results []*storepb.PlanCheckRunResult_Result
				for _, advice := range adviceList {
					status := storepb.PlanCheckRunResult_Result_SUCCESS
					switch advice.Status {
					case advisor.Success:
						continue
					case advisor.Warn:
						status = storepb.PlanCheckRunResult_Result_WARNING
					case advisor.Error:
						status = storepb.PlanCheckRunResult_Result_ERROR
					}

					results = append(results, &storepb.PlanCheckRunResult_Result{
						Status:  status,
						Title:   advice.Title,
						Content: advice.Content,
						Code:    0,
						Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
							SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
								Line:   int64(advice.Line),
								Column: int64(advice.Column),
								Code:   advice.Code.Int64(),
								Detail: advice.Details,
							},
						},
					})
				}
				return results, nil
			}()
			if err != nil {
				results = append(results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_ERROR,
					Title:   "Failed to run SQL review",
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

func getSyntaxMode(t storepb.PlanCheckRunConfig_ChangeDatabaseType) advisor.SyntaxMode {
	if t == storepb.PlanCheckRunConfig_SDL {
		return advisor.SyntaxModeSDL
	}
	return advisor.SyntaxModeNormal
}
