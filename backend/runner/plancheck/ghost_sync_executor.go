package plancheck

import (
	"context"
	"math/rand"

	"github.com/github/gh-ost/go/logic"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/ghost"
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
func (e *GhostSyncExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error) {
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
					Code:    common.Internal.Int32(),
					Report:  nil,
				},
			}
			err = nil
		}
	}()

	if config.DatabaseGroupUid != nil {
		return nil, errors.Errorf("database group is not supported")
	}

	instanceUID := int(config.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &config.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", config.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", config.DatabaseName)
	}

	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %d", instance.UID)
	}

	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID}, api.SystemBotID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetUID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetUID)
	}
	statement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetUID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	tableName, err := ghost.GetTableNameFromStatement(renderedStatement)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", statement)
	}

	migrationContext, err := ghost.NewMigrationContext(rand.Intn(10000000), database, adminDataSource, e.secret, tableName, renderedStatement, true, config.GhostFlags, 20000000)
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
				Code:    common.Internal.Int32(),
				Report:  nil,
			},
		}, nil
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
			Title:   "OK",
			Content: "gh-ost dry run succeeded",
			Code:    common.Ok.Int32(),
			Report:  nil,
		},
	}, nil
}
