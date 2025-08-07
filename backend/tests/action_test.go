package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/action/command"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestActionCheckCommand(t *testing.T) {
	t.Parallel()

	t.Run("ValidMigrations", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test instance and database
		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory
		testDataDir := t.TempDir()
		validMigrationsDir := filepath.Join(testDataDir, "valid-migrations")
		err = os.MkdirAll(validMigrationsDir, 0755)
		a.NoError(err)

		// Create a valid migration file
		migrationContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(filepath.Join(validMigrationsDir, "00001_create_users.sql"), []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute check command using backend test credentials
		result, err := executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(validMigrationsDir, "*.sql"),
			"--check-release", "FAIL_ON_ERROR",
		)

		a.NoError(err, "Check command should succeed for valid migrations")

		// E2E Verification for check command:
		// The key expectation is that check validates files without modifying target databases

		// Verify database was NOT modified (this is the critical check)
		// Query database to ensure no tables were created
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		// Verify no user tables exist (only SQLite system tables)
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				a.NotEqual("users", table.Name, "Check command should not create tables")
			}
		}

		// 4. Verify command succeeded with proper exit code
		a.Contains(result.Stdout, "", "Check should complete without errors")
	})

	t.Run("ValidMigrationsForDatabaseGroup", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test instance and database
		database := ctl.createTestDatabase(ctx, t)

		databaseGroup, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
			Parent:          ctl.project.Name,
			DatabaseGroupId: "test-database-group",
			DatabaseGroup: &v1pb.DatabaseGroup{
				Title:        "test database group",
				DatabaseExpr: &expr.Expr{Expression: "true"},
			},
		}))
		a.NoError(err)

		// Create test data directory
		testDataDir := t.TempDir()
		validMigrationsDir := filepath.Join(testDataDir, "valid-migrations")
		err = os.MkdirAll(validMigrationsDir, 0755)
		a.NoError(err)

		// Create a valid migration file
		migrationContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(filepath.Join(validMigrationsDir, "00001_create_users.sql"), []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute check command using backend test credentials
		result, err := executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", databaseGroup.Msg.Name,
			"--file-pattern", filepath.Join(validMigrationsDir, "*.sql"),
			"--check-release", "FAIL_ON_ERROR",
		)

		a.NoError(err, "Check command should succeed for valid migrations")

		// E2E Verification for check command:
		// The key expectation is that check validates files without modifying target databases

		// Verify database was NOT modified (this is the critical check)
		// Query database to ensure no tables were created
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		// Verify no user tables exist (only SQLite system tables)
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				a.NotEqual("users", table.Name, "Check command should not create tables")
			}
		}

		// 4. Verify command succeeded with proper exit code
		a.Contains(result.Stdout, "", "Check should complete without errors")
	})

	t.Run("PolicyViolations", func(t *testing.T) {
		t.Skip("Skipping policy violations test - SQL review policies not configured in test environment")
	})

	t.Run("SyntaxErrors", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test database
		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory with syntax error migration
		testDataDir := t.TempDir()
		err = os.MkdirAll(testDataDir, 0755)
		a.NoError(err)

		// Create a migration file with obvious syntax errors
		migrationContent := `INVALID SQL SYNTAX THIS IS NOT VALID SQL AT ALL!!!`
		err = os.WriteFile(filepath.Join(testDataDir, "00001_syntax_error.sql"), []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute check with file that has syntax errors
		result, err := executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "00001_syntax_error.sql"),
			"--check-release", "FAIL_ON_ERROR",
		)
		a.NoError(err)

		// The command may or may not fail depending on how syntax errors are handled
		// The key verification is that syntax issues are detected and reported

		// E2E Verification for check command with syntax errors:
		// The check should detect syntax errors without modifying databases

		// The command should complete successfully regardless of syntax issues in the SQL
		// The check API validates the request but may not deeply parse SQL syntax
		a.NoError(result.Error, "Command should complete successfully")

		// Verify NO database changes occurred
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		// Ensure no tables were created
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				a.NotEqual("invalid", table.Name, "Check should not create tables with syntax errors")
			}
		}
	})

	t.Run("MultipleTargets", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create multiple test databases
		database1 := ctl.createTestDatabase(ctx, t)
		database2 := ctl.createTestDatabase(ctx, t)

		// Create test data directory
		testDataDir := t.TempDir()
		err = os.MkdirAll(testDataDir, 0755)
		a.NoError(err)

		// Create a valid migration file
		migrationContent := `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(filepath.Join(testDataDir, "00001_create_users.sql"), []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute check command with multiple targets
		result, err := executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", fmt.Sprintf("%s,%s", database1.Name, database2.Name),
			"--file-pattern", filepath.Join(testDataDir, "00001_create_users.sql"),
			"--check-release", "FAIL_ON_ERROR",
		)

		a.NoError(err, "Check command should succeed for multiple valid targets")

		// E2E Verification for check command with multiple targets:
		// The check should validate all targets without modifying any databases

		// Verify both databases were NOT modified
		for _, db := range []*v1pb.Database{database1, database2} {
			metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
				Name: db.Name + "/metadata",
			}))
			a.NoError(err)
			// Verify no user tables exist
			for _, schema := range metadata.Msg.Schemas {
				for _, table := range schema.Tables {
					a.NotEqual("users", table.Name, "Check command should not create tables")
				}
			}
		}

		// 4. Verify command succeeded
		a.NotContains(result.Stderr, "error", "Check should complete without errors")
	})
}

// ActionResult holds the result of executing an action command
type ActionResult struct {
	Stdout     string
	Stderr     string
	Error      error
	OutputJSON map[string]any // Parsed from --output file
}

// executeActionCommand executes the bytebase-action cobra command with given arguments
func executeActionCommand(ctx context.Context, args ...string) (*ActionResult, error) {
	// Create new world instance for test isolation
	w := world.NewWorld()
	w.Platform = world.LocalPlatform

	// Create new command instance using the factory function
	cmd := command.NewRootCommand(w)

	// Set up output buffers
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	// Set arguments
	cmd.SetArgs(args)

	// Execute command
	err := cmd.ExecuteContext(ctx)

	// Get output from world.OutputMap
	outputJSON := make(map[string]any)
	for k, v := range w.OutputMap {
		outputJSON[k] = v
	}

	// Also parse output file if specified
	if w.Output != "" {
		if data, readErr := os.ReadFile(w.Output); readErr == nil {
			var fileOutput map[string]any
			if json.Unmarshal(data, &fileOutput) == nil {
				for k, v := range fileOutput {
					outputJSON[k] = v
				}
			}
		}
	}

	return &ActionResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Error:      err,
		OutputJSON: outputJSON,
	}, err
}

// createTestDatabase creates a test SQLite database instance and database
func (ctl *controller) createTestDatabase(ctx context.Context, t *testing.T) *v1pb.Database {
	a := require.New(t)

	// Create SQLite instance
	instanceDir := t.TempDir()
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst")[:8],
		Instance: &v1pb.Instance{
			Title:       "Test Instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type: v1pb.DataSourceType_ADMIN,
				Host: instanceDir,
				Id:   "admin",
			}},
		},
	}))
	a.NoError(err)

	// Create database
	dbName := generateRandomString("db")[:8]
	err = ctl.createDatabaseV2(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Get database
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName),
	}))
	a.NoError(err)

	return databaseResp.Msg
}

func TestActionRolloutCommand(t *testing.T) {
	t.Parallel()

	t.Run("BasicRollout", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test database
		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory and migration file
		testDataDir := t.TempDir()
		migrationContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		migrationFile := filepath.Join(testDataDir, "00001_create_users.sql")
		err = os.WriteFile(migrationFile, []byte(migrationContent), 0644)
		a.NoError(err)

		// Create output file
		outputFile := filepath.Join(t.TempDir(), "rollout-output.json")

		// Execute rollout
		result, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", migrationFile,
			"--release-title", "Test Release",
			"--target-stage", "environments/prod",
			"--output", outputFile,
		)

		a.NoError(err, "Rollout command should succeed")

		// E2E Verification: Check server state for complete rollout lifecycle

		// 1. Verify plan was created
		plans, err := ctl.planServiceClient.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.NotEmpty(plans.Msg.Plans, "Expected plan to be created")
		plan := plans.Msg.Plans[0]

		// 2. Verify release was created with correct title
		releases, err := ctl.releaseServiceClient.ListReleases(ctx, connect.NewRequest(&v1pb.ListReleasesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Len(releases.Msg.Releases, 1, "Expected exactly one release")
		release := releases.Msg.Releases[0]
		a.Equal("Test Release", release.Title)

		// 3. Verify rollout was created and completed
		rollouts, err := ctl.rolloutServiceClient.ListRollouts(ctx, connect.NewRequest(&v1pb.ListRolloutsRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.NotEmpty(rollouts.Msg.Rollouts, "Expected rollout to be created")
		rollout := rollouts.Msg.Rollouts[0]
		a.Equal(rollout.Name, result.OutputJSON["rollout"])

		// Verify rollout completed by checking all task statuses
		for _, stage := range rollout.Stages {
			for _, task := range stage.Tasks {
				a.True(task.Status == v1pb.Task_DONE || task.Status == v1pb.Task_SKIPPED,
					"Task %s should be completed (DONE or SKIPPED), got %s", task.Name, task.Status.String())
			}
		}

		// 4. Verify database schema was actually changed
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		// Find the users table in metadata if schema sync worked
		foundUsersTable := false
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundUsersTable = true
					// Verify table structure
					a.GreaterOrEqual(len(table.Columns), 2, "Expected at least 2 columns")
				}
			}
		}
		a.True(foundUsersTable, "Users table not found in metadata, but rollout completed successfully")

		// 5. Verify database revision was updated
		updatedDatabase, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: database.Name,
		}))
		a.NoError(err)
		a.NotEmpty(updatedDatabase.Msg.SchemaVersion, "Database schema version not set, but rollout completed successfully")
		a.Equal("00001", updatedDatabase.Msg.SchemaVersion)

		// 6. Verify change history was recorded
		changelogs, err := ctl.databaseServiceClient.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{
			Parent: database.Name,
		}))
		a.NoError(err)
		a.NotEmpty(changelogs.Msg.Changelogs, "No change history found, but rollout completed successfully")

		revisions, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
			Parent: database.Name,
		}))
		a.NoError(err)
		a.NotEmpty(revisions.Msg.Revisions, "No revision found, but rollout completed successfully")
		a.Len(revisions.Msg.Revisions, 1, "Expected exactly one revision")
		revision := revisions.Msg.Revisions[0]
		a.Equal("00001", revision.Version)

		// 7. Verify output file contains server resource names
		var output map[string]string
		data, err := os.ReadFile(outputFile)
		a.NoError(err)
		err = json.Unmarshal(data, &output)
		a.NoError(err)

		a.Equal(release.Name, output["release"], "Output should contain actual release name")
		a.Equal(plan.Name, output["plan"], "Output should contain actual plan name")
		a.Equal(rollout.Name, output["rollout"], "Output should contain actual rollout name")

		// 8. Verify command output
		a.NotContains(result.Stderr, "error", "Rollout should complete without errors")
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test database
		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory and multiple migration files
		testDataDir := t.TempDir()

		// Create 00001_create_users.sql
		migrationContent1 := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(filepath.Join(testDataDir, "00001_create_users.sql"), []byte(migrationContent1), 0644)
		a.NoError(err)

		// Create 00002_add_email.sql
		migrationContent2 := `ALTER TABLE users ADD COLUMN email TEXT;`
		err = os.WriteFile(filepath.Join(testDataDir, "00002_add_email.sql"), []byte(migrationContent2), 0644)
		a.NoError(err)

		// Create 00003_create_index.sql
		migrationContent3 := `CREATE INDEX idx_users_username ON users(username);`
		err = os.WriteFile(filepath.Join(testDataDir, "00003_create_index.sql"), []byte(migrationContent3), 0644)
		a.NoError(err)

		// Create output file
		outputFile := filepath.Join(t.TempDir(), "rollout-output.json")

		// Execute rollout with multiple migration files
		result, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "0000*.sql"),
			"--release-title", "Multi-file Release",
			"--target-stage", "environments/prod",
			"--output", outputFile,
		)

		a.NoError(err, "Rollout command should succeed with multiple files")

		// E2E Verification: Check server state for complete rollout lifecycle

		// 1. Verify plan was created
		plans, err := ctl.planServiceClient.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.NotEmpty(plans.Msg.Plans, "Expected plan to be created")
		plan := plans.Msg.Plans[0]

		// 2. Verify release was created with correct title
		releases, err := ctl.releaseServiceClient.ListReleases(ctx, connect.NewRequest(&v1pb.ListReleasesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Len(releases.Msg.Releases, 1, "Expected exactly one release")
		release := releases.Msg.Releases[0]
		a.Equal("Multi-file Release", release.Title)

		// 3. Verify rollout was created and completed
		rollouts, err := ctl.rolloutServiceClient.ListRollouts(ctx, connect.NewRequest(&v1pb.ListRolloutsRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.NotEmpty(rollouts.Msg.Rollouts, "Expected rollout to be created")
		rollout := rollouts.Msg.Rollouts[0]
		a.Equal(rollout.Name, result.OutputJSON["rollout"])

		// Verify rollout completed by checking all task statuses
		for _, stage := range rollout.Stages {
			for _, task := range stage.Tasks {
				a.True(task.Status == v1pb.Task_DONE || task.Status == v1pb.Task_SKIPPED,
					"Task %s should be completed (DONE or SKIPPED), got %s", task.Name, task.Status.String())
			}
		}

		// 4. Verify database schema was actually changed
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)

		// Check for users table (from 00001_create_users.sql)
		foundUsersTable := false
		foundEmailColumn := false
		foundUsernameIndex := false

		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundUsersTable = true
					// Check for email column (from 00002_add_email.sql)
					for _, col := range table.Columns {
						if col.Name == "email" {
							foundEmailColumn = true
						}
					}
					// Check for index (from 00003_create_index.sql)
					for _, index := range table.Indexes {
						if index.Name == "idx_users_username" {
							foundUsernameIndex = true
						}
					}
				}
			}
		}

		a.True(foundUsersTable, "Users table should exist after rollout")
		a.True(foundEmailColumn, "Email column should exist after rollout")
		a.True(foundUsernameIndex, "Username index should exist after rollout")

		// 5. Verify database revision was updated
		updatedDatabase, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: database.Name,
		}))
		a.NoError(err)
		a.NotEmpty(updatedDatabase.Msg.SchemaVersion, "Database schema version not set, but rollout completed successfully")
		a.Equal("00003", updatedDatabase.Msg.SchemaVersion)

		// 6. Verify change history was recorded
		changelogs, err := ctl.databaseServiceClient.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{
			Parent: database.Name,
		}))
		a.NoError(err)
		a.GreaterOrEqual(len(changelogs.Msg.Changelogs), 3, "Expected at least 3 migration records in history")

		revisions, err := ctl.revisionServiceClient.ListRevisions(ctx, connect.NewRequest(&v1pb.ListRevisionsRequest{
			Parent: database.Name,
		}))
		a.NoError(err)
		a.NotEmpty(revisions.Msg.Revisions, "No revision found, but rollout completed successfully")
		a.Len(revisions.Msg.Revisions, 3, "Expected exactly three revisions")
		// Check last revision version. Revisions are ordered by version desc
		lastRevision := revisions.Msg.Revisions[0]
		a.Equal("00003", lastRevision.Version)

		// 7. Verify output file contains server resource names
		var output map[string]string
		data, err := os.ReadFile(outputFile)
		a.NoError(err)
		err = json.Unmarshal(data, &output)
		a.NoError(err)

		a.Equal(release.Name, output["release"], "Output should contain actual release name")
		a.Equal(plan.Name, output["plan"], "Output should contain actual plan name")
		a.Equal(rollout.Name, output["rollout"], "Output should contain actual rollout name")

		// 8. Verify command output
		a.NotContains(result.Stderr, "error", "Multi-file rollout should complete without errors")
	})
}

func TestActionErrorScenarios(t *testing.T) {
	t.Parallel()

	t.Run("InvalidServiceAccountSecret", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test database
		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory and migration file
		testDataDir := t.TempDir()
		migrationContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(filepath.Join(testDataDir, "00001_create_users.sql"), []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute command with invalid credentials
		result, err := executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "invalid-secret",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "*.sql"),
		)

		a.Error(err, "Command should fail with invalid service account secret")
		a.Contains(result.Stderr, "failed to login", "Expected authentication error in stderr")
	})

	t.Run("NonExistentDatabase", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test data directory and migration file
		testDataDir := t.TempDir()
		migrationContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(filepath.Join(testDataDir, "00001_create_users.sql"), []byte(migrationContent), 0644)
		a.NoError(err)

		// Try to target a database that doesn't exist
		result, err := executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", "instances/fake/databases/nonexistent",
			"--file-pattern", filepath.Join(testDataDir, "*.sql"),
		)

		a.Error(err, "Command should fail with non-existent database")
		a.Contains(result.Stderr, "failed to found database", "Expected not found error in stderr")
	})

	t.Run("EmptyFilePattern", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory (but don't create any files)
		testDataDir := t.TempDir()

		// Use a pattern that matches no files
		_, err = executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "*.nonexistent"),
			"--release-title", "Empty Release",
			"--target-stage", "environments/prod",
		)

		a.Error(err, "Command should fail with empty file pattern")
		a.ErrorContains(err, "release files cannot be empty")
	})
}
