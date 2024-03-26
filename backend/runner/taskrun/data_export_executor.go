package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/activity"
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
func NewDataExportExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &DataExportExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// DataExportExecutor is the data export task executor.
type DataExportExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	license         enterprise.LicenseService
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the data export task executor once.
func (exec *DataExportExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *storepb.TaskRunResult, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

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
	exportRequest := &v1pb.ExportRequest{
		Name:               fmt.Sprintf("instances/%s", instance.ResourceID),
		ConnectionDatabase: database.DatabaseName,
		Statement:          statement,
		Limit:              int32(payload.MaxRows),
		Format:             v1pb.ExportFormat(payload.Format),
		Password:           payload.Password,
	}

	schemaName := ""
	if instance.Engine == storepb.Engine_ORACLE {
		// For Oracle, there are two modes, schema-based and database-based management.
		// For schema-based management, also say tenant mode, we need to use the schemaName as the databaseName.
		// So the default schemaName is the database name.
		// For database-based management, we need to use the dataSource.Username as the schemaName.
		// So the default schemaName is the dataSource.Username.
		isSchemaTenantMode := (instance.Options != nil && instance.Options.GetSchemaTenantMode())
		if isSchemaTenantMode {
			schemaName = database.DatabaseName
		} else {
			dataSource, _, err := exec.dbFactory.GetReadOnlyDatabaseSource(instance, database, "" /* dataSourceID */)
			if err != nil {
				return true, nil, errors.Wrap(err, "failed to get read only database source")
			}
			schemaName = dataSource.Username
		}
	}

	spans, err := base.GetQuerySpan(
		ctx,
		instance.Engine,
		statement,
		database.DatabaseName,
		schemaName,
		apiv1.BuildGetDatabaseMetadataFunc(exec.store, instance, database.DatabaseName),
		apiv1.BuildListDatabaseNamesFunc(exec.store, instance),
		store.IgnoreDatabaseAndTableCaseSensitive(instance),
	)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get query span")
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
		Detail:           fmt.Sprintf("Exported successfully in %v", time.Duration(durationNs).String()),
		ExportArchiveUid: int32(exportArchive.UID),
	}, nil
}
