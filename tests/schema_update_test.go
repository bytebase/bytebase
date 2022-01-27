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

	// provision an instance.
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
		t.Fatal(fmt.Errorf("failed to add instance, error %w", err))
	}

	// Expecting no database.
	databases, err := ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance1.ID,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get databases, error %w", err))
	}
	if len(databases) != 0 {
		t.Fatal(fmt.Errorf("invalid number of databases %v, expecting no database", len(databases)))
	}
}
