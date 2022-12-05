package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestExternalApprovalFeishu_AllUserCanBeFound(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:                 dataDir,
		vcsProviderCreator:      fake.NewGitLab,
		feishuProverdierCreator: fake.NewFeishu,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)

	err = ctl.setLicense()
	a.NoError(err)

	// close existing issues
	issues, err := ctl.getIssues(api.IssueFind{})
	a.NoError(err)
	for _, issue := range issues {
		patchedIssue, err := ctl.patchIssueStatus(api.IssueStatusPatch{
			ID:     issue.ID,
			Status: api.IssueCanceled,
		})
		a.NoError(err)
		a.Equal(api.IssueCanceled, patchedIssue.Status)
	}

	err = ctl.patchSetting(api.SettingPatch{
		Name:  api.SettingAppIM,
		Value: `{"imType":"im.feishu","appId":"123","appSecret":"123","externalApproval":{"enabled":true}}`,
	})
	a.NoError(err)

	// Create a DBA account.
	dba, err := ctl.createPrincipal(api.PrincipalCreate{
		Type:  api.EndUser,
		Name:  "DBA",
		Email: "dba@dba.com",
	})
	a.NoError(err)

	_, err = ctl.createMember(api.MemberCreate{
		Status:      api.Active,
		Role:        api.DBA,
		PrincipalID: dba.ID,
	})
	a.NoError(err)

	err = ctl.feishuProvider.RegisterEmails("demo@example.com", "dba@dba.com")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  "TestExternalApprovalFeishu",
	})
	a.NoError(err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	a.NoError(err)

	// Expecting project to have no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))
	// Expecting instance to have no database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(project, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	// Expecting project to have 1 database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))
	database := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     migrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    dba.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	attention := true
	issue, err = ctl.patchIssue(api.IssuePatch{
		ID:                    issue.ID,
		AssigneeNeedAttention: &attention,
	})
	a.NoError(err)
	a.Equal(true, issue.AssigneeNeedAttention)

	// Sleep for 65 seconds, giving time to ApplicationRunner to create external approvals.
	time.Sleep(65 * time.Second)
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
	status, err := ctl.waitIssuePipelineWithNoApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
}

func TestExternalApprovalFeishu_AssigneeCanBeFound(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:                 dataDir,
		vcsProviderCreator:      fake.NewGitLab,
		feishuProverdierCreator: fake.NewFeishu,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)

	err = ctl.setLicense()
	a.NoError(err)

	// close existing issues
	issues, err := ctl.getIssues(api.IssueFind{})
	a.NoError(err)
	for _, issue := range issues {
		patchedIssue, err := ctl.patchIssueStatus(api.IssueStatusPatch{
			ID:     issue.ID,
			Status: api.IssueCanceled,
		})
		a.NoError(err)
		a.Equal(api.IssueCanceled, patchedIssue.Status)
	}

	err = ctl.patchSetting(api.SettingPatch{
		Name:  api.SettingAppIM,
		Value: `{"imType":"im.feishu","appId":"123","appSecret":"123","externalApproval":{"enabled":true}}`,
	})
	a.NoError(err)

	// Create a DBA account.
	dba, err := ctl.createPrincipal(api.PrincipalCreate{
		Type:  api.EndUser,
		Name:  "DBA",
		Email: "dba@dba.com",
	})
	a.NoError(err)

	_, err = ctl.createMember(api.MemberCreate{
		Status:      api.Active,
		Role:        api.DBA,
		PrincipalID: dba.ID,
	})
	a.NoError(err)

	err = ctl.feishuProvider.RegisterEmails("dba@dba.com")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  "TestExternalApprovalFeishu",
	})
	a.NoError(err)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	a.NoError(err)

	// Expecting project to have no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))
	// Expecting instance to have no database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	a.NoError(err)
	a.Zero(len(databases))

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabase(project, instance, databaseName, "", nil /* labelMap */)
	a.NoError(err)

	// Expecting project to have 1 database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))
	database := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     migrationStatement,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     project.ID,
		Name:          fmt.Sprintf("update schema for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("This updates the schema of database %q.", databaseName),
		AssigneeID:    dba.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	attention := true
	issue, err = ctl.patchIssue(api.IssuePatch{
		ID:                    issue.ID,
		AssigneeNeedAttention: &attention,
	})
	a.NoError(err)
	a.Equal(true, issue.AssigneeNeedAttention)

	// Sleep for 65 seconds, giving time to ApplicationRunner to create external approvals.
	time.Sleep(65 * time.Second)
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
	status, err := ctl.waitIssuePipelineWithNoApproval(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)
}
