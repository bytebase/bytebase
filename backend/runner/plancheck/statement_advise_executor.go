package plancheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"

	v1api "github.com/bytebase/bytebase/backend/api/v1"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewStatementAdviseExecutor creates a plan check statement advise executor.
func NewStatementAdviseExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	licenseService enterprise.LicenseService,
) Executor {
	return &StatementAdviseExecutor{
		store:          store,
		sheetManager:   sheetManager,
		dbFactory:      dbFactory,
		licenseService: licenseService,
	}
}

// StatementAdviseExecutor is the plan check statement advise executor.
type StatementAdviseExecutor struct {
	store          *store.Store
	sheetManager   *sheet.Manager
	dbFactory      *dbfactory.DBFactory
	licenseService enterprise.LicenseService
}

// Run will run the plan check statement advise executor once, and run its sub-advisors one-by-one.
func (e *StatementAdviseExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	if config.ChangeDatabaseType == storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("change database type is unspecified")
	}
	if err := e.licenseService.IsFeatureEnabled(api.FeatureSQLReview); err != nil {
		// nolint:nilerr
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_WARNING,
				Code:    0,
				Title:   "SQL review is disabled",
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

	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
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
		return nil, err
	}
	changeType := config.ChangeDatabaseType
	preUpdateBackupDetail := config.PreUpdateBackupDetail

	instanceUID := int(config.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}
	if !common.StatementAdviseEngines[instance.Engine] {
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

	results, err := e.runReview(ctx, instance, database, changeType, statement, preUpdateBackupDetail)
	if err != nil {
		return nil, err
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

func (e *StatementAdviseExecutor) runReview(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	statement string,
	preUpdateBackupDetail *storepb.PreUpdateBackupDetail,
) ([]*storepb.PlanCheckRunResult_Result, error) {
	dbSchema, err := e.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	if dbSchema == nil {
		return nil, errors.Errorf("database schema not found: %d", database.UID)
	}
	if dbSchema.GetMetadata() == nil {
		return nil, errors.Errorf("database schema metadata not found: %d", database.UID)
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

	catalog, err := catalog.NewCatalog(ctx, e.store, database.UID, instance.Engine, store.IgnoreDatabaseAndTableCaseSensitive(instance), nil /* Override Metadata */)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create a catalog")
	}

	useDatabaseOwner := false
	if changeType != storepb.PlanCheckRunConfig_SQL_EDITOR {
		useDatabaseOwner, err = getUseDatabaseOwner(ctx, e.store, instance, database)
		if err != nil {
			return nil, common.Wrapf(err, common.Internal, "failed to get use database owner")
		}
	}
	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{UseDatabaseOwner: useDatabaseOwner})
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)
	classificationConfig := v1api.GetClassificationByProject(ctx, e.store, database.ProjectID)

	adviceList, err := advisor.SQLReviewCheck(e.sheetManager, renderedStatement, reviewConfig.SqlReviewRules, advisor.SQLReviewCheckContext{
		Charset:                  dbSchema.GetMetadata().CharacterSet,
		Collation:                dbSchema.GetMetadata().Collation,
		DBSchema:                 dbSchema.GetMetadata(),
		ChangeType:               changeType,
		DbType:                   instance.Engine,
		Catalog:                  catalog,
		Driver:                   connection,
		Context:                  ctx,
		PreUpdateBackupDetail:    preUpdateBackupDetail,
		ClassificationConfig:     classificationConfig,
		UsePostgresDatabaseOwner: useDatabaseOwner,
		ListDatabaseNamesFunc:    e.buildListDatabaseNamesFunc(),
		InstanceID:               instance.ResourceID,
	})
	if err != nil {
		return nil, err
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, advice := range adviceList {
		status := storepb.PlanCheckRunResult_Result_SUCCESS
		switch advice.Status {
		case storepb.Advice_SUCCESS:
			continue
		case storepb.Advice_WARNING:
			status = storepb.PlanCheckRunResult_Result_WARNING
		case storepb.Advice_ERROR:
			status = storepb.PlanCheckRunResult_Result_ERROR
		}

		results = append(results, &storepb.PlanCheckRunResult_Result{
			Status:  status,
			Title:   advice.Title,
			Content: advice.Content,
			Code:    0,
			Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
				SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
					Line:          advice.GetStartPosition().GetLine(),
					Column:        advice.GetStartPosition().GetColumn(),
					Code:          advice.Code,
					Detail:        advice.Detail,
					StartPosition: advice.StartPosition,
					EndPosition:   advice.EndPosition,
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

func (e *StatementAdviseExecutor) buildListDatabaseNamesFunc() base.ListDatabaseNamesFunc {
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
