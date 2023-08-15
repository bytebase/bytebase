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

// StatementAdviseExecutor is the plan check statement statement advise executor.
type StatementAdviseExecutor struct {
	store          *store.Store
	dbFactory      *dbfactory.DBFactory
	licenseService enterpriseAPI.LicenseService
}

// Run will run the plan check statement advise executor once, and run its sub-advisor one-by-one.
func (e *StatementAdviseExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	if planCheckRun.Type != store.PlanCheckDatabaseStatementAdvise {
		return nil, common.Errorf(common.Invalid, "unexpected plan check type in statement advise executor: %v", planCheckRun.Type)
	}

	changeType := planCheckRun.Config.ChangeDatabaseType
	if changeType == storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("change database type is unspecified")
	}

	databaseID := int(planCheckRun.Config.DatabaseId)
	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database ID %v", databaseID)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %v", databaseID)
	}

	environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
	}
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %v", database.InstanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found %v", database.InstanceID)
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

	sheetID := int(planCheckRun.Config.SheetId)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetID)
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
	statement, err := e.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetID)
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

func getSyntaxMode(t storepb.PlanCheckRunConfig_ChangeDatabaseType) advisor.SyntaxMode {
	if t == storepb.PlanCheckRunConfig_SDL {
		return advisor.SyntaxModeSDL
	}
	return advisor.SyntaxModeNormal
}
