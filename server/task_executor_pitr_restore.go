package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// NewPITRRestoreTaskExecutor creates a PITR restore task executor.
func NewPITRRestoreTaskExecutor(instance *mysqlutil.Instance) TaskExecutor {
	return &PITRRestoreTaskExecutor{
		mysqlutil: instance,
	}
}

// PITRRestoreTaskExecutor is the PITR restore task executor.
type PITRRestoreTaskExecutor struct {
	mysqlutil *mysqlutil.Instance
}

// RunOnce will run the PITR restore task executor once.
func (exec *PITRRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR restore task", zap.String("task", task.Name))

	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return true, nil, fmt.Errorf("invalid PITR restore payload[%s], error[%w]", task.Payload, err)
	}

	return exec.pitrRestore(ctx, task, server)
}

// TODO(dragonly): Should establish a BASELINE migration in the swap database task.
// And what's the right schema version in tenant mode?
func (exec *PITRRestoreTaskExecutor) pitrRestore(ctx context.Context, task *api.Task, server *Server) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", "" /* pgInstanceDir */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if err := exec.doPITRRestore(ctx, task, server.store, driver, server.profile.DataDir); err != nil {
		return true, nil, err
	}

	log.Info("created PITR database", zap.String("target database", task.Database.Name))

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Created PITR database for target database %q", task.Database.Name),
	}, nil
}

func (exec *PITRRestoreTaskExecutor) doPITRRestore(ctx context.Context, task *api.Task, store *store.Store, driver db.Driver, dataDir string) error {
	instance := task.Instance
	database := task.Database

	issue, err := getIssueByPipelineID(ctx, store, task.PipelineID)
	if err != nil {
		return err
	}

	connCfg, err := getConnectionConfig(ctx, instance, database.Name)
	if err != nil {
		return err
	}

	binlogDir, err := getAndCreateBinlogDirectory(dataDir, task.Instance)
	if err != nil {
		return err
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		log.Error("failed to cast driver to mysql.Driver", zap.Stack("stack"))
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	mysqlRestore := restoremysql.New(mysqlDriver, exec.mysqlutil, connCfg, binlogDir)

	binlogInfo := api.BinlogInfo{}
	// TODO(dragonly): Search and put the file io of the logical backup file here.
	// Currently, let's just use the empty backup dump as a placeholder.
	var buf bytes.Buffer
	buf.WriteString("-- This is a fake backup dump")

	log.Debug("Start creating and restoring PITR database",
		zap.String("instance", instance.Name),
		zap.String("database", database.Name),
	)

	// RestorePITR will create the pitr database.
	// Since it's ephemeral and will be renamed to the original database soon, we will reuse the original
	// database's migration history, and append a new BASELINE migration.
	if err := mysqlRestore.RestorePITR(ctx, bufio.NewScanner(&buf), binlogInfo, database.Name, issue.CreatedTs); err != nil {
		log.Error("failed to perform a PITR restore in the PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("database", database.Name),
			zap.Stack("stack"),
			zap.Error(err))
		return fmt.Errorf("failed to perform a PITR restore in the PITR database, error[%w]", err)
	}

	return nil
}

// getAndCreateBinlogDirectory returns the path of a instance binlog directory.
func getAndCreateBinlogDirectory(dataDir string, instance *api.Instance) (string, error) {
	dir := filepath.Join("backup", "instance", fmt.Sprintf("%d", instance.ID))
	absDir := filepath.Join(dataDir, dir)
	if err := os.MkdirAll(absDir, os.ModePerm); err != nil {
		return "", nil
	}

	return dir, nil
}

func getIssueByPipelineID(ctx context.Context, store *store.Store, pid int) (*api.Issue, error) {
	issue, err := store.GetIssueByPipelineID(ctx, pid)
	if err != nil {
		log.Error("failed to get issue by PipelineID", zap.Int("PipelineID", pid), zap.Error(err))
		return nil, fmt.Errorf("failed to get issue by PipelineID[%d], error[%w]", pid, err)
	}
	if issue == nil {
		log.Error("issue not found with PipelineID", zap.Int("PipelineID", pid))
		return nil, fmt.Errorf("issue not found with PipelineID[%d]", pid)
	}
	return issue, nil
}
