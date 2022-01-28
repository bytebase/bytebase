package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

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

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  "TestSchemaUpdate",
	})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create project, error: %w", err))
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
		t.Fatal(fmt.Errorf("failed to add instance, error: %w", err))
	}

	// Expecting no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance1.ID,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get databases, error: %w", err))
	}
	if len(databases) != 0 {
		t.Fatal(fmt.Errorf("invalid number of databases %v, expecting no database", len(databases)))
	}

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	createContext, err := getDatabaseCreateIssueCreateContext(instance1.ID, databaseName)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get create database create context, error: %w", err))
	}
	_, err = ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("create database %q", databaseName),
		Type:        api.IssueDatabaseCreate,
		Description: fmt.Sprintf("This creates a database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: createContext,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create issue, error: %w", err))
	}
}
