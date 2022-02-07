package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/kr/pretty"
)

var (
	migrationStatement = `
	CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	schemaSQLResult = `[{"name":"book","rootpage":"2","sql":"CREATE TABLE book (\n\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\t\tname TEXT NOT NULL\n\t)","tbl_name":"book","type":"table"}]`
)

func TestSchemaUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()
	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
	if err := ctl.setLicense(); err != nil {
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
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	if err != nil {
		t.Fatal(err)
	}

	environments, err := ctl.getEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	prodEnvironment, err := findEnvironment(environments, "Prod")
	if err != nil {
		t.Fatal(err)
	}

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
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
		t.Fatalf("invalid number of databases %v in project %v, expecting no database", len(databases), project.ID)
	}
	// Expecting instance to have no database.
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
	})
	if err != nil {
		t.Fatalf("failed to get databases, error: %v", err)
	}
	if len(databases) != 0 {
		t.Fatalf("invalid number of databases %v in instance %v, expecting no database", len(databases), instance.ID)
	}

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	if err := ctl.createDatabase(project, instance, databaseName, nil /* labelMap */); err != nil {
		t.Fatal(err)
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
	if database.Instance.ID != instance.ID {
		t.Fatalf("expect database %v name %q to be in instance %v, got %v", database.ID, database.Name, instance.ID, database.Instance.ID)
	}

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
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
	issue, err := ctl.createIssue(api.IssueCreate{
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
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Query schema.
	result, err := ctl.query(instance, databaseName)
	if err != nil {
		t.Fatal(err)
	}
	if schemaSQLResult != result {
		t.Fatalf("SQL result want %q, got %q, diff %q", schemaSQLResult, result, pretty.Diff(schemaSQLResult, result))
	}
}

func TestVCSSchemaUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()
	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
	if err := ctl.setLicense(); err != nil {
		t.Fatal(err)
	}

	// Create a VCS.
	applicationID := "testApplicationID"
	applicationSecret := "testApplicationSecret"
	vcs, err := ctl.createVCS(api.VCSCreate{
		Name:          "TestVCS",
		Type:          vcs.GitLabSelfHost,
		InstanceURL:   ctl.gitURL,
		APIURL:        ctl.gitAPIURL,
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
		WebURL:             fmt.Sprintf("%s/%s", ctl.gitURL, repositoryPath),
		BranchFilter:       "feature/foo",
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

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	if err != nil {
		t.Fatal(err)
	}

	environments, err := ctl.getEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	prodEnvironment, err := findEnvironment(environments, "Prod")
	if err != nil {
		t.Fatal(err)
	}

	// Add an instance.
	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          instanceName,
		Engine:        db.SQLite,
		Host:          instanceDir,
	})
	if err != nil {
		t.Fatalf("failed to add instance, error: %v", err)
	}

	// Create an issue that creates a database.
	databaseName := "testVCSSchemaUpdate"
	if err := ctl.createDatabase(project, instance, databaseName, nil /* labelMap */); err != nil {
		t.Fatal(err)
	}

	// Simulate Git commits.
	gitFile := "bbtest/Prod/testVCSSchemaUpdate__ver1__migrate__create_a_test_table.sql"
	pushEvent := &gitlab.WebhookPushEvent{
		ObjectKind: gitlab.WebhookPush,
		Ref:        "refs/heads/feature/foo",
		Project: gitlab.WebhookProject{
			ID: gitlabProjectID,
		},
		CommitList: []gitlab.WebhookCommit{
			{
				Timestamp: "2021-01-13T13:14:00Z",
				AddedList: []string{
					gitFile,
				},
			},
		},
	}
	if err := ctl.gitlab.AddFiles(gitlabProjectIDStr, map[string]string{gitFile: migrationStatement}); err != nil {
		t.Fatalf("failed to add files to gitlab project %v, error %v", gitlabProjectID, err)
	}
	if err := ctl.gitlab.SendCommits(gitlabProjectIDStr, pushEvent); err != nil {
		t.Fatalf("failed to send commits to gitlab project %v, error %v", gitlabProjectID, err)
	}

	// Get schema update issue.
	openStatus := []api.IssueStatus{api.IssueOpen}
	issues, err := ctl.getIssues(api.IssueFind{ProjectID: &project.ID, StatusList: &openStatus})
	if err != nil {
		t.Fatalf("failed to get open issues for project %v, error: %v", project.ID, err)
	}
	if len(issues) != 1 {
		t.Fatalf("invalid number of open issues %v in project %v, expecting one issue", len(issues), project.ID)
	}
	issue := issues[0]
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		t.Fatalf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		t.Fatalf("issue %v pipeline %v is expected to finish with status done got %v", issue.ID, issue.Pipeline.ID, status)
	}

	// Query schema.
	result, err := ctl.query(instance, databaseName)
	if err != nil {
		t.Fatal(err)
	}
	if schemaSQLResult != result {
		t.Fatalf("SQL result want %q, got %q, diff %q", schemaSQLResult, result, pretty.Diff(schemaSQLResult, result))
	}
}
