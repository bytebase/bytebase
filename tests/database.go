package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
)

func (ctl *controller) createDatabase(project *api.Project, instance *api.Instance, databaseName string, owner string, labelMap map[string]string) error {
	labels, err := marshalLabels(labelMap, instance.Environment.Name)
	if err != nil {
		return err
	}
	ctx := &api.CreateDatabaseContext{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Labels:       labels,
		CharacterSet: "utf8mb4",
		Collation:    "utf8mb4_general_ci",
	}
	if instance.Engine == db.Postgres {
		ctx.Owner = owner
		ctx.CharacterSet = "UTF8"
		ctx.Collation = "en_US.UTF-8"
	}
	createContext, err := json.Marshal(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to construct database creation issue CreateContext payload")
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("create database %q", databaseName),
		Type:        api.IssueDatabaseCreate,
		Description: fmt.Sprintf("This creates a database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create database creation issue")
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return errors.Errorf("issue %v pipeline %v is supposed to be pending manual approval", issue.ID, issue.Pipeline.ID)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
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
func (ctl *controller) cloneDatabaseFromBackup(project *api.Project, instance *api.Instance, databaseName string, backup *api.Backup, labelMap map[string]string) error {
	labels, err := marshalLabels(labelMap, instance.Environment.Name)
	if err != nil {
		return err
	}

	createContext, err := json.Marshal(&api.CreateDatabaseContext{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		BackupID:     backup.ID,
		Labels:       labels,
	})
	if err != nil {
		return errors.Wrap(err, "failed to construct database creation issue CreateContext payload")
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("create database %q from backup %q", databaseName, backup.Name),
		Type:        api.IssueDatabaseCreate,
		Description: fmt.Sprintf("This creates a database %q from backup %q.", databaseName, backup.Name),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create database creation issue")
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return errors.Errorf("issue %v pipeline %v is supposed to be pending manual approval", issue.ID, issue.Pipeline.ID)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
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

// getDatabases gets the databases.
func (ctl *controller) getDatabases(databaseFind api.DatabaseFind) ([]*api.Database, error) {
	params := make(map[string]string)
	if databaseFind.InstanceID != nil {
		params["instance"] = fmt.Sprintf("%d", *databaseFind.InstanceID)
	}
	if databaseFind.ProjectID != nil {
		params["project"] = fmt.Sprintf("%d", *databaseFind.ProjectID)
	}
	if databaseFind.Name != nil {
		params["name"] = *databaseFind.Name
	}
	body, err := ctl.get("/database", params)
	if err != nil {
		return nil, err
	}

	var databases []*api.Database
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Database)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get database response")
	}
	for _, p := range ps {
		database, ok := p.(*api.Database)
		if !ok {
			return nil, errors.Errorf("fail to convert database")
		}
		databases = append(databases, database)
	}
	return databases, nil
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

func marshalLabels(labelMap map[string]string, environmentName string) (string, error) {
	var labelList []*api.DatabaseLabel
	for k, v := range labelMap {
		labelList = append(labelList, &api.DatabaseLabel{
			Key:   k,
			Value: v,
		})
	}
	labelList = append(labelList, &api.DatabaseLabel{
		Key:   api.EnvironmentKeyName,
		Value: environmentName,
	})

	labels, err := json.Marshal(labelList)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal labels %+v", labelList)
	}
	return string(labels), nil
}

// getLabels gets all the labels.
func (ctl *controller) getLabels() ([]*api.LabelKey, error) {
	body, err := ctl.get("/label", nil)
	if err != nil {
		return nil, err
	}

	var labelKeys []*api.LabelKey
	lks, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.LabelKey)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get label response")
	}
	for _, lk := range lks {
		labelKey, ok := lk.(*api.LabelKey)
		if !ok {
			return nil, errors.Errorf("fail to convert label key")
		}
		labelKeys = append(labelKeys, labelKey)
	}
	return labelKeys, nil
}

// patchLabelKey patches the label key with given ID.
func (ctl *controller) patchLabelKey(labelKeyPatch api.LabelKeyPatch) (*api.LabelKey, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &labelKeyPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal label key patch")
	}

	body, err := ctl.patch(fmt.Sprintf("/label/%d", labelKeyPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	labelKey := new(api.LabelKey)
	if err = jsonapi.UnmarshalPayload(body, labelKey); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patch label key response")
	}
	return labelKey, nil
}

// addLabelValues adds values to an existing label key.
func (ctl *controller) addLabelValues(key string, values []string) error {
	labelKeys, err := ctl.getLabels()
	if err != nil {
		return errors.Wrap(err, "failed to get labels")
	}
	var labelKey *api.LabelKey
	for _, lk := range labelKeys {
		if lk.Key == key {
			labelKey = lk
			break
		}
	}
	if labelKey == nil {
		return errors.Errorf("failed to find label with key %q", key)
	}
	var valueList []string
	valueList = append(valueList, labelKey.ValueList...)
	valueList = append(valueList, values...)
	_, err = ctl.patchLabelKey(api.LabelKeyPatch{
		ID:        labelKey.ID,
		ValueList: valueList,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to patch label key for key %q ID %d values %+v", key, labelKey.ID, valueList)
	}
	return nil
}

func (ctl *controller) createDataSource(dataSourceCreate api.DataSourceCreate) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &dataSourceCreate); err != nil {
		return errors.Wrap(err, "failed to marshal dataSourceCreate")
	}

	body, err := ctl.post(fmt.Sprintf("/database/%d/data-source", dataSourceCreate.DatabaseID), buf)
	if err != nil {
		return err
	}

	dataSource := new(api.DataSource)
	if err = jsonapi.UnmarshalPayload(body, dataSource); err != nil {
		return errors.Wrap(err, "fail to unmarshal dataSource response")
	}
	return nil
}

func (ctl *controller) patchDataSource(databaseID int, dataSourcePatch api.DataSourcePatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &dataSourcePatch); err != nil {
		return errors.Wrap(err, "failed to marshal dataSourcePatch")
	}

	body, err := ctl.patch(fmt.Sprintf("/database/%d/data-source/%d", databaseID, dataSourcePatch.ID), buf)
	if err != nil {
		return err
	}

	dataSource := new(api.DataSource)
	if err = jsonapi.UnmarshalPayload(body, dataSource); err != nil {
		return errors.Wrap(err, "fail to unmarshal dataSource response")
	}
	return nil
}

func (ctl *controller) deleteDataSource(databaseID, dataSourceID int) error {
	_, err := ctl.delete(fmt.Sprintf("/database/%d/data-source/%d", databaseID, dataSourceID), nil)
	if err != nil {
		return err
	}
	return nil
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

// createBackup creates a backup.
func (ctl *controller) createBackup(backupCreate api.BackupCreate) (*api.Backup, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &backupCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal backupCreate")
	}

	body, err := ctl.post(fmt.Sprintf("/database/%d/backup", backupCreate.DatabaseID), buf)
	if err != nil {
		return nil, err
	}

	backup := new(api.Backup)
	if err = jsonapi.UnmarshalPayload(body, backup); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal backup response")
	}
	return backup, nil
}

// listBackups lists backups for a database.
func (ctl *controller) listBackups(databaseID int) ([]*api.Backup, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/backup", databaseID), nil)
	if err != nil {
		return nil, err
	}

	var backups []*api.Backup
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Backup)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get backup response")
	}
	for _, p := range ps {
		backup, ok := p.(*api.Backup)
		if !ok {
			return nil, errors.Errorf("fail to convert backup")
		}
		backups = append(backups, backup)
	}
	return backups, nil
}

// waitBackup waits for a backup to be done.
func (ctl *controller) waitBackup(databaseID, backupID int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Debug("Waiting for backup.", zap.Int("id", backupID))
	for range ticker.C {
		backups, err := ctl.listBackups(databaseID)
		if err != nil {
			return err
		}
		var backup *api.Backup
		for _, b := range backups {
			if b.ID == backupID {
				backup = b
				break
			}
		}
		if backup == nil {
			return errors.Errorf("backup %v for database %v not found", backupID, databaseID)
		}
		switch backup.Status {
		case api.BackupStatusDone:
			return nil
		case api.BackupStatusFailed:
			return errors.Errorf("backup %v for database %v failed", backupID, databaseID)
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return errors.Errorf("failed to wait for backup as this condition should never be reached")
}

// waitBackupArchived waits for a backup to be archived.
func (ctl *controller) waitBackupArchived(databaseID, backupID int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Debug("Waiting for backup.", zap.Int("id", backupID))
	for range ticker.C {
		backups, err := ctl.listBackups(databaseID)
		if err != nil {
			return err
		}
		var backup *api.Backup
		for _, b := range backups {
			if b.ID == backupID {
				backup = b
				break
			}
		}
		if backup == nil {
			return errors.Errorf("backup %d for database %d not found", backupID, databaseID)
		}
		if backup.RowStatus == api.Archived {
			return nil
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return errors.Errorf("failed to wait for backup as this condition should never be reached")
}
