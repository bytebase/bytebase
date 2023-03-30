package taskcheck

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/store"
)

// SQL review policy consists of a list of SQL review rules.
// There is such a logical mapping in Bytebase backend:
//   1. One SQL review policy maps a TaskCheckRun.
//   2. Each SQL review rule type maps an advisor.Type.
//   3. Each [db.Type][AdvisorType] maps an advisor.

// NewStatementAdvisorCompositeExecutor creates a task check statement advisor composite executor.
func NewStatementAdvisorCompositeExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementAdvisorCompositeExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// StatementAdvisorCompositeExecutor is the task check statement advisor composite executor with has sub-advisor.
type StatementAdvisorCompositeExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
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
	environment, err := e.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", database.EnvironmentID)
	}
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}
	if payload.SheetID > 0 {
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

	policy, err := e.store.GetSQLReviewPolicy(ctx, environment.UID)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusSuccess,
					Namespace: api.AdvisorNamespace,
					Code:      common.Ok.Int(),
					Title:     "Empty SQL review policy or disabled",
					Content:   "",
				},
			}, nil
		}
		return nil, common.Wrapf(err, common.Internal, "failed to get SQL review policy")
	}

	catalog, err := e.store.NewCatalog(ctx, *task.DatabaseID, instance.Engine, task.GetSyntaxMode())
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
	if err != nil {
		return nil, err
	}

	driver, err := e.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	connection, err := driver.GetDBConnection(ctx, database.DatabaseName)
	if err != nil {
		return nil, err
	}

	adviceList, err := advisor.SQLReviewCheck(payload.Statement, policy.RuleList, advisor.SQLReviewCheckContext{
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
