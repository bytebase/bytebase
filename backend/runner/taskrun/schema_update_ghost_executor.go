package taskrun

import (
	"context"
	"log/slog"
	"strings"

	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/ghost"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewSchemaUpdateGhostExecutor creates a schema update (gh-ost) task executor.
func NewSchemaUpdateGhostExecutor(s *store.Store, secret string) Executor {
	return &SchemaUpdateGhostExecutor{
		s:      s,
		secret: secret,
	}
}

// SchemaUpdateGhostExecutor is the schema update (gh-ost) task executor.
type SchemaUpdateGhostExecutor struct {
	s      *store.Store
	secret string
}

func (exec *SchemaUpdateGhostExecutor) RunOnce(ctx context.Context, taskContext context.Context, task *store.TaskMessage, _ int) (terminated bool, result *storepb.TaskRunResult, err error) {
	payload := &storepb.TaskDatabaseUpdatePayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}
	statement, err := exec.s.GetSheetStatementByID(ctx, int(payload.SheetId))
	if err != nil {
		return true, nil, err
	}

	return exec.runGhostMigration(ctx, taskContext, task, statement, payload.Flags)
}

func (exec *SchemaUpdateGhostExecutor) runGhostMigration(ctx context.Context, taskContext context.Context, task *store.TaskMessage, statement string, flags map[string]string) (terminated bool, result *storepb.TaskRunResult, err error) {
	// set buffer size to 1 to unblock the sender because there is no listener if the task is canceled.
	migrationError := make(chan error, 1)

	statement = strings.TrimSpace(statement)

	tableName, err := ghost.GetTableNameFromStatement(statement)
	if err != nil {
		return true, nil, err
	}

	instance, err := exec.s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance %s not found", task.InstanceID)
	}
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return true, nil, common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
	}

	database, err := exec.s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found")
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	migrationContext, err := ghost.NewMigrationContext(ctx, task.ID, database, adminDataSource, exec.secret, tableName, "", renderedStatement, false, flags, 10000000)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to init migrationContext for gh-ost")
	}
	defer func() {
		// Use migrationContext.Uuid as the tls_config_key by convention.
		// We need to deregister it when gh-ost exits.
		// https://github.com/bytebase/gh-ost2/pull/4
		gomysql.DeregisterTLSConfig(migrationContext.Uuid)
	}()
	// TODO(p0ny): unset in NewMigrationContext.
	migrationContext.PostponeCutOverFlagFile = ""

	migrator := logic.NewMigrator(migrationContext, "bb")

	go func() {
		if err := migrator.Migrate(); err != nil {
			slog.Error("failed to run gh-ost migration", log.BBError(err))
			migrationError <- err
			return
		}
		migrationError <- nil
	}()

	select {
	case err := <-migrationError:
		if err != nil {
			return true, &storepb.TaskRunResult{Detail: err.Error()}, err
		}
		return true, nil, nil
	case <-ctx.Done():
		migrationContext.PanicAbort <- errors.New("task canceled")
		return true, nil, errors.New("task canceled")
	case <-taskContext.Done():
		migrationContext.PanicAbort <- errors.New("task canceled")
		return true, nil, errors.New("task canceled")
	}
}
