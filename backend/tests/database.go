package tests

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) createDatabaseV2(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, environment *v1pb.Environment, databaseName string, owner string, labels map[string]string) error {
	characterSet, collation := "utf8mb4", "utf8mb4_general_ci"
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet = "UTF8"
		collation = "en_US.UTF-8"
	}
	environmentName := ""
	if environment != nil {
		environmentName = environment.Name
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
									Environment:  environmentName,
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
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        plan.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return err
	}
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Plan: plan.Name})
	if err != nil {
		return err
	}

	if err := ctl.waitRollout(ctx, issue.Name, rollout.Name); err != nil {
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
	issue, err := ctl.issueServiceClient.CreateIssue(ctx, &v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        plan.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Assignee:    fmt.Sprintf("users/%s", api.SystemBotEmail),
		},
	})
	if err != nil {
		return err
	}
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, &v1pb.CreateRolloutRequest{Parent: project.Name, Plan: plan.Name})
	if err != nil {
		return err
	}

	if err := ctl.waitRollout(ctx, issue.Name, rollout.Name); err != nil {
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
	return nil
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

	slog.Debug("Waiting for backup.", slog.String("id", backupName))
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

	slog.Debug("Waiting for backup.", slog.String("id", backupName))
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
