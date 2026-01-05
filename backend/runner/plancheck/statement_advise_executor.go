package plancheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewStatementAdviseExecutor creates a plan check statement advise executor.
func NewStatementAdviseExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
) Executor {
	return &StatementAdviseExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// StatementAdviseExecutor is the plan check statement advise executor.
type StatementAdviseExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

// RunForTarget runs the statement advise check for a single target.
func (e *StatementAdviseExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	fullSheet, err := e.store.GetSheetFull(ctx, target.SheetSha256)
	if err != nil {
		return nil, err
	}
	if fullSheet == nil {
		return nil, errors.Errorf("sheet full %s not found", target.SheetSha256)
	}
	if fullSheet.Size > common.MaxSheetCheckSize {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_WARNING,
				Code:    common.SizeExceeded.Int32(),
				Title:   "Large SQL review policy is disabled",
				Content: "",
			},
		}, nil
	}
	enablePriorBackup := target.EnablePriorBackup
	enableGhost := target.EnableGhost

	instanceID, databaseName, err := common.GetInstanceDatabaseID(target.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse target %s", target.Target)
	}

	instance, err := e.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %s not found", instanceID)
	}
	if !common.EngineSupportStatementAdvise(instance.Metadata.GetEngine()) {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   fmt.Sprintf("Statement advise is not supported for %s", instance.Metadata.GetEngine()),
				Content: "",
			},
		}, nil
	}

	database, err := e.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", databaseName)
	}

	results, err := e.runReview(ctx, instance, database, fullSheet.Statement, enablePriorBackup, enableGhost)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_SUCCESS,
				Title:   "OK",
				Content: "",
				Code:    common.Ok.Int32(),
				Report:  nil,
			},
		}, nil
	}
	return results, nil
}

func (e *StatementAdviseExecutor) runReview(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	statement string,
	enablePriorBackup bool,
	enableGhost bool,
) ([]*storepb.PlanCheckRunResult_Result, error) {
	dbMetadata, err := e.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, err
	}
	if dbMetadata == nil {
		return nil, errors.Errorf("database schema %s not found", database.String())
	}
	if dbMetadata.GetProto() == nil {
		return nil, errors.Errorf("database schema metadata %s not found", database.String())
	}

	reviewConfig, err := e.store.GetReviewConfigForDatabase(ctx, database)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			// Continue to check the builtin rules.
			reviewConfig = &storepb.ReviewConfigPayload{}
		} else {
			return nil, common.Wrapf(err, common.Internal, "failed to get SQL review config")
		}
	}

	// Create original metadata as read-only
	originMetadata := model.NewDatabaseMetadata(dbMetadata.GetProto(), nil, nil, instance.Metadata.GetEngine(), store.IsObjectCaseSensitive(instance))

	// Clone metadata for final to avoid modifying the original
	clonedMetadata, ok := proto.Clone(dbMetadata.GetProto()).(*storepb.DatabaseSchemaMetadata)
	if !ok {
		return nil, common.Wrapf(errors.New("failed to clone database schema metadata"), common.Internal, "failed to create a catalog")
	}
	finalMetadata := model.NewDatabaseMetadata(clonedMetadata, nil, nil, instance.Metadata.GetEngine(), store.IsObjectCaseSensitive(instance))

	project, err := e.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to get project")
	}
	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{TenantMode: project.Setting.GetPostgresDatabaseTenantMode()})
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	adviceList, err := advisor.SQLReviewCheck(ctx, e.sheetManager, statement, reviewConfig.SqlReviewRules, advisor.Context{
		DBSchema:              dbMetadata.GetProto(),
		DBType:                instance.Metadata.GetEngine(),
		OriginalMetadata:      originMetadata,
		FinalMetadata:         finalMetadata,
		Driver:                connection,
		EnablePriorBackup:     enablePriorBackup,
		EnableGhost:           enableGhost,
		TenantMode:            project.Setting.GetPostgresDatabaseTenantMode(),
		ListDatabaseNamesFunc: e.buildListDatabaseNamesFunc(),
		InstanceID:            instance.ResourceID,
		IsObjectCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, err
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, advice := range adviceList {
		status := storepb.Advice_SUCCESS
		switch advice.Status {
		case storepb.Advice_SUCCESS:
			continue
		case storepb.Advice_WARNING:
			status = storepb.Advice_WARNING
		case storepb.Advice_ERROR:
			status = storepb.Advice_ERROR
		default:
			// Other status types
		}

		results = append(results, &storepb.PlanCheckRunResult_Result{
			Status:  status,
			Title:   advice.Title,
			Content: advice.Content,
			Code:    advice.Code,
			Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
				SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
					StartPosition: advice.StartPosition,
					EndPosition:   advice.EndPosition,
				},
			},
		})
	}

	if len(results) == 0 {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_SUCCESS,
				Title:   "OK",
				Content: "",
				Code:    common.Ok.Int32(),
				Report:  nil,
			},
		}, nil
	}

	return results, nil
}

func (e *StatementAdviseExecutor) buildListDatabaseNamesFunc() parserbase.ListDatabaseNamesFunc {
	return func(ctx context.Context, instanceID string) ([]string, error) {
		databases, err := e.store.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instanceID,
		})
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(databases))
		for _, database := range databases {
			names = append(names, database.DatabaseName)
		}
		return names, nil
	}
}
