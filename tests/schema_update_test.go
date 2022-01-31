package tests

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/kr/pretty"
)

//go:embed fake
var fakeFS embed.FS

func TestSchemaUpdate(t *testing.T) {
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()

	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}

	license, err := fs.ReadFile(fakeFS, "fake/license")
	if err != nil {
		t.Fatal(err)
	}
	err = ctl.switchPlan(&enterpriseAPI.SubscriptionPatch{
		License: string(license),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  "TestSchemaUpdate",
	})
	if err != nil {
		t.Fatalf("failed to create project, error: %v", err)
	}

	// Provision an instance.
	instanceDir := t.TempDir()
	instance1Name := "testInstance1"
	instance1Dir, err := ctl.provisionSQLiteInstance(instanceDir, instance1Name)
	if err != nil {
		t.Fatal(err)
	}

	environments, err := ctl.getEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	var prodEnvironment *api.Environment
	for _, environment := range environments {
		if environment.Name == "Prod" {
			prodEnvironment = environment
			break
		}
	}
	if prodEnvironment == nil {
		t.Fatal("unable to find prod environment")
	}

	// Add an instance.
	instance1, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instance1Name,
		Engine:        db.SQLite,
		Host:          instance1Dir,
	})
	if err != nil {
		t.Fatalf("failed to add instance, error: %v", err)
	}

	// Expecting project to have no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}
	if len(databases) != 0 {
		t.Fatalf("invalid number of databases %v in project %v, expecting no database", project.ID, len(databases))
	}
	// Expecting instance to have no database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance1.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}
	if len(databases) != 0 {
		t.Fatalf("invalid number of databases %v in instance %v, expecting no database", instance1.ID, len(databases))
	}

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	createContext, err := json.Marshal(&api.CreateDatabaseContext{
		InstanceID:   instance1.ID,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to construct database creation issue CreateContext payload, error: %w", err))
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
		t.Fatalf("failed to create database creation issue, error: %v", err)
	}

	if status, _ := getAggregatedTaskStatus(issue); status != api.TaskPendingApproval {
		t.Fatalf("issue %v pipeline %v is supposed to be pending manual approval.", issue.ID, issue.Pipeline.ID)
	}

	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Expecting project to have 1 database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}
	if len(databases) != 1 {
		t.Fatalf("invalid number of databases %v in project %v, expecting one database", project.ID, len(databases))
	}
	database := databases[0]
	if database.Instance.ID != instance1.ID {
		t.Fatalf("expect database %v name %q to be in instance %v, got %v", database.ID, database.Name, instance1.ID, database.Instance.ID)
	}

	migrationStatement := `
	CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	// Create an issue that updates database schema.
	createContext, err = json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		UpdateSchemaDetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseID: database.ID,
				Statement:  migrationStatement,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to construct schema update issue CreateContext payload, error: %v", err)
	}
	issue, err = ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("update schema for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("This updates the schema of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	if err != nil {
		t.Fatalf("failed to create schema update issue, error: %v", err)
	}
	status, err = ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Query schema.
	sqlResultSet, err := ctl.executeSQL(api.SQLExecute{
		InstanceID:   instance1.ID,
		DatabaseName: databaseName,
		Statement:    "SELECT * FROM sqlite_schema WHERE type = 'table' AND tbl_name = 'book';",
		Readonly:     true,
	})
	if err != nil {
		t.Fatalf("failed to execute SQL, error: %v", err)
	}
	if sqlResultSet.Error != "" {
		t.Fatalf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	wantResult := `[{"name":"book","rootpage":"2","sql":"CREATE TABLE book (\n\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\t\tname TEXT NOT NULL\n\t)","tbl_name":"book","type":"table"}]`
	if sqlResultSet.Data != wantResult {
		t.Fatalf("want SQL result %q, got %q, diff %q", wantResult, sqlResultSet.Data, pretty.Diff(wantResult, sqlResultSet.Data))
	}
}

func TestVCSSchemaUpdate(t *testing.T) {
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()

	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}

	// Create a VCS.
	applicationID := "testApplicationID"
	applicationSecret := "testApplicationSecret"
	vcs, err := ctl.createVCS(api.VCSCreate{
		Name:          "TestVCS",
		Type:          vcs.GitLabSelfHost,
		InstanceURL:   gitURL,
		APIURL:        gitAPIURL,
		ApplicationID: applicationID,
		Secret:        applicationSecret,
	})
	if err != nil {
		t.Fatalf("failed to create VCS, error: %v", err)
	}

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test VCS Project",
		Key:  "TestVCSSchemaUpdate",
	})
	if err != nil {
		t.Fatalf("failed to create project, error: %v", err)
	}

	// Create a repository.
	repositoryPath := "test/schemaUpdate"
	accessToken := "accessToken1"
	refreshToken := "refreshToken1"
	gitlabProjectID := 121
	gitlabProjectIDStr := fmt.Sprintf("%d", gitlabProjectID)
	// create a gitlab project.
	ctl.gitlab.CreateProject(gitlabProjectIDStr)
	_, err = ctl.createRepository(api.RepositoryCreate{
		VCSID:              vcs.ID,
		ProjectID:          project.ID,
		Name:               "Test Repository",
		FullPath:           repositoryPath,
		WebURL:             fmt.Sprintf("%s/%s", gitURL, repositoryPath),
		BranchFilter:       "master",
		BaseDirectory:      "bbtest",
		FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
		SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql",
		ExternalID:         gitlabProjectIDStr,
		AccessToken:        accessToken,
		ExpiresTs:          0,
		RefreshToken:       refreshToken,
	})
	if err != nil {
		t.Fatalf("failed to create repository, error: %v", err)
	}

	// Simulate Git commits.
	pushEvent := &gitlab.WebhookPushEvent{
		ObjectKind: gitlab.WebhookPush,
		Project: gitlab.WebhookProject{
			ID: gitlabProjectID,
		},
	}
	if err := ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent); err != nil {
		t.Fatalf("failed to send commits to gitlab project %q, error %v", gitlabProjectID, err)
	}
}
