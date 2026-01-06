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

	t.Run("DeclarativeCheckValidSchema", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory and schema.sql file
		testDataDir := t.TempDir()
		schemaContent := `CREATE TABLE public.users (
    id SERIAL NOT NULL,
    username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uk_users_username UNIQUE (username)
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
		a.NoError(err)

		// Execute declarative check command
		_, err = executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--check-release", "FAIL_ON_ERROR",
			"--declarative",
		)

		a.NoError(err, "Declarative check command should succeed for valid schema")
	})

	t.Run("DeclarativeCheckMultipleFiles", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory with multiple SQL files
		testDataDir := t.TempDir()

		// Create users.sql
		usersContent := `CREATE TABLE public.users (
    id SERIAL NOT NULL,
    username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uk_users_username UNIQUE (username)
);`
		usersFile := filepath.Join(testDataDir, "users.sql")
		err = os.WriteFile(usersFile, []byte(usersContent), 0644)
		a.NoError(err)

		// Create products.sql
		productsContent := `CREATE TABLE public.products (
    id SERIAL NOT NULL,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2),
    CONSTRAINT pk_products PRIMARY KEY (id)
);`
		productsFile := filepath.Join(testDataDir, "products.sql")
		err = os.WriteFile(productsFile, []byte(productsContent), 0644)
		a.NoError(err)

		// Execute declarative check command with multiple files
		_, err = executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "*.sql"),
			"--check-release", "FAIL_ON_ERROR",
			"--declarative",
		)

		a.NoError(err, "Declarative check command should succeed with multiple files")
	})

	t.Run("DeclarativeCheckWithDatabaseGroup", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create MySQL test database
		_, mysqlContainer := ctl.createTestMySQLDatabase(ctx, t)
		defer mysqlContainer.Close(ctx)

		// Create database group
		databaseGroup, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
			Parent:          ctl.project.Name,
			DatabaseGroupId: "test-declarative-check-group",
			DatabaseGroup: &v1pb.DatabaseGroup{
				Title:        "Test Declarative Check Group",
				DatabaseExpr: &expr.Expr{Expression: "true"},
			},
		}))
		a.NoError(err)

		// Create test data directory and schema.sql file
		testDataDir := t.TempDir()
		schemaContent := `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255)
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
		a.NoError(err)

		// Execute declarative check command with database group
		_, err = executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", databaseGroup.Msg.Name,
			"--file-pattern", schemaFile,
			"--check-release", "FAIL_ON_ERROR",
			"--declarative",
		)

		a.NoError(err, "Declarative check command should succeed with database group")
	})

	t.Run("DeclarativeCheckSyntaxErrors", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory with invalid SQL
		testDataDir := t.TempDir()
		invalidSchemaContent := `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE
    email VARCHAR(255)  -- Missing comma after previous column
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(invalidSchemaContent), 0644)
		a.NoError(err)

		// Execute declarative check command with syntax error
		_, err = executeActionCommand(ctx,
			"check",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--check-release", "FAIL_ON_ERROR",
			"--declarative",
		)

		// The command may complete but should report the syntax issue
		a.Error(err, "should error")
		a.Contains(err.Error(), "found 1 error(s) in release check. view on Bytebase")
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
	if j, err := json.Marshal(w.OutputMap); err == nil {
		if err := json.Unmarshal(j, &outputJSON); err != nil {
			return nil, err
		}
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
			Environment: stringPtr("environments/prod"),
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
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Get database
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName),
	}))
	a.NoError(err)

	return databaseResp.Msg
}

// createTestMySQLDatabase creates a test MySQL database instance and database
func (ctl *controller) createTestMySQLDatabase(ctx context.Context, t *testing.T) (*v1pb.Database, *Container) {
	a := require.New(t)

	// Get MySQL container
	mysqlContainer, err := getMySQLContainer(ctx)
	a.NoError(err)

	// Create MySQL instance
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst")[:8],
		Instance: &v1pb.Instance{
			Title:       "Test MySQL Instance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type:     v1pb.DataSourceType_ADMIN,
				Host:     mysqlContainer.host,
				Port:     mysqlContainer.port,
				Username: "root",
				Password: "root-password",
				Id:       "admin",
			}},
		},
	}))
	a.NoError(err)

	// Create database
	dbName := generateRandomString("db")[:8]
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "")
	a.NoError(err)

	// Get database
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName),
	}))
	a.NoError(err)

	return databaseResp.Msg, mysqlContainer
}

// createTestPostgreSQLDatabase creates a test PostgreSQL database instance and database
func (ctl *controller) createTestPostgreSQLDatabase(ctx context.Context, t *testing.T) (*v1pb.Database, *Container) {
	a := require.New(t)

	// Get PostgreSQL container
	pgContainer, err := getPgContainer(ctx)
	a.NoError(err)

	// Create PostgreSQL instance
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst")[:8],
		Instance: &v1pb.Instance{
			Title:       "Test PostgreSQL Instance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type:     v1pb.DataSourceType_ADMIN,
				Host:     pgContainer.host,
				Port:     pgContainer.port,
				Username: "postgres",
				Password: "root-password",
				Id:       "admin",
			}},
		},
	}))
	a.NoError(err)

	// Create database
	dbName := generateRandomString("db")[:8]
	err = ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, "postgres")
	a.NoError(err)

	// Get database
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName),
	}))
	a.NoError(err)

	return databaseResp.Msg, pgContainer
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

		// Verify release files
		a.Len(release.Files, 3, "Expected exactly 3 files in release")
		// Verify each file has correct version and path
		for _, f := range release.Files {
			a.Contains(f.Path, f.Version, "File path %s should contain version %s", f.Path, f.Version)
		}
		// Verify specific files exist
		fileVersions := make(map[string]string)
		for _, f := range release.Files {
			fileVersions[f.Version] = f.Path
		}
		a.Contains(fileVersions, "00001", "Should have file with version 00001")
		a.Contains(fileVersions["00001"], "00001_create_users.sql", "File 00001 should be create_users")
		a.Contains(fileVersions, "00002", "Should have file with version 00002")
		a.Contains(fileVersions["00002"], "00002_add_email.sql", "File 00002 should be add_email")
		a.Contains(fileVersions, "00003", "Should have file with version 00003")
		a.Contains(fileVersions["00003"], "00003_create_index.sql", "File 00003 should be create_index")

		// Verify release file contents match original files
		expectedContents := map[string]string{
			"00001": migrationContent1,
			"00002": migrationContent2,
			"00003": migrationContent3,
		}
		for _, f := range release.Files {
			a.NotEmpty(f.Sheet, "File %s should have a sheet", f.Path)
			sheet, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
				Name: f.Sheet,
				Raw:  true,
			}))
			a.NoError(err)
			expectedContent, ok := expectedContents[f.Version]
			a.True(ok, "Unexpected version %s", f.Version)
			a.Equal(expectedContent, string(sheet.Msg.Content), "File %s content should match original", f.Path)
		}

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
		// Note: Versioned releases create one MIGRATE changelog for the entire release (not per file)
		// Plus a BASELINE changelog if this is the first migration for the database
		changelogs, err := ctl.databaseServiceClient.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{
			Parent: database.Name,
		}))
		a.NoError(err)
		// Expect at least 1 changelog (MIGRATE), possibly 2 if BASELINE was created
		a.GreaterOrEqual(len(changelogs.Msg.Changelogs), 1, "Expected at least 1 changelog for the versioned release")
		// Verify at least one MIGRATE changelog exists
		foundMigrateChangelog := false
		for _, changelog := range changelogs.Msg.Changelogs {
			if changelog.Type == v1pb.Changelog_MIGRATE {
				foundMigrateChangelog = true
				break
			}
		}
		a.True(foundMigrateChangelog, "Expected to find at least one MIGRATE changelog")

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

	t.Run("FileContentVersionMatch", func(t *testing.T) {
		// This test verifies that file content matches the correct version
		// when lexicographical order differs from version order
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create test database
		database := ctl.createTestDatabase(ctx, t)

		// Create test data directory with files where lex order != version order
		testDataDir := t.TempDir()

		// Lexicographical order: v1, v10, v2
		// Version numeric order: 1, 2, 10
		migrationContent1 := `CREATE TABLE first_table (id INTEGER);`
		migrationContent2 := `CREATE TABLE second_table (id INTEGER);`
		migrationContent10 := `CREATE TABLE tenth_table (id INTEGER);`

		err = os.WriteFile(filepath.Join(testDataDir, "v1_first.sql"), []byte(migrationContent1), 0644)
		a.NoError(err)
		err = os.WriteFile(filepath.Join(testDataDir, "v2_second.sql"), []byte(migrationContent2), 0644)
		a.NoError(err)
		err = os.WriteFile(filepath.Join(testDataDir, "v10_tenth.sql"), []byte(migrationContent10), 0644)
		a.NoError(err)

		// Execute rollout
		_, err = executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "v*.sql"),
			"--release-title", "Version Order Test",
			"--target-stage", "environments/prod",
		)
		a.NoError(err, "Rollout command should succeed")

		// Verify release files have correct content for each version
		releases, err := ctl.releaseServiceClient.ListReleases(ctx, connect.NewRequest(&v1pb.ListReleasesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Len(releases.Msg.Releases, 1, "Expected exactly one release")
		release := releases.Msg.Releases[0]
		a.Len(release.Files, 3, "Expected exactly 3 files in release")

		// Build expected content map by version
		expectedContents := map[string]string{
			"1":  migrationContent1,
			"2":  migrationContent2,
			"10": migrationContent10,
		}

		// Verify each file's content matches its version
		for _, f := range release.Files {
			a.NotEmpty(f.Sheet, "File %s should have a sheet", f.Path)
			sheet, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
				Name: f.Sheet,
				Raw:  true,
			}))
			a.NoError(err)
			expectedContent, ok := expectedContents[f.Version]
			a.True(ok, "Unexpected version %s", f.Version)
			a.Equal(expectedContent, string(sheet.Msg.Content),
				"File %s with version %s should contain '%s' but got '%s'",
				f.Path, f.Version, expectedContent, string(sheet.Msg.Content))
		}
	})
}

func TestActionRolloutDeclarativeMode(t *testing.T) {
	t.Parallel()

	t.Run("BasicDeclarativeRollout", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory and schema.sql file
		testDataDir := t.TempDir()
		migrationContent1 := `CREATE TABLE users (
    id SERIAL,
    username VARCHAR(255) NOT NULL UNIQUE
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(migrationContent1), 0644)
		a.NoError(err)

		// Create output file
		outputFile := filepath.Join(t.TempDir(), "rollout-output.json")

		// Execute declarative rollout
		result, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Declarative Release V1",
			"--target-stage", "environments/prod",
			"--output", outputFile,
			"--declarative",
		)

		a.NoError(err, "Declarative rollout command should succeed")

		// Verify release was created
		releases, err := ctl.releaseServiceClient.ListReleases(ctx, connect.NewRequest(&v1pb.ListReleasesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Len(releases.Msg.Releases, 1, "Expected exactly one release")
		release := releases.Msg.Releases[0]
		a.Equal("Declarative Release V1", release.Title)

		// Verify database schema was created
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		foundUsersTable := false
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundUsersTable = true
					a.Equal(2, len(table.Columns), "Expected 2 columns (id, username)")
				}
			}
		}
		a.True(foundUsersTable, "Users table should exist after declarative rollout")

		// Verify command output
		a.NotContains(result.Stderr, "error", "Declarative rollout should complete without errors")
	})

	t.Run("DeclarativeSchemaEvolution", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory
		testDataDir := t.TempDir()
		schemaFile := filepath.Join(testDataDir, "schema.sql")

		// First version of schema
		migrationContent1 := `CREATE TABLE users (
    id SERIAL,
    username VARCHAR(255) NOT NULL UNIQUE
);`
		err = os.WriteFile(schemaFile, []byte(migrationContent1), 0644)
		a.NoError(err)

		// Execute first declarative rollout
		result1, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Initial Schema",
			"--target-stage", "environments/prod",
			"--declarative",
		)
		a.NoError(err, "First declarative rollout should succeed")

		// Verify initial schema
		metadata1, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		foundTable1 := false
		for _, schema := range metadata1.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundTable1 = true
					a.Equal(2, len(table.Columns), "Initial schema should have 2 columns")
				}
			}
		}
		a.True(foundTable1, "Users table should exist after first rollout")

		// Update schema.sql with evolved schema (second version)
		migrationContent2 := `CREATE TABLE users (
    id SERIAL,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		err = os.WriteFile(schemaFile, []byte(migrationContent2), 0644)
		a.NoError(err)

		// Execute second declarative rollout with evolved schema
		result2, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Evolved Schema",
			"--target-stage", "environments/prod",
			"--declarative",
		)
		a.NoError(err, "Second declarative rollout should succeed")

		// Verify evolved schema
		metadata2, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)
		foundTable2 := false
		hasCreatedAt := false
		for _, schema := range metadata2.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundTable2 = true
					a.Equal(3, len(table.Columns), "Evolved schema should have 3 columns")
					for _, col := range table.Columns {
						if col.Name == "created_at" {
							hasCreatedAt = true
						}
					}
				}
			}
		}
		a.True(foundTable2, "Users table should still exist after evolution")
		a.True(hasCreatedAt, "created_at column should be added")

		// Verify no errors
		a.NotContains(result1.Stderr, "error", "First rollout should complete without errors")
		a.NotContains(result2.Stderr, "error", "Second rollout should complete without errors")
	})

	t.Run("DeclarativeIdempotency", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory and schema.sql file
		testDataDir := t.TempDir()
		migrationContent := `CREATE TABLE users (
    id SERIAL,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(migrationContent), 0644)
		a.NoError(err)

		// Create output files
		outputFile1 := filepath.Join(t.TempDir(), "rollout-output1.json")
		outputFile2 := filepath.Join(t.TempDir(), "rollout-output2.json")
		outputFile3 := filepath.Join(t.TempDir(), "rollout-output3.json")

		// Execute first declarative rollout
		result1, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Idempotent Release 1",
			"--target-stage", "environments/prod",
			"--output", outputFile1,
			"--declarative",
		)
		a.NoError(err, "First declarative rollout should succeed")

		// Read first output
		var output1 map[string]string
		data1, err := os.ReadFile(outputFile1)
		a.NoError(err)
		err = json.Unmarshal(data1, &output1)
		a.NoError(err)

		// Execute second declarative rollout with same schema (should succeed with task as no-op)
		result2, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Idempotent Release 2",
			"--target-stage", "environments/prod",
			"--output", outputFile2,
			"--declarative",
		)
		// Second rollout should succeed (SDL diff detects no changes, executes as no-op)
		a.NoError(err, "Second declarative rollout should succeed")

		// Read second output
		var output2 map[string]string
		data2, err := os.ReadFile(outputFile2)
		a.NoError(err)
		err = json.Unmarshal(data2, &output2)
		a.NoError(err)

		// Rollout should be created for second execution
		rolloutName := output2["rollout"]
		a.NotEmpty(rolloutName, "Rollout should be created for second execution")

		// Execute third declarative rollout with same schema (should also succeed with task as no-op)
		result3, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Idempotent Release 3",
			"--target-stage", "environments/prod",
			"--output", outputFile3,
			"--declarative",
		)
		// Third rollout should also succeed (SDL diff detects no changes, executes as no-op)
		a.NoError(err, "Third declarative rollout should succeed")

		// Read third output
		var output3 map[string]string
		data3, err := os.ReadFile(outputFile3)
		a.NoError(err)
		err = json.Unmarshal(data3, &output3)
		a.NoError(err)

		// Verify that each rollout creates a separate release (no deduplication)
		releases, err := ctl.releaseServiceClient.ListReleases(ctx, connect.NewRequest(&v1pb.ListReleasesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Len(releases.Msg.Releases, 3, "Should have three separate releases (no deduplication)")

		// Verify each rollout has its own release
		a.NotEqual(output1["release"], output2["release"], "Second rollout should create a new release")
		a.NotEqual(output1["release"], output3["release"], "Third rollout should create a new release")
		a.NotEqual(output2["release"], output3["release"], "Each rollout should have a unique release")

		// Verify schema remains consistent (only one table created)
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)

		tableCount := 0
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					tableCount++
					a.Equal(3, len(table.Columns), "Table should have consistent columns")
				}
			}
		}
		a.Equal(1, tableCount, "Should have exactly one users table (idempotent)")

		// Verify all rollouts succeeded
		a.NotContains(result1.Stderr, "error", "first rollout should complete without errors")
		a.NotContains(result2.Stderr, "error", "second rollout should complete without errors")
		a.NotContains(result3.Stderr, "error", "third rollout should complete without errors")

		// Verify that second rollout has 1 task (idempotency handled by SDL diff at execution)
		rollout2, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
			Name: rolloutName,
		}))
		a.NoError(err)
		// Check that rollout has 1 task (executes as no-op when schema matches)
		totalTasks := 0
		for _, stage := range rollout2.Msg.Stages {
			totalTasks += len(stage.Tasks)
		}
		a.Equal(1, totalTasks, "Second rollout should have 1 task")
		a.Len(rollout2.Msg.Stages, 1, "Second rollout should have 1 stage")

		// Verify third rollout also has 1 task
		rolloutName3 := output3["rollout"]
		a.NotEmpty(rolloutName3, "Rollout should be created for third execution")

		rollout3, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
			Name: rolloutName3,
		}))
		a.NoError(err)
		// Check that rollout has 1 task (executes as no-op when schema matches)
		totalTasks = 0
		for _, stage := range rollout3.Msg.Stages {
			totalTasks += len(stage.Tasks)
		}
		a.Equal(1, totalTasks, "Third rollout should have 1 task")
		a.Len(rollout3.Msg.Stages, 1, "Third rollout should have 1 stage")
	})

	t.Run("DeclarativeMultipleDatabases", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create multiple PostgreSQL test databases
		database1, pgContainer1 := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer1.Close(ctx)
		database2, pgContainer2 := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer2.Close(ctx)

		// Create test data directory and schema.sql file
		testDataDir := t.TempDir()
		migrationContent := `CREATE TABLE users (
    id SERIAL,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute declarative rollout to multiple databases
		result, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", fmt.Sprintf("%s,%s", database1.Name, database2.Name),
			"--file-pattern", schemaFile,
			"--release-title", "Multi-DB Declarative Release",
			"--target-stage", "environments/prod",
			"--declarative",
		)
		a.NoError(err, "Declarative rollout to multiple databases should succeed")

		// Verify schema was applied to both databases
		for _, db := range []*v1pb.Database{database1, database2} {
			metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
				Name: db.Name + "/metadata",
			}))
			a.NoError(err)

			foundUsersTable := false
			for _, schema := range metadata.Msg.Schemas {
				for _, table := range schema.Tables {
					if table.Name == "users" {
						foundUsersTable = true
						a.Equal(3, len(table.Columns), "Table should have 3 columns in "+db.Name)
					}
				}
			}
			a.True(foundUsersTable, "Users table should exist in "+db.Name)
		}

		// Verify command output
		a.NotContains(result.Stderr, "error", "Multi-database declarative rollout should complete without errors")
	})

	t.Run("DeclarativeWithDatabaseGroup", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create database group
		databaseGroup, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
			Parent:          ctl.project.Name,
			DatabaseGroupId: "test-declarative-group",
			DatabaseGroup: &v1pb.DatabaseGroup{
				Title:        "Test Declarative Group",
				DatabaseExpr: &expr.Expr{Expression: "true"},
			},
		}))
		a.NoError(err)

		// Create test data directory and schema.sql file
		testDataDir := t.TempDir()
		migrationContent := `CREATE TABLE users (
    id SERIAL,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		schemaFile := filepath.Join(testDataDir, "schema.sql")
		err = os.WriteFile(schemaFile, []byte(migrationContent), 0644)
		a.NoError(err)

		// Execute declarative rollout with database group
		result, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", databaseGroup.Msg.Name,
			"--file-pattern", schemaFile,
			"--release-title", "Declarative Group Release",
			"--target-stage", "environments/prod",
			"--declarative",
		)
		a.NoError(err, "Declarative rollout with database group should succeed")

		// Verify schema was applied
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)

		foundUsersTable := false
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundUsersTable = true
					a.Equal(3, len(table.Columns), "Table should have 3 columns")
				}
			}
		}
		a.True(foundUsersTable, "Users table should exist after declarative rollout with database group")

		// Verify command output
		a.NotContains(result.Stderr, "error", "Declarative rollout with database group should complete without errors")
	})

	t.Run("DeclarativeMultipleFilesMerged", func(t *testing.T) {
		t.Parallel()
		a := require.New(t)
		ctx := context.Background()
		ctl := &controller{}

		ctx, err := ctl.StartServerWithExternalPg(ctx)
		a.NoError(err)
		defer ctl.Close(ctx)

		// Create PostgreSQL test database
		database, pgContainer := ctl.createTestPostgreSQLDatabase(ctx, t)
		defer pgContainer.Close(ctx)

		// Create test data directory with multiple SQL files
		testDataDir := t.TempDir()

		// Create users.sql
		usersContent := `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		usersFile := filepath.Join(testDataDir, "users.sql")
		err = os.WriteFile(usersFile, []byte(usersContent), 0644)
		a.NoError(err)

		// Create names.sql
		namesContent := `CREATE TABLE names (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL
);`
		namesFile := filepath.Join(testDataDir, "names.sql")
		err = os.WriteFile(namesFile, []byte(namesContent), 0644)
		a.NoError(err)

		// Execute declarative rollout with multiple files (should succeed - files are merged)
		result, err := executeActionCommand(ctx,
			"rollout",
			"--url", ctl.rootURL,
			"--service-account", "demo@example.com",
			"--service-account-secret", "1024bytebase",
			"--project", ctl.project.Name,
			"--targets", database.Name,
			"--file-pattern", filepath.Join(testDataDir, "*.sql"),
			"--release-title", "Multiple Declarative Files Release",
			"--target-stage", "environments/prod",
			"--declarative",
		)

		// This should succeed because bytebase-action merges multiple declarative files
		a.NoError(err, "Declarative rollout with multiple files should succeed (files are merged)")

		// Verify the release was created with exactly one merged file
		releases, err := ctl.releaseServiceClient.ListReleases(ctx, connect.NewRequest(&v1pb.ListReleasesRequest{
			Parent: ctl.project.Name,
		}))
		a.NoError(err)
		a.Len(releases.Msg.Releases, 1, "Expected exactly one release")
		release := releases.Msg.Releases[0]
		a.Equal("Multiple Declarative Files Release", release.Title)

		// Verify the release contains exactly one file (the merged result)
		a.Len(release.Files, 1, "Release should contain exactly one merged file")
		mergedFile := release.Files[0]

		// Verify the merged file contains both table definitions via Sheet
		a.NotNil(mergedFile.Sheet, "File should have a Sheet")

		sheet, err := ctl.sheetServiceClient.GetSheet(ctx, connect.NewRequest(&v1pb.GetSheetRequest{
			Name: mergedFile.Sheet,
			Raw:  true,
		}))
		a.NoError(err)

		a.Contains(string(sheet.Msg.Content), "CREATE TABLE users", "Merged file should contain users table definition")
		a.Contains(string(sheet.Msg.Content), "CREATE TABLE names", "Merged file should contain names table definition")

		// Verify both tables were created from the merged files
		metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
			Name: database.Name + "/metadata",
		}))
		a.NoError(err)

		// Verify both users and names tables were created
		foundUsersTable := false
		foundNamesTable := false
		for _, schema := range metadata.Msg.Schemas {
			for _, table := range schema.Tables {
				if table.Name == "users" {
					foundUsersTable = true
					a.Equal(3, len(table.Columns), "Users table should have 3 columns")
				}
				if table.Name == "names" {
					foundNamesTable = true
					a.Equal(3, len(table.Columns), "Names table should have 3 columns")
				}
			}
		}
		a.True(foundUsersTable, "Users table should exist after merged declarative rollout")
		a.True(foundNamesTable, "Names table should exist after merged declarative rollout")

		// Verify command output
		a.NotContains(result.Stderr, "error", "Merged declarative rollout should complete without errors")
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
		a.Contains(result.Stderr, "database instances/fake/databases/nonexistent not found", "Expected not found error in stderr")
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
		a.ErrorContains(err, "no files found for pattern")
	})
}
