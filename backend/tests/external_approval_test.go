package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestExternalApprovalFeishu_AllUserCanBeFound(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:                 dataDir,
		vcsProviderCreator:      fake.NewGitLab,
		feishuProverdierCreator: fake.NewFeishu,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	err = ctl.setLicense()
	a.NoError(err)

	// close existing issues
	issues, err := ctl.getIssues(nil /* projectID */)
	a.NoError(err)
	for _, issue := range issues {
		patchedIssue, err := ctl.patchIssueStatus(api.IssueStatusPatch{
			ID:     issue.ID,
			Status: api.IssueCanceled,
		})
		a.NoError(err)
		a.Equal(api.IssueCanceled, patchedIssue.Status)
	}

	_, err = ctl.settingServiceClient.SetSetting(ctx, &v1pb.SetSettingRequest{
		Setting: &v1pb.Setting{
			Name: fmt.Sprintf("settings/%s", api.SettingAppIM),
			Value: &v1pb.Value{
				Value: &v1pb.Value_AppImSettingValue{
					AppImSettingValue: &v1pb.AppIMSetting{
						ImType:    v1pb.AppIMSetting_FEISHU,
						AppId:     "123",
						AppSecret: "123",
						ExternalApproval: &v1pb.AppIMSetting_ExternalApproval{
							Enabled: true,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Create a DBA account.
	dbaUser, err := ctl.authServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "DBA",
			Email:    "dba@dba.com",
			UserRole: v1pb.UserRole_DBA,
			UserType: v1pb.UserType_USER,
			Password: "dbapass",
		},
	})
	a.NoError(err)
	dbaUserUID, err := strconv.Atoi(strings.TrimPrefix(dbaUser.Name, "users/"))
	a.NoError(err)

	err = ctl.feishuProvider.RegisterEmails("demo@example.com", "dba@dba.com")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_DEPLOYMENT_APPROVAL,
			Policy: &v1pb.Policy_DeploymentApprovalPolicy{
				DeploymentApprovalPolicy: &v1pb.DeploymentApprovalPolicy{
					DefaultStrategy: v1pb.ApprovalStrategy_MANUAL,
				},
			},
		},
	})
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "migration statement sheet",
			Content:    []byte(migrationStatement),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    databaseUID,
				SheetID:       sheetUID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    dbaUserUID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	for {
		issue, err := ctl.issueServiceClient.GetIssue(ctx, &v1pb.GetIssueRequest{
			Name: fmt.Sprintf("projects/%d/issues/%d", issue.ProjectID, issue.ID),
		})
		a.NoError(err)
		if issue.ApprovalFindingDone {
			break
		}
		time.Sleep(time.Second)
	}

	attention := true
	issue, err = ctl.patchIssue(issue.ID, api.IssuePatch{
		AssigneeNeedAttention: &attention,
	})
	a.NoError(err)
	a.Equal(true, issue.AssigneeNeedAttention)

	// Sleep for a few seconds, giving time to ApplicationRunner to create external approvals.
	time.Sleep(ctl.profile.AppRunnerInterval + 2*time.Second)
	issue, err = ctl.getIssue(issue.ID)
	a.NoError(err)
	taskStatus, err := getNextTaskStatus(issue)
	a.NoError(err)
	// The task is still waiting for approval.
	a.Equal(api.TaskPendingApproval, taskStatus)

	// Should have 1 PENDING approval on the feishu side.
	a.Equal(1, ctl.feishuProvider.PendingApprovalCount())
	// Simulate users approving on the feishu side.
	ctl.feishuProvider.ApprovePendingApprovals()

	// Waiting ApplicationRunner to approves the issue.
	status, err := ctl.waitIssuePipelineWithNoApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
}

func TestExternalApprovalFeishu_AssigneeCanBeFound(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:                 dataDir,
		vcsProviderCreator:      fake.NewGitLab,
		feishuProverdierCreator: fake.NewFeishu,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	err = ctl.setLicense()
	a.NoError(err)

	// close existing issues
	issues, err := ctl.getIssues(nil /* projectID */)
	a.NoError(err)
	for _, issue := range issues {
		patchedIssue, err := ctl.patchIssueStatus(api.IssueStatusPatch{
			ID:     issue.ID,
			Status: api.IssueCanceled,
		})
		a.NoError(err)
		a.Equal(api.IssueCanceled, patchedIssue.Status)
	}

	_, err = ctl.settingServiceClient.SetSetting(ctx, &v1pb.SetSettingRequest{
		Setting: &v1pb.Setting{
			Name: fmt.Sprintf("settings/%s", api.SettingAppIM),
			Value: &v1pb.Value{
				Value: &v1pb.Value_AppImSettingValue{
					AppImSettingValue: &v1pb.AppIMSetting{
						ImType:    v1pb.AppIMSetting_FEISHU,
						AppId:     "123",
						AppSecret: "123",
						ExternalApproval: &v1pb.AppIMSetting_ExternalApproval{
							Enabled: true,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	// Create a DBA account.
	// Create a DBA account.
	dbaUser, err := ctl.authServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "DBA",
			Email:    "dba@dba.com",
			UserRole: v1pb.UserRole_DBA,
			UserType: v1pb.UserType_USER,
			Password: "dbapass",
		},
	})
	a.NoError(err)
	dbaUserUID, err := strconv.Atoi(strings.TrimPrefix(dbaUser.Name, "users/"))
	a.NoError(err)

	err = ctl.feishuProvider.RegisterEmails("dba@dba.com")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_DEPLOYMENT_APPROVAL,
			Policy: &v1pb.Policy_DeploymentApprovalPolicy{
				DeploymentApprovalPolicy: &v1pb.DeploymentApprovalPolicy{
					DefaultStrategy: v1pb.ApprovalStrategy_MANUAL,
				},
			},
		},
	})
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "migration statement sheet",
			Content:    []byte(migrationStatement),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", project.Name)))
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    databaseUID,
				SheetID:       sheetUID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    dbaUserUID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	for {
		issue, err := ctl.issueServiceClient.GetIssue(ctx, &v1pb.GetIssueRequest{
			Name: fmt.Sprintf("projects/%d/issues/%d", issue.ProjectID, issue.ID),
		})
		a.NoError(err)
		if issue.ApprovalFindingDone {
			break
		}
		time.Sleep(time.Second)
	}

	attention := true
	issue, err = ctl.patchIssue(issue.ID, api.IssuePatch{
		AssigneeNeedAttention: &attention,
	})
	a.NoError(err)
	a.Equal(true, issue.AssigneeNeedAttention)

	// Sleep for a few seconds, giving time to ApplicationRunner to create external approvals.
	time.Sleep(ctl.profile.AppRunnerInterval + 2*time.Second)
	issue, err = ctl.getIssue(issue.ID)
	a.NoError(err)
	taskStatus, err := getNextTaskStatus(issue)
	a.NoError(err)
	// The task is still waiting for approval.
	a.Equal(api.TaskPendingApproval, taskStatus)

	// Should have 1 PENDING approval on the feishu side.
	a.Equal(1, ctl.feishuProvider.PendingApprovalCount())
	// Simulate users approving on the feishu side.
	ctl.feishuProvider.ApprovePendingApprovals()

	// Waiting ApplicationRunner to approves the issue.
	status, err := ctl.waitIssuePipelineWithNoApproval(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
}
