package taskrun

import (
	"context"

	"github.com/pkg/errors"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/bytebase/bytebase/backend/store"
)

// NewDataExportExecutor creates a data export task executor.
func NewDataExportExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &DataExportExecutor{
		store:        store,
		dbFactory:    dbFactory,
		license:      license,
		stateCfg:     stateCfg,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// DataExportExecutor is the data export task executor.
type DataExportExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	license      enterprise.LicenseService
	stateCfg     *state.State
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

// RunOnce will run the data export task executor once.
func (exec *DataExportExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, _ int) (terminated bool, result *storepb.TaskRunResult, err error) {
	issue, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get issue")
	}

	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName, ShowDeleted: true})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found")
	}
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID, ShowDeleted: true})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get instance")
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance not found")
	}

	statement, err := exec.store.GetSheetStatementByID(ctx, int(task.Payload.GetSheetId()))
	if err != nil {
		return true, nil, err
	}

	exportRequest := &v1pb.ExportRequest{
		Name:      common.FormatDatabase(instance.ResourceID, database.DatabaseName),
		Statement: statement,
		Format:    v1pb.ExportFormat(task.Payload.GetFormat()),
		Password:  task.Payload.GetPassword(),
	}
	bytes, _, exportErr := apiv1.DoExport(ctx, exec.store, exec.dbFactory, exec.license, exportRequest, issue.Creator /* user */, instance, database, nil, exec.schemaSyncer, true /* auto data source */)
	if exportErr != nil {
		return true, nil, errors.Wrap(exportErr, "failed to export data")
	}

	exportArchive, err := exec.store.CreateExportArchive(ctx, &store.ExportArchiveMessage{
		Bytes: bytes,
		Payload: &storepb.ExportArchivePayload{
			FileFormat: task.Payload.GetFormat(),
		},
	})
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to create export archive")
	}

	return true, &storepb.TaskRunResult{
		Detail:           "Data export succeeded",
		ExportArchiveUid: int32(exportArchive.UID),
	}, nil
}
