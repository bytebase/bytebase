package taskcheck

import (
	"context"
	"encoding/json"
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
)

// SQL review policy consists of a list of SQL review rules.
// There is such a logical mapping in Bytebase backend:
//   1. One SQL review policy maps a TaskCheckRun.
//   2. Each SQL review rule type maps an advisor.Type.
//   3. Each [db.Type][AdvisorType] maps an advisor.

// NewStatementAdvisorCompositeExecutor creates a task check statement advisor composite executor.
func NewStatementAdvisorCompositeExecutor(
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	licenseService enterpriseAPI.LicenseService,
) Executor {
	return &StatementAdvisorCompositeExecutor{
		store:          store,
		dbFactory:      dbFactory,
		licenseService: licenseService,
	}
}

// StatementAdvisorCompositeExecutor is the task check statement advisor composite executor with has sub-advisor.
type StatementAdvisorCompositeExecutor struct {
	store          *store.Store
	dbFactory      *dbfactory.DBFactory
	licenseService enterpriseAPI.LicenseService
}

// Run will run the task check statement advisor composite executor once, and run its sub-advisor one-by-one.
func (e *StatementAdvisorCompositeExecutor) Run(ctx context.Context, taskCheckRun *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error) {
	if taskCheckRun.Type != api.TaskCheckDatabaseStatementAdvise {
		return nil, common.Errorf(common.Invalid, "invalid check statement advisor composite type: %v", taskCheckRun.Type)
	}

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database %v not found", *task.DatabaseID)
	}
	environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", database.EffectiveEnvironmentID)
	}
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", task.InstanceID)
	}
	if err := e.licenseService.IsFeatureEnabledForInstance(api.FeatureSQLReview, instance); err != nil {
		// nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusWarn,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.Unsupported.Int(),
				Title:     fmt.Sprintf("SQL review disabled for instance %s", instance.ResourceID),
				Content:   err.Error(),
			},
		}, nil
	}

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}

	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
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
	statement, err := e.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", payload.SheetID)
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

	catalog, err := e.store.NewCatalog(ctx, *task.DatabaseID, instance.Engine, task.GetSyntaxMode())
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

	result = []api.TaskCheckResult{}
	for _, advice := range adviceList {
		status := api.TaskCheckStatusSuccess
		switch advice.Status {
		case advisor.Success:
			continue
		case advisor.Warn:
			status = api.TaskCheckStatusWarn
		case advisor.Error:
			status = api.TaskCheckStatusError
		}

		result = append(result, api.TaskCheckResult{
			Status:    status,
			Namespace: api.AdvisorNamespace,
			Code:      advice.Code.Int(),
			Title:     advice.Title,
			Content:   advice.Content,
			Line:      advice.Line,
			Details:   advice.Details,
		})
	}

	if len(result) == 0 {
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "",
			Line:      0,
			Details:   "",
		})
	}

	return result, nil
}
