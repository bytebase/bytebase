package plancheck

import (
	"context"

	"github.com/github/gh-ost/go/logic"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewGhostSyncExecutor creates a gh-ost sync check executor.
func NewGhostSyncExecutor(store *store.Store, secret string) Executor {
	return &GhostSyncExecutor{
		store:  store,
		secret: secret,
	}
}

// GhostSyncExecutor is the gh-ost sync check executor.
type GhostSyncExecutor struct {
	store  *store.Store
	secret string
}

// Run runs the gh-ost sync check executor.
func (e *GhostSyncExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) (results []*storepb.PlanCheckRunResult_Result, err error) {
	// gh-ost dry run could panic.
	// It may be bytebase who panicked, but that's rare. So
	// capture the error and send it into the result list.
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}

			results = []*storepb.PlanCheckRunResult_Result{
				{
					Status:  storepb.PlanCheckRunResult_Result_ERROR,
					Title:   "gh-ost dry run failed",
					Content: panicErr.Error(),
					Code:    common.Internal.Int64(),
					Report:  nil,
				},
			}
			err = nil
		}
	}()

	databaseID := int(planCheckRun.Config.DatabaseId)
	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database ID %v", databaseID)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %v", databaseID)
	}
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %v", database.InstanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found %v", database.InstanceID)
	}

	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %d", instance.UID)
	}

	instanceUsers, err := e.store.ListInstanceUsers(ctx, &store.FindInstanceUserMessage{InstanceUID: instance.UID})
	if err != nil {
		return nil, common.Errorf(common.Internal, "failed to find instance user by instanceID %d", instance.UID)
	}

	sheetID := int(planCheckRun.Config.SheetId)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetID)
	}
	statement, err := e.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	tableName, err := utils.GetTableNameFromStatement(renderedStatement)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", statement)
	}

	config, err := utils.GetGhostConfig(planCheckRun.UID, database, adminDataSource, e.secret, instanceUsers, tableName, renderedStatement, true, 20000000)
	if err != nil {
		return nil, err
	}

	migrationContext, err := utils.NewMigrationContext(config)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create migration context")
	}

	migrator := logic.NewMigrator(migrationContext, "bb")

	if err := migrator.Migrate(); err != nil {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "gh-ost dry run failed",
				Content: err.Error(),
				Code:    common.Internal.Int64(),
				Report:  nil,
			},
		}, nil
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
			Title:   "OK",
			Content: "gh-ost dry run succeeded",
			Code:    common.Ok.Int64(),
			Report:  nil,
		},
	}, nil
}
