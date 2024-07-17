package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
func (exec *DataExportExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *storepb.TaskRunResult, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	ctx = context.WithValue(ctx, common.PrincipalIDContextKey, task.CreatorID)
	payload := &api.TaskDatabaseDataExportPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database data export payload")
	}

	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID, ShowDeleted: true})
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

	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, err
	}

	spans, err := base.GetQuerySpan(
		ctx,
		base.GetQuerySpanContext{
			InstanceID:                    instance.ResourceID,
			GetDatabaseMetadataFunc:       apiv1.BuildGetDatabaseMetadataFunc(exec.store),
			ListDatabaseNamesFunc:         apiv1.BuildListDatabaseNamesFunc(exec.store),
			GetLinkedDatabaseMetadataFunc: apiv1.BuildGetLinkedDatabaseMetadataFunc(exec.store, instance.Engine),
		},
		instance.Engine,
		statement,
		database.DatabaseName,
		"",
		store.IgnoreDatabaseAndTableCaseSensitive(instance),
	)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get query span")
	}

	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_EXECUTING,
			UpdateTime:      time.Now(),
		})
	exportRequest := &v1pb.ExportRequest{
		Name:      fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, database.DatabaseName),
		Statement: statement,
		Format:    v1pb.ExportFormat(payload.Format),
		Password:  payload.Password,
	}
	bytes, durationNs, exportErr := apiv1.DoExport(ctx, exec.store, exec.dbFactory, exec.license, exportRequest, instance, database, spans)
	if exportErr != nil {
		return true, nil, errors.Wrap(exportErr, "failed to export data")
	}

	encryptedBytes, err := apiv1.DoEncrypt(bytes, exportRequest)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to encrypt data")
	}

	exportArchive, err := exec.store.CreateExportArchive(ctx, &store.ExportArchiveMessage{
		Bytes: encryptedBytes,
		Payload: &storepb.ExportArchivePayload{
			FileFormat: payload.Format,
		},
	})
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to create export archive")
	}

	return true, &storepb.TaskRunResult{
		Detail:           fmt.Sprintf("Data export succeeded within %v", time.Duration(durationNs).String()),
		ExportArchiveUid: int32(exportArchive.UID),
	}, nil
}
