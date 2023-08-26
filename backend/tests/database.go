package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) createDatabaseV2(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, databaseName string, owner string, labels map[string]string) error {
	characterSet, collation := "utf8mb4", "utf8mb4_general_ci"
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet = "UTF8"
		collation = "en_US.UTF-8"
	}

	plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Steps: []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
								CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
									Target:       instance.Name,
									Database:     databaseName,
									CharacterSet: characterSet,
									Collation:    collation,
									Owner:        owner,
									Labels:       labels,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Plan: plan.Name})
	if err != nil {
		return err
	}

	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        plan.Name,
			Rollout:     rollout.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return err
	}

	if err := ctl.waitRollout(ctx, rollout.Name); err != nil {
		return err
	}

	_, err = ctl.issueServiceClient.BatchUpdateIssuesStatus(ctx, &v1pb.BatchUpdateIssuesStatusRequest{
		Parent: project.Name,
		Issues: []string{issue.Name},
		Status: v1pb.IssueStatus_DONE,
	})
	if err != nil {
		return err
	}
	// Add a second sleep to avoid schema version conflict.
	time.Sleep(time.Second)
	return nil
}

func (ctl *controller) createDatabaseFromBackup(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, databaseName string, owner string, labels map[string]string, backup *v1pb.Backup) error {
	characterSet, collation := "utf8mb4", "utf8mb4_general_ci"
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet = "UTF8"
		collation = "en_US.UTF-8"
	}

	plan, err := ctl.rolloutServiceClient.CreatePlan(ctx, &v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Steps: []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Config: &v1pb.Plan_Spec_RestoreDatabaseConfig{
								RestoreDatabaseConfig: &v1pb.Plan_RestoreDatabaseConfig{
									Target: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
									CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
										Target:       instance.Name,
										Database:     databaseName,
										CharacterSet: characterSet,
										Collation:    collation,
										Owner:        owner,
										Labels:       labels,
									},
									Source: &v1pb.Plan_RestoreDatabaseConfig_Backup{
										Backup: backup.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Plan: plan.Name})
	if err != nil {
		return err
	}

	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        plan.Name,
			Rollout:     rollout.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return err
	}

	if err := ctl.waitRollout(ctx, rollout.Name); err != nil {
		return err
	}

	_, err = ctl.issueServiceClient.BatchUpdateIssuesStatus(ctx, &v1pb.BatchUpdateIssuesStatusRequest{
		Parent: project.Name,
		Issues: []string{issue.Name},
		Status: v1pb.IssueStatus_DONE,
	})
	if err != nil {
		return err
	}
	// Add a second sleep to avoid schema version conflict.
	time.Sleep(time.Second)
	return nil
}

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
func (ctl *controller) disableAutomaticBackup(ctx context.Context, databaseName string) error {
	if _, err := ctl.databaseServiceClient.UpdateBackupSetting(ctx, &v1pb.UpdateBackupSettingRequest{
		Setting: &v1pb.BackupSetting{
			Name:                 fmt.Sprintf("%s/backupSetting", databaseName),
			CronSchedule:         "",
			BackupRetainDuration: durationpb.New(time.Duration(7*24*60*60) * time.Second),
		},
	}); err != nil {
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
		resp, err := ctl.databaseServiceClient.ListBackups(ctx, &v1pb.ListBackupsRequest{Parent: databaseName})
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
		resp, err := ctl.databaseServiceClient.ListBackups(ctx, &v1pb.ListBackupsRequest{Parent: databaseName})
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
