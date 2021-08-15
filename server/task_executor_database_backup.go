package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/dump/mysqldump"
	"github.com/bytebase/bytebase/db"
	"go.uber.org/zap"
)

// NewDatabaseBackupTaskExecutor creates a new database backup task executor.
func NewDatabaseBackupTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &DatabaseBackupTaskExecutor{
		l: logger,
	}
}

// DatabaseBackupTaskExecutor is the task executor for database backup.
type DatabaseBackupTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run database backup once.
func (exec *DatabaseBackupTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, detail string, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseBackupTaskExecutor PANIC RECOVER", zap.Error(panicErr))
			terminated = true
			err = fmt.Errorf("encounter internal error when backing database")
		}
	}()

	payload := &api.TaskDatabaseBackupPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, "", fmt.Errorf("invalid database backup payload: %w", err)
	}

	if err := server.ComposeTaskRelationship(ctx, task); err != nil {
		return true, "", err
	}

	backup, err := server.BackupService.FindBackup(ctx, &api.BackupFind{ID: &payload.BackupId})
	if err != nil {
		return true, "", fmt.Errorf("failed to find backup: %w", err)
	}
	exec.l.Debug("Start database backup...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", task.Database.Name),
		zap.String("backup", backup.Name),
	)

	backupErr := backupDatabase(task.Instance, task.Database, backup, server.dataDir)
	// Update the status of the backup.
	newBackupStatus := string(api.BackupStatusDone)
	if backupErr != nil {
		newBackupStatus = string(api.BackupStatusFailed)
	}
	if _, err = server.BackupService.PatchBackup(ctx, &api.BackupPatch{
		ID:        backup.ID,
		Status:    newBackupStatus,
		UpdaterId: api.SYSTEM_BOT_ID,
	}); err != nil {
		return true, "", fmt.Errorf("failed to patch backup: %w", err)
	}

	if backupErr != nil {
		return true, "", backupErr
	}

	return true, fmt.Sprintf("Backup database '%s'", task.Database.Name), nil
}

// backupDatabase will take a backup of a database.
func backupDatabase(instance *api.Instance, database *api.Database, backup *api.Backup, dataDir string) error {
	conn, err := connect.NewMysql(instance.Username, instance.Password, instance.Host, instance.Port, database.Name, nil /* tlsConfig */)
	if err != nil {
		return fmt.Errorf("connect.NewMysql(%q, %q, %q, %q) got error: %v", instance.Username, instance.Password, instance.Host, instance.Port, err)
	}
	defer conn.Close()
	dp := mysqldump.New(conn)

	f, err := os.Create(filepath.Join(dataDir, backup.Path))
	if err != nil {
		return fmt.Errorf("failed to open backup path: %s", backup.Path)
	}
	defer f.Close()

	if err := dp.Dump(database.Name, f, false /* schemaOnly */, false /* dumpAll */); err != nil {
		return err
	}

	return nil
}

// getAndCreateBackupDirectory returns the path of a database backup.
func getAndCreateBackupDirectory(dataDir string, database *api.Database) (string, error) {
	dir := filepath.Join("backup", "db", fmt.Sprintf("%d", database.ID))
	absDir := filepath.Join(dataDir, dir)
	if err := os.MkdirAll(absDir, 0700); err != nil {
		return "", nil
	}

	return dir, nil
}

// getAndCreateBackupPath returns the path of a database backup.
func getAndCreateBackupPath(dataDir string, database *api.Database, name string) (string, error) {
	dir, err := getAndCreateBackupDirectory(dataDir, database)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("%s.sql", name)), nil
}

func getMigrationVersion(database *api.Database, logger *zap.Logger) (string, error) {
	instance := database.Instance
	driver, err := db.Open(
		instance.Engine,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
			Database: database.Name,
		},
		db.ConnectionContext{
			EnvironmentName: instance.Environment.Name,
			InstanceName:    instance.Name,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect instance: %v with user: %v. %w", instance.Name, instance.Username, err)
	}
	defer driver.Close(context.Background())

	limit := 1
	find := &db.MigrationHistoryFind{
		Database: &database.Name,
		Limit:    &limit,
	}
	list, err := driver.FindMigrationHistoryList(context.Background(), find)
	if err != nil {
		return "", fmt.Errorf("failed to fetch migration history list: %v", err)
	}
	if len(list) == 0 {
		return "", nil
	}
	return list[0].Version, nil
}
