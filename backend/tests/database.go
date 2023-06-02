package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) createDatabase(ctx context.Context, projectUID int, instance *v1pb.Instance, databaseName string, owner string, labelMap map[string]string) error {
	environmentResourceID := strings.TrimPrefix(instance.Environment, "environments/")
	instanceUID, err := strconv.Atoi(instance.Uid)
	if err != nil {
		return err
	}

	labels, err := marshalLabels(labelMap, environmentResourceID)
	if err != nil {
		return err
	}
	createCtx := &api.CreateDatabaseContext{
		InstanceID:   instanceUID,
		DatabaseName: databaseName,
		Labels:       labels,
		CharacterSet: "utf8mb4",
		Collation:    "utf8mb4_general_ci",
	}
	if instance.Engine == v1pb.Engine_POSTGRES {
		createCtx.Owner = owner
		createCtx.CharacterSet = "UTF8"
		createCtx.Collation = "en_US.UTF-8"
	}
	createContext, err := json.Marshal(createCtx)
	if err != nil {
		return errors.Wrap(err, "failed to construct database creation issue CreateContext payload")
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("create database %q", databaseName),
		Type:          api.IssueDatabaseCreate,
		Description:   fmt.Sprintf("This creates a database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create database creation issue")
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return errors.Errorf("issue %v pipeline %v is supposed to be pending manual approval %s", issue.ID, issue.Pipeline.ID, status)
	}
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to wait for issue %v pipeline %v", issue.ID, issue.Pipeline.ID)
	}
	if status != api.TaskDone {
		return errors.Errorf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}
	issue, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to patch issue status %v to done", issue.ID)
	}
	// Add a second sleep to avoid schema version conflict.
	time.Sleep(time.Second)
	return nil
}

// cloneDatabaseFromBackup clones the database from an existing backup.
func (ctl *controller) cloneDatabaseFromBackup(ctx context.Context, projectUID int, instance *v1pb.Instance, databaseName string, backup *v1pb.Backup, labelMap map[string]string) error {
	environmentID := strings.TrimPrefix(instance.Environment, "environments/")
	instanceUID, err := strconv.Atoi(instance.Uid)
	if err != nil {
		return err
	}
	labels, err := marshalLabels(labelMap, environmentID)
	if err != nil {
		return err
	}

	backupUID, err := strconv.Atoi(backup.Uid)
	if err != nil {
		return err
	}
	createContext, err := json.Marshal(&api.CreateDatabaseContext{
		InstanceID:   instanceUID,
		DatabaseName: databaseName,
		BackupID:     backupUID,
		Labels:       labels,
	})
	if err != nil {
		return errors.Wrap(err, "failed to construct database creation issue CreateContext payload")
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("create database %q from backup %q", databaseName, backup.Name),
		Type:          api.IssueDatabaseCreate,
		Description:   fmt.Sprintf("This creates a database %q from backup %q.", databaseName, backup.Name),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create database creation issue")
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return errors.Errorf("issue %v pipeline %v is supposed to be pending manual approval %s", issue.ID, issue.Pipeline.ID, status)
	}
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to wait for issue %v pipeline %v", issue.ID, issue.Pipeline.ID)
	}
	if status != api.TaskDone {
		return errors.Errorf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}
	issue, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to patch issue status %v to done", issue.ID)
	}
	return nil
}

// DatabaseEditResult is a subset struct of api.DatabaseEditResult for testing,
// because of jsonapi doesn't support to unmarshal struct pointer slice.
type DatabaseEditResult struct {
	Statement string `jsonapi:"attr,statement"`
}

// postDatabaseEdit posts the database edit.
func (ctl *controller) postDatabaseEdit(databaseEdit api.DatabaseEdit) (*DatabaseEditResult, error) {
	buf, err := json.Marshal(&databaseEdit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal databaseEdit")
	}

	res, err := ctl.post(fmt.Sprintf("/database/%v/edit", databaseEdit.DatabaseID), strings.NewReader(string(buf)))
	if err != nil {
		return nil, err
	}

	databaseEditResult := new(DatabaseEditResult)
	if err = jsonapi.UnmarshalPayload(res, databaseEditResult); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post database edit response")
	}
	return databaseEditResult, nil
}

func (ctl *controller) getLatestSchemaSDL(databaseID int) (string, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/schema", databaseID), map[string]string{"sdl": "true"})
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (ctl *controller) getLatestSchemaDump(databaseID int) (string, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/schema", databaseID), nil)
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (ctl *controller) getLatestSchemaMetadata(databaseID int) (string, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/schema", databaseID), map[string]string{"metadata": "true"})
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func marshalLabels(labelMap map[string]string, environmentID string) (string, error) {
	var labelList []*api.DatabaseLabel
	for k, v := range labelMap {
		labelList = append(labelList, &api.DatabaseLabel{
			Key:   k,
			Value: v,
		})
	}
	labelList = append(labelList, &api.DatabaseLabel{
		Key:   api.EnvironmentLabelKey,
		Value: environmentID,
	})

	labels, err := json.Marshal(labelList)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal labels %+v", labelList)
	}
	return string(labels), nil
}

// disableAutomaticBackup disables the automatic backup of a database.
func (ctl *controller) disableAutomaticBackup(databaseID int) error {
	backupSetting := api.BackupSettingUpsert{
		DatabaseID: databaseID,
		Enabled:    false,
	}
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &backupSetting); err != nil {
		return errors.Wrap(err, "failed to marshal backupSetting")
	}

	if _, err := ctl.patch(fmt.Sprintf("/database/%d/backup-setting", databaseID), buf); err != nil {
		return err
	}
	return nil
}

// waitBackup waits for a backup to be done.
func (ctl *controller) waitBackup(ctx context.Context, databaseName, backupName string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Debug("Waiting for backup.", zap.String("id", backupName))
	for range ticker.C {
		resp, err := ctl.databaseServiceClient.ListBackup(ctx, &v1pb.ListBackupRequest{Parent: databaseName})
		if err != nil {
			return err
		}
		backups := resp.Backups
		var backup *v1pb.Backup
		for _, b := range backups {
			if b.Name == backupName {
				backup = b
				break
			}
		}
		if backup == nil {
			return errors.Errorf("backup %v not found", backupName)
		}
		switch backup.State {
		case v1pb.Backup_DONE:
			return nil
		case v1pb.Backup_FAILED:
			return errors.Errorf("backup %v failed", backupName)
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return errors.Errorf("failed to wait for backup as this condition should never be reached")
}

// waitBackupArchived waits for a backup to be archived.
func (ctl *controller) waitBackupArchived(ctx context.Context, databaseName, backupName string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Debug("Waiting for backup.", zap.String("id", backupName))
	for range ticker.C {
		resp, err := ctl.databaseServiceClient.ListBackup(ctx, &v1pb.ListBackupRequest{Parent: databaseName})
		if err != nil {
			return err
		}
		backups := resp.Backups
		var backup *v1pb.Backup
		for _, b := range backups {
			if b.Name == backupName {
				backup = b
				break
			}
		}
		if backup == nil {
			return nil
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return errors.Errorf("failed to wait for backup as this condition should never be reached")
}
