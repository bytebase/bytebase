package plancheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewStatementAdviseExecutor creates a plan check statement advise executor.
func NewStatementAdviseExecutor(
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	licenseService enterprise.LicenseService,
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
	licenseService enterprise.LicenseService
}

// Run will run the plan check statement advise executor once, and run its sub-advisors one-by-one.
func (e *StatementAdviseExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	if config.ChangeDatabaseType == storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("change database type is unspecified")
	}

	if config.DatabaseGroupUid != nil {
		return e.runForDatabaseGroupTarget(ctx, config, *config.DatabaseGroupUid)
	}
	return e.runForDatabaseTarget(ctx, config)
}

func (e *StatementAdviseExecutor) runForDatabaseTarget(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	changeType := config.ChangeDatabaseType

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

	environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
	}

	if err := e.licenseService.IsFeatureEnabled(api.FeatureSQLReview); err != nil {
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
						Code:   advisor.Unsupported.Int32(),
					},
				},
			},
		}, nil
	}

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
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

	policy, err := e.store.GetSQLReviewPolicy(ctx, environment.UID)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			policy = &storepb.SQLReviewPolicy{
				Name: "Default",
			}
		} else {
			return nil, common.Wrapf(err, common.Internal, "failed to get SQL review policy")
		}
	}

	catalog, err := e.store.NewCatalog(ctx, database.UID, instance.Engine, store.IgnoreDatabaseAndTableCaseSensitive(instance), nil /* Override Metadata */, getSyntaxMode(changeType))
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{UseDatabaseOwner: true})
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)
	adviceList, err := advisor.SQLReviewCheck(renderedStatement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:   dbSchema.GetMetadata().CharacterSet,
		Collation: dbSchema.GetMetadata().Collation,
		DbType:    instance.Engine,
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
					Line:   int32(advice.Line),
					Column: int32(advice.Column),
					Code:   advice.Code.Int32(),
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
				Code:    common.Ok.Int32(),
				Report:  nil,
			},
		}, nil
	}

	return results, nil
}

func (e *StatementAdviseExecutor) runForDatabaseGroupTarget(ctx context.Context, config *storepb.PlanCheckRunConfig, databaseGroupUID int64) ([]*storepb.PlanCheckRunResult_Result, error) {
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

		if err := e.licenseService.IsFeatureEnabled(api.FeatureSQLReview); err != nil {
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
							Code:   advisor.Unsupported.Int32(),
						},
					},
				},
			}, nil
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
			policy, err := e.store.GetSQLReviewPolicy(ctx, environment.UID)
			if err != nil {
				if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
					policy = &storepb.SQLReviewPolicy{
						Name: "Default",
					}
				} else {
					return nil, common.Wrapf(err, common.Internal, "failed to get SQL review policy")
				}
			}

			catalog, err := e.store.NewCatalog(ctx, database.UID, instance.Engine, store.IgnoreDatabaseAndTableCaseSensitive(instance), nil /* Override Metadata */, getSyntaxMode(changeType))
			if err != nil {
				return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
			}

			stmtResults, err := func() ([]*storepb.PlanCheckRunResult_Result, error) {
				driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{UseDatabaseOwner: true})
				if err != nil {
					return nil, err
				}
				defer driver.Close(ctx)
				connection := driver.GetDB()

				materials := utils.GetSecretMapFromDatabaseMessage(database)
				// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
				renderedStatement := utils.RenderStatement(statement, materials)
				adviceList, err := advisor.SQLReviewCheck(renderedStatement, policy.RuleList, advisor.SQLReviewCheckContext{
					Charset:   dbSchema.GetMetadata().CharacterSet,
					Collation: dbSchema.GetMetadata().Collation,
					DbType:    instance.Engine,
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
								Line:   int32(advice.Line),
								Column: int32(advice.Column),
								Code:   advice.Code.Int32(),
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

func getSyntaxMode(t storepb.PlanCheckRunConfig_ChangeDatabaseType) advisor.SyntaxMode {
	if t == storepb.PlanCheckRunConfig_SDL {
		return advisor.SyntaxModeSDL
	}
	return advisor.SyntaxModeNormal
}
