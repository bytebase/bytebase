package taskcheck

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewStatementAdvisorSimpleExecutor creates a task check statement simple advisor executor.
func NewStatementAdvisorSimpleExecutor(store *store.Store, licenseService enterpriseAPI.LicenseService) Executor {
	return &StatementAdvisorSimpleExecutor{
		store:          store,
		licenseService: licenseService,
	}
}

// StatementAdvisorSimpleExecutor is the task check statement advisor simple executor.
type StatementAdvisorSimpleExecutor struct {
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// Run will run the task check statement advisor executor once.
func (e *StatementAdvisorSimpleExecutor) Run(ctx context.Context, taskCheckRun *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error) {
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

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database %v not found", task.DatabaseID)
	}

	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}

	sheet, err := e.store.GetSheetV2(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
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

	var advisorType advisor.Type
	switch taskCheckRun.Type {
	case api.TaskCheckDatabaseStatementFakeAdvise:
		advisorType = advisor.Fake
	case api.TaskCheckDatabaseStatementSyntax:
		switch instance.Engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			advisorType = advisor.MySQLSyntax
		case db.Postgres:
			advisorType = advisor.PostgreSQLSyntax
		case db.Oracle:
			advisorType = advisor.OracleSyntax
		case db.Snowflake:
			advisorType = advisor.SnowflakeSyntax
		default:
			return nil, common.Errorf(common.Invalid, "invalid database type: %s for syntax statement advisor", instance.Engine)
		}
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
	if err != nil {
		return nil, err
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	adviceList, err := advisor.Check(
		dbType,
		advisorType,
		advisor.Context{
			Charset:    dbSchema.Metadata.CharacterSet,
			Collation:  dbSchema.Metadata.Collation,
			SyntaxMode: task.GetSyntaxMode(),
		},
		renderedStatement,
	)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to check statement")
	}

	result = []api.TaskCheckResult{}
	for _, advice := range adviceList {
		status := api.TaskCheckStatusSuccess
		switch advice.Status {
		case advisor.Success:
			status = api.TaskCheckStatusSuccess
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

	return result, nil
}
