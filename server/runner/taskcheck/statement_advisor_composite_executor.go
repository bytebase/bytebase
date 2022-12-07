package taskcheck

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
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
func (e *StatementAdvisorCompositeExecutor) Run(ctx context.Context, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	if taskCheckRun.Type != api.TaskCheckDatabaseStatementAdvise {
		return nil, common.Errorf(common.Invalid, "invalid check statement advisor composite type: %v", taskCheckRun.Type)
	}

	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return nil, common.Wrapf(err, common.Invalid, "invalid check statement advise payload")
	}

	policy, err := e.store.GetNormalSQLReviewPolicy(ctx, &api.PolicyFind{ID: &payload.PolicyID})
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

	task, err := e.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to get task by id")
	}

	catalog, err := e.store.NewCatalog(ctx, *task.DatabaseID, payload.DbType)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(payload.DbType))
	if err != nil {
		return nil, err
	}

	driver, err := e.dbFactory.GetReadOnlyDatabaseDriver(ctx, task.Instance, task.Database.Name)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	connection, err := driver.GetDBConnection(ctx, task.Database.Name)
	if err != nil {
		return nil, err
	}

	adviceList, err := advisor.SQLReviewCheck(payload.Statement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:   payload.Charset,
		Collation: payload.Collation,
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
		})
	}

	if len(result) == 0 {
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "",
		})
	}

	return result, nil
}
