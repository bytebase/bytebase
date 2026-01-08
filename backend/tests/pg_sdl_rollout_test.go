package tests

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// sdlTestContext holds shared resources for SDL tests.
type sdlTestContext struct {
	ctl         *controller
	pgContainer *Container
	ctx         context.Context
}

// sdlRolloutResult captures the result of an SDL rollout including executed SQL statements.
type sdlRolloutResult struct {
	// ExecutedStatements contains the SQL statements that were actually executed during the rollout.
	ExecutedStatements []string
}

// setupSDLTestContext creates shared resources for all SDL tests.
func setupSDLTestContext(t *testing.T) *sdlTestContext {
	t.Helper()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)

	pgContainer, err := getPgContainer(ctx)
	a.NoError(err)

	return &sdlTestContext{
		ctl:         ctl,
		pgContainer: pgContainer,
		ctx:         ctx,
	}
}

// cleanup releases shared resources.
func (stc *sdlTestContext) cleanup() {
	if stc.pgContainer != nil {
		stc.pgContainer.Close(stc.ctx)
	}
	if stc.ctl != nil {
		_ = stc.ctl.Close(stc.ctx)
	}
}

// createTestPgDatabase creates an independent PostgreSQL database for a test case.
func (stc *sdlTestContext) createTestPgDatabase(t *testing.T, dbNamePrefix string) *v1pb.Database {
	t.Helper()
	a := require.New(t)

	// Create unique database name
	dbName := fmt.Sprintf("%s_%s", dbNamePrefix, strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

	// Create database directly in PostgreSQL
	_, err := stc.pgContainer.db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	a.NoError(err)

	// Create instance in Bytebase
	instanceResp, err := stc.ctl.instanceServiceClient.CreateInstance(stc.ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst")[:8],
		Instance: &v1pb.Instance{
			Title:       fmt.Sprintf("SDL Test Instance %s", dbName),
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type:     v1pb.DataSourceType_ADMIN,
				Host:     stc.pgContainer.host,
				Port:     stc.pgContainer.port,
				Username: "postgres",
				Password: "root-password",
				Id:       "admin",
			}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	// Create database in Bytebase
	err = stc.ctl.createDatabase(stc.ctx, stc.ctl.project, instance, nil, dbName, "postgres")
	a.NoError(err)

	// Get database
	databaseResp, err := stc.ctl.databaseServiceClient.GetDatabase(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbName),
	}))
	a.NoError(err)

	return databaseResp.Msg
}

// executeSDLRollout performs a complete SDL rollout via gRPC APIs.
func (stc *sdlTestContext) executeSDLRollout(t *testing.T, database *v1pb.Database, sdlContent string) error {
	t.Helper()
	_, err := stc.executeSDLRolloutWithResult(t, database, sdlContent)
	return err
}

// executeSDLRolloutWithResult performs a complete SDL rollout and returns the result including executed SQL.
func (stc *sdlTestContext) executeSDLRolloutWithResult(t *testing.T, database *v1pb.Database, sdlContent string) (*sdlRolloutResult, error) {
	t.Helper()

	// Create a DECLARATIVE release with SDL content
	// Empty SDL content is allowed for DECLARATIVE releases (represents dropping all objects)
	releaseResp, err := stc.ctl.releaseServiceClient.CreateRelease(stc.ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent: stc.ctl.project.Name,
		Release: &v1pb.Release{
			Title: "SDL Rollout Release",
			Type:  v1pb.Release_DECLARATIVE,
			Files: []*v1pb.Release_File{
				{
					Path:      "schema.sql",
					Version:   fmt.Sprintf("%d", time.Now().Unix()),
					Statement: []byte(sdlContent),
				},
			},
		},
	}))
	if err != nil {
		return nil, err
	}
	release := releaseResp.Msg

	// Create plan with the release
	planResp, err := stc.ctl.planServiceClient.CreatePlan(stc.ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: stc.ctl.project.Name,
		Plan: &v1pb.Plan{
			Title: "SDL Rollout Plan",
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Release: release.Name,
							Targets: []string{database.Name},
						},
					},
				},
			},
		},
	}))
	if err != nil {
		return nil, err
	}
	plan := planResp.Msg

	// Create rollout (no issue needed)
	rolloutResp, err := stc.ctl.rolloutServiceClient.CreateRollout(stc.ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: plan.Name,
	}))
	if err != nil {
		return nil, err
	}
	rollout := rolloutResp.Msg

	// Wait for rollout to complete (without issue approval flow)
	err = stc.ctl.waitRolloutWithoutApproval(stc.ctx, rollout.Name)
	if err != nil {
		return nil, err
	}

	// Get executed SQL statements from task run logs
	executedStatements, err := stc.getExecutedStatements(rollout.Name)
	if err != nil {
		return nil, err
	}

	return &sdlRolloutResult{
		ExecutedStatements: executedStatements,
	}, nil
}

// getExecutedStatements retrieves the SQL statements that were actually executed during the rollout.
func (stc *sdlTestContext) getExecutedStatements(rolloutName string) ([]string, error) {
	var statements []string

	// Get the rollout to find all tasks
	rolloutResp, err := stc.ctl.rolloutServiceClient.GetRollout(stc.ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: rolloutName,
	}))
	if err != nil {
		return nil, err
	}

	// Iterate through all stages and tasks to get task run logs
	for _, stage := range rolloutResp.Msg.Stages {
		for _, task := range stage.Tasks {
			// Get task runs for this task
			taskRunsResp, err := stc.ctl.rolloutServiceClient.ListTaskRuns(stc.ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
				Parent: task.Name,
			}))
			if err != nil {
				return nil, err
			}

			// Get logs for each task run
			for _, taskRun := range taskRunsResp.Msg.TaskRuns {
				logResp, err := stc.ctl.rolloutServiceClient.GetTaskRunLog(stc.ctx, connect.NewRequest(&v1pb.GetTaskRunLogRequest{
					Parent: taskRun.Name,
				}))
				if err != nil {
					return nil, err
				}

				// Extract executed statements from log entries
				for _, entry := range logResp.Msg.Entries {
					if entry.Type == v1pb.TaskRunLogEntry_COMMAND_EXECUTE && entry.CommandExecute != nil {
						stmt := entry.CommandExecute.Statement
						if stmt != "" {
							statements = append(statements, stmt)
						}
					}
				}
			}
		}
	}

	return statements, nil
}

// verifyExecutedSQL verifies that the executed SQL statements exactly match the expected statements.
// Both the count and content of statements must match exactly (no more, no less).
func verifyExecutedSQL(t *testing.T, result *sdlRolloutResult, expectedStatements []string) {
	t.Helper()
	a := require.New(t)

	// Normalize statements for comparison (trim whitespace)
	normalize := func(stmts []string) []string {
		normalized := make([]string, len(stmts))
		for i, stmt := range stmts {
			normalized[i] = strings.TrimSpace(stmt)
		}
		return normalized
	}

	actual := normalize(result.ExecutedStatements)
	expected := normalize(expectedStatements)

	// First check: count must match
	a.Equal(len(expected), len(actual),
		"SQL statement count mismatch: expected %d statements, got %d.\nExpected statements:\n%s\nActual statements:\n%s",
		len(expected), len(actual),
		strings.Join(expected, "\n---\n"),
		strings.Join(actual, "\n---\n"))

	// Second check: each statement must match in order
	for i := range expected {
		a.Equal(expected[i], actual[i],
			"SQL statement mismatch at index %d.\nExpected:\n%s\nActual:\n%s",
			i, expected[i], actual[i])
	}
}

// verifyNoSQL verifies that no SQL statements were executed during the rollout.
func verifyNoSQL(t *testing.T, result *sdlRolloutResult) {
	t.Helper()
	a := require.New(t)
	a.Empty(result.ExecutedStatements,
		"Expected no SQL statements to be executed, but got %d statements:\n%s",
		len(result.ExecutedStatements),
		strings.Join(result.ExecutedStatements, "\n---\n"))
}

// verifyTableExists checks if a table exists in the database schema.
func (stc *sdlTestContext) verifyTableExists(t *testing.T, database *v1pb.Database, schemaName, tableName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == schemaName {
			for _, table := range schema.Tables {
				if table.Name == tableName {
					return true
				}
			}
		}
	}
	return false
}

// verifyTableNotExists checks if a table does not exist in the database schema.
func (stc *sdlTestContext) verifyTableNotExists(t *testing.T, database *v1pb.Database, tableName string) bool {
	return !stc.verifyTableExists(t, database, "public", tableName)
}

// verifyColumnExists checks if a column exists in a table.
//
//nolint:unparam
func (stc *sdlTestContext) verifyColumnExists(t *testing.T, database *v1pb.Database, tableName, columnName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
			for _, table := range schema.Tables {
				if table.Name == tableName {
					for _, col := range table.Columns {
						if col.Name == columnName {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// getTableColumnCount returns the number of columns in a table.
//
//nolint:unparam
func (stc *sdlTestContext) getTableColumnCount(t *testing.T, database *v1pb.Database, tableName string) int {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
			for _, table := range schema.Tables {
				if table.Name == tableName {
					return len(table.Columns)
				}
			}
		}
	}
	return 0
}

// verifyViewExists checks if a view exists in the database schema.
func (stc *sdlTestContext) verifyViewExists(t *testing.T, database *v1pb.Database, viewName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
			for _, view := range schema.Views {
				if view.Name == viewName {
					return true
				}
			}
		}
	}
	return false
}

// verifyMaterializedViewExists checks if a materialized view exists in the database schema.
func (stc *sdlTestContext) verifyMaterializedViewExists(t *testing.T, database *v1pb.Database, mvName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
			for _, mv := range schema.MaterializedViews {
				if mv.Name == mvName {
					return true
				}
			}
		}
	}
	return false
}

// verifyFunctionExists checks if a function exists in the database schema.
func (stc *sdlTestContext) verifyFunctionExists(t *testing.T, database *v1pb.Database, functionName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
			for _, fn := range schema.Functions {
				if fn.Name == functionName {
					return true
				}
			}
		}
	}
	return false
}

// verifySequenceExists checks if a sequence exists in the database schema.
func (stc *sdlTestContext) verifySequenceExists(t *testing.T, database *v1pb.Database, sequenceName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
			for _, seq := range schema.Sequences {
				if seq.Name == sequenceName {
					return true
				}
			}
		}
	}
	return false
}

// verifyForeignKeyExists checks if a foreign key constraint exists on a table.
//
//nolint:unparam
func (stc *sdlTestContext) verifyForeignKeyExists(t *testing.T, database *v1pb.Database, schemaName, tableName, constraintName string) bool {
	t.Helper()
	a := require.New(t)

	metadata, err := stc.ctl.databaseServiceClient.GetDatabaseMetadata(stc.ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	a.NoError(err)

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == schemaName {
			for _, table := range schema.Tables {
				if table.Name == tableName {
					for _, fk := range table.ForeignKeys {
						if fk.Name == constraintName {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// directExecuteSQL executes SQL directly on the database (for drift tests).
func (stc *sdlTestContext) directExecuteSQL(t *testing.T, database *v1pb.Database, sqlStmt string) {
	t.Helper()
	a := require.New(t)

	// Extract database name from resource name
	parts := strings.Split(database.Name, "/")
	dbName := parts[len(parts)-1]

	// Connect to the specific database
	connStr := fmt.Sprintf("host=%s port=%s user=postgres password=root-password dbname=%s sslmode=disable",
		stc.pgContainer.host, stc.pgContainer.port, dbName)
	db, err := sql.Open("pgx", connStr)
	a.NoError(err)
	defer db.Close()

	_, err = db.Exec(sqlStmt)
	a.NoError(err)
}

// syncDatabase syncs database metadata.
func (stc *sdlTestContext) syncDatabase(t *testing.T, database *v1pb.Database) {
	t.Helper()
	a := require.New(t)

	_, err := stc.ctl.databaseServiceClient.SyncDatabase(stc.ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{
		Name: database.Name,
	}))
	a.NoError(err)
}

// TestPgSDLRollout tests PostgreSQL SDL rollout flow end-to-end.
func TestPgSDLRollout(t *testing.T) {
	t.Parallel()

	t.Run("BasicObjects", func(t *testing.T) {
		t.Parallel()

		// ==================== Table ====================
		t.Run("Table", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tbl_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "created_at" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				result, err := stc.executeSDLRolloutWithResult(t, database, sdl)
				a.NoError(err)

				// Verify executed SQL - SDL engine uses the SDL as-is for creation
				verifyExecutedSQL(t, result, []string{sdl})

				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.Equal(4, stc.getTableColumnCount(t, database, "users"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tbl_alter")

				// First create the table
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`
				result1, err := stc.executeSDLRolloutWithResult(t, database, sdl1)
				a.NoError(err)

				// Verify create SQL - SDL engine uses the SDL as-is
				verifyExecutedSQL(t, result1, []string{sdl1})
				a.Equal(2, stc.getTableColumnCount(t, database, "users"))

				// Then alter: add columns
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "created_at" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`
				result2, err := stc.executeSDLRolloutWithResult(t, database, sdl2)
				a.NoError(err)

				// Verify alter SQL - diff engine generates ALTER statements
				// The exact format is determined by the diff engine
				a.NotEmpty(result2.ExecutedStatements, "Expected ALTER statements to be executed")

				a.Equal(4, stc.getTableColumnCount(t, database, "users"))
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
				a.True(stc.verifyColumnExists(t, database, "users", "created_at"))
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tbl_drop")

				// First create the table
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`
				_, err := stc.executeSDLRolloutWithResult(t, database, sdl1)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))

				// Then drop it (empty SDL)
				sdl2 := ``
				result2, err := stc.executeSDLRolloutWithResult(t, database, sdl2)
				a.NoError(err)

				// Verify drop SQL - should have DROP statement
				a.NotEmpty(result2.ExecutedStatements, "Expected DROP statement to be executed")

				a.True(stc.verifyTableNotExists(t, database, "users"))
			})
		})

		// ==================== Index ====================
		t.Run("Index", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "idx_create")

				createTableSQL := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`
				createIndexSQL := `CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				sdl := createTableSQL + "\n\n" + createIndexSQL

				result, err := stc.executeSDLRolloutWithResult(t, database, sdl)
				a.NoError(err)

				// Verify executed SQL - SDL engine splits multiple statements
				verifyExecutedSQL(t, result, []string{createTableSQL, createIndexSQL})

				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "idx_alter")

				// Create table with index
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				_, err := stc.executeSDLRolloutWithResult(t, database, sdl1)
				a.NoError(err)

				// Change index to composite (drop old, create new)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email_name" ON "public"."users" ("email", "name");`

				result2, err := stc.executeSDLRolloutWithResult(t, database, sdl2)
				a.NoError(err)

				// Verify alter SQL - diff engine generates DROP and CREATE statements
				a.NotEmpty(result2.ExecutedStatements, "Expected index change statements to be executed")
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "idx_drop")

				// Create table with index
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				_, err := stc.executeSDLRolloutWithResult(t, database, sdl1)
				a.NoError(err)

				// Drop index (keep table)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				result2, err := stc.executeSDLRolloutWithResult(t, database, sdl2)
				a.NoError(err)

				// Verify executed SQL - should have DROP INDEX statement
				a.NotEmpty(result2.ExecutedStatements, "Expected DROP INDEX statement to be executed")
			})
		})

		// ==================== Constraint ====================
		t.Run("Constraint", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cst_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    "age" integer,
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email"),
    CONSTRAINT "ck_users_age" CHECK (age >= 0)
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cst_alter")

				// Create with basic constraint
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`
				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Add unique constraint
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);`
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cst_drop")

				// Create with constraints
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);`
				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop unique constraint (keep PK)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// ==================== View ====================
		t.Run("View", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "view_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "active" boolean DEFAULT true,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."active_users" AS
SELECT id, name FROM "public"."users" WHERE active = true;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyViewExists(t, database, "active_users"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "view_alter")

				// Create initial view
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "active" boolean DEFAULT true,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."active_users" AS
SELECT id, name FROM "public"."users" WHERE active = true;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify view to include email
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "active" boolean DEFAULT true,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."active_users" AS
SELECT id, name, email FROM "public"."users" WHERE active = true;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "view_drop")

				// Create view
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.True(stc.verifyViewExists(t, database, "user_names"))

				// Drop view (keep table)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifyViewExists(t, database, "user_names"))
			})
		})

		// ==================== MaterializedView ====================
		t.Run("MaterializedView", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mview_create")

				sdl := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_summary" AS
SELECT user_id, SUM(amount) as total_amount FROM "public"."orders" GROUP BY user_id;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mview_alter")

				// Create initial mview
				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_summary" AS
SELECT user_id, SUM(amount) as total_amount FROM "public"."orders" GROUP BY user_id;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify mview to include count
				sdl2 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_summary" AS
SELECT user_id, SUM(amount) as total_amount, COUNT(*) as order_count FROM "public"."orders" GROUP BY user_id;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mview_drop")

				// Create mview
				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_summary" AS
SELECT user_id, SUM(amount) as total_amount FROM "public"."orders" GROUP BY user_id;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop mview (keep table)
				sdl2 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// ==================== Function ====================
		t.Run("Function", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "func_create")

				sdl := `CREATE FUNCTION "public"."add_numbers"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyFunctionExists(t, database, "add_numbers"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "func_alter")

				// Create initial function
				sdl1 := `CREATE FUNCTION "public"."add_numbers"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify function body
				sdl2 := `CREATE FUNCTION "public"."add_numbers"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b + 1;
END;
$$;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "func_drop")

				// Create function
				sdl1 := `CREATE FUNCTION "public"."add_numbers"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.True(stc.verifyFunctionExists(t, database, "add_numbers"))

				// Drop function (empty SDL)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifyFunctionExists(t, database, "add_numbers"))
			})
		})

		// ==================== Procedure ====================
		t.Run("Procedure", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "proc_create")

				sdl := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "message" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_message"(msg text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("message") VALUES (msg);
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "logs"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "proc_alter")

				// Create initial procedure
				sdl1 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "message" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_message"(msg text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("message") VALUES (msg);
END;
$$;`

				result1, err := stc.executeSDLRolloutWithResult(t, database, sdl1)
				a.NoError(err)
				t.Logf("First rollout executed %d SQL statements:", len(result1.ExecutedStatements))
				for i, stmt := range result1.ExecutedStatements {
					t.Logf("  [%d]: %s", i, stmt)
				}

				// Modify procedure
				sdl2 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "message" text,
    "created_at" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_message"(msg text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("message", "created_at") VALUES (msg, CURRENT_TIMESTAMP);
END;
$$;`

				result2, err := stc.executeSDLRolloutWithResult(t, database, sdl2)
				if err != nil {
					t.Logf("Second rollout failed with error: %v", err)
					if result2 != nil {
						t.Logf("It attempted to execute %d SQL statements:", len(result2.ExecutedStatements))
						for i, stmt := range result2.ExecutedStatements {
							t.Logf("  [%d]: %s", i, stmt)
						}
					}
				}
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "proc_drop")

				// Create procedure
				sdl1 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "message" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_message"(msg text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("message") VALUES (msg);
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop procedure (keep table)
				sdl2 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "message" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// ==================== Trigger ====================
		t.Run("Trigger", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trg_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."update_timestamp"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_update"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."update_timestamp"();`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyFunctionExists(t, database, "update_timestamp"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trg_alter")

				// Create initial trigger
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."update_timestamp"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_update"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."update_timestamp"();`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Change trigger to AFTER UPDATE
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."update_timestamp"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_update"
AFTER UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."update_timestamp"();`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trg_drop")

				// Create trigger
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."update_timestamp"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_update"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."update_timestamp"();`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop trigger (keep table and function)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."update_timestamp"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// ==================== Sequence ====================
		t.Run("Sequence", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "seq_create")

				sdl := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1000
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifySequenceExists(t, database, "order_seq"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "seq_alter")

				// Create initial sequence
				sdl1 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify sequence
				sdl2 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1000
    INCREMENT BY 10
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "seq_drop")

				// Create sequence
				sdl1 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.True(stc.verifySequenceExists(t, database, "order_seq"))

				// Drop sequence (empty SDL)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifySequenceExists(t, database, "order_seq"))
			})
		})

		// ==================== Enum ====================
		t.Run("Enum", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "enum_create")

				sdl := `CREATE TYPE "public"."status" AS ENUM ('pending', 'active', 'completed');`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "enum_alter")

				// Create initial enum
				sdl1 := `CREATE TYPE "public"."status" AS ENUM ('pending', 'active');`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Add enum value
				sdl2 := `CREATE TYPE "public"."status" AS ENUM ('pending', 'active', 'completed');`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "enum_drop")

				// Create enum
				sdl1 := `CREATE TYPE "public"."status" AS ENUM ('pending', 'active', 'completed');`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop enum (empty SDL)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// ==================== Extension ====================
		t.Run("Extension", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ext_create")

				sdl := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ext_alter")

				// Create initial extension
				sdl1 := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Add another extension
				sdl2 := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ext_drop")

				// Create extension
				sdl1 := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop extension (empty SDL)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// ==================== Comment ====================
		t.Run("Comment", func(t *testing.T) {
			t.Parallel()

			t.Run("Create", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cmt_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'User information table';
COMMENT ON COLUMN "public"."users"."name" IS 'User display name';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("Alter", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cmt_alter")

				// Create with comment
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'User table';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Change comment
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'User information table - updated';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("Drop", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cmt_drop")

				// Create with comment
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'User table';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop comment (keep table)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})
	})

	// ==================== Dependencies ====================
	t.Run("Dependencies", func(t *testing.T) {
		t.Parallel()

		// ==================== Object Dependencies ====================

		// -------------------- ForeignKey --------------------
		t.Run("ForeignKey", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateTablesWithFK", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_create")

				// Create two tables where orders references users
				// Intentionally put orders BEFORE users to test dependency ordering
				sdl := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id"),
    CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id")
);

CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
			})

			t.Run("DropTableWithFK", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_drop")

				// Create tables with FK
				// Intentionally put orders BEFORE users to test dependency ordering
				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id"),
    CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id")
);

CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop both tables (verify order: orders → users)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
				a.True(stc.verifyTableNotExists(t, database, "orders"))
			})

			t.Run("AlterFKReference", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_alter")

				// Create tables with FK
				// Intentionally put orders BEFORE users to test dependency ordering
				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id"),
    CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id")
);

CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify FK to add ON DELETE CASCADE
				// Intentionally put orders BEFORE users to test dependency ordering
				sdl2 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id"),
    CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON DELETE CASCADE
);

CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("SelfReferencingFK", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_self_ref")

				// Create a table with self-referencing FK (like categories with parent_id)
				sdl := `CREATE TABLE "public"."categories" (
    "id" bigserial,
    "name" text NOT NULL,
    "parent_id" bigint,
    CONSTRAINT "categories_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_categories_parent" FOREIGN KEY ("parent_id") REFERENCES "public"."categories" ("id") ON DELETE SET NULL
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "categories"))
				// Verify the self-referencing FK constraint was created
				a.True(stc.verifyForeignKeyExists(t, database, "public", "categories", "fk_categories_parent"))
			})

			t.Run("BidirectionalFK", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_bidirectional")

				// Create two tables with bidirectional FK references (like projects <-> namespaces)
				// This creates a cycle: projects -> namespaces -> projects
				sdl := `CREATE TABLE "public"."namespaces" (
    "id" bigserial,
    "name" text NOT NULL,
    "file_template_project_id" bigint,
    CONSTRAINT "namespaces_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_namespaces_project" FOREIGN KEY ("file_template_project_id") REFERENCES "public"."projects" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."projects" (
    "id" bigserial,
    "name" text NOT NULL,
    "namespace_id" bigint NOT NULL,
    CONSTRAINT "projects_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_projects_namespace" FOREIGN KEY ("namespace_id") REFERENCES "public"."namespaces" ("id") ON DELETE CASCADE
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "namespaces"))
				a.True(stc.verifyTableExists(t, database, "public", "projects"))
				// Verify both FK constraints in the bidirectional cycle were created
				a.True(stc.verifyForeignKeyExists(t, database, "public", "namespaces", "fk_namespaces_project"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "projects", "fk_projects_namespace"))
			})

			t.Run("ThreeWayCyclicFK", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_three_way")

				// Create three tables with cyclic FK: A -> B -> C -> A
				sdl := `CREATE TABLE "public"."table_c" (
    "id" bigserial,
    "name" text NOT NULL,
    "ref_a_id" bigint,
    CONSTRAINT "table_c_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_c_to_a" FOREIGN KEY ("ref_a_id") REFERENCES "public"."table_a" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."table_a" (
    "id" bigserial,
    "name" text NOT NULL,
    "ref_b_id" bigint,
    CONSTRAINT "table_a_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_a_to_b" FOREIGN KEY ("ref_b_id") REFERENCES "public"."table_b" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."table_b" (
    "id" bigserial,
    "name" text NOT NULL,
    "ref_c_id" bigint,
    CONSTRAINT "table_b_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_b_to_c" FOREIGN KEY ("ref_c_id") REFERENCES "public"."table_c" ("id") ON DELETE SET NULL
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "table_a"))
				a.True(stc.verifyTableExists(t, database, "public", "table_b"))
				a.True(stc.verifyTableExists(t, database, "public", "table_c"))
				// Verify all three FK constraints in the cycle were created: A -> B -> C -> A
				a.True(stc.verifyForeignKeyExists(t, database, "public", "table_a", "fk_a_to_b"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "table_b", "fk_b_to_c"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "table_c", "fk_c_to_a"))
			})

			t.Run("ComplexCyclicWithMultipleFK", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_complex_cycle")

				// Complex scenario: multiple FK constraints including cycles and self-references
				// Similar to GitLab schema: projects, namespaces, issues, merge_requests
				sdl := `CREATE TABLE "public"."issues" (
    "id" bigserial,
    "title" text NOT NULL,
    "project_id" bigint,
    "moved_to_id" bigint,
    CONSTRAINT "issues_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_issues_project" FOREIGN KEY ("project_id") REFERENCES "public"."projects" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_issues_moved_to" FOREIGN KEY ("moved_to_id") REFERENCES "public"."issues" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."merge_requests" (
    "id" bigserial,
    "title" text NOT NULL,
    "project_id" bigint,
    "latest_merge_request_diff_id" bigint,
    CONSTRAINT "merge_requests_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_mr_project" FOREIGN KEY ("project_id") REFERENCES "public"."projects" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_mr_latest_diff" FOREIGN KEY ("latest_merge_request_diff_id") REFERENCES "public"."merge_request_diffs" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."merge_request_diffs" (
    "id" bigserial,
    "merge_request_id" bigint NOT NULL,
    CONSTRAINT "merge_request_diffs_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_diff_mr" FOREIGN KEY ("merge_request_id") REFERENCES "public"."merge_requests" ("id") ON DELETE CASCADE
);

CREATE TABLE "public"."projects" (
    "id" bigserial,
    "name" text NOT NULL,
    "namespace_id" bigint NOT NULL,
    CONSTRAINT "projects_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_projects_namespace" FOREIGN KEY ("namespace_id") REFERENCES "public"."namespaces" ("id") ON DELETE CASCADE
);

CREATE TABLE "public"."namespaces" (
    "id" bigserial,
    "name" text NOT NULL,
    "file_template_project_id" bigint,
    "parent_id" bigint,
    CONSTRAINT "namespaces_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_namespaces_project" FOREIGN KEY ("file_template_project_id") REFERENCES "public"."projects" ("id") ON DELETE SET NULL,
    CONSTRAINT "fk_namespaces_parent" FOREIGN KEY ("parent_id") REFERENCES "public"."namespaces" ("id") ON DELETE CASCADE
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "issues"))
				a.True(stc.verifyTableExists(t, database, "public", "merge_requests"))
				a.True(stc.verifyTableExists(t, database, "public", "merge_request_diffs"))
				a.True(stc.verifyTableExists(t, database, "public", "projects"))
				a.True(stc.verifyTableExists(t, database, "public", "namespaces"))
				// Verify all FK constraints were created including cycles and self-references
				a.True(stc.verifyForeignKeyExists(t, database, "public", "issues", "fk_issues_project"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "issues", "fk_issues_moved_to"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "merge_requests", "fk_mr_project"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "merge_requests", "fk_mr_latest_diff"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "merge_request_diffs", "fk_diff_mr"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "projects", "fk_projects_namespace"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "namespaces", "fk_namespaces_project"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "namespaces", "fk_namespaces_parent"))
			})

			t.Run("CyclicFKWithDependentViews", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_cycle_views")

				// Create tables with cyclic FK and views that depend on each other
				// The views are intentionally ordered wrong (v2 before v1)
				// to test that topological sorting works correctly even when table FK cycle exists
				sdl := `CREATE TABLE "public"."namespaces" (
    "id" bigserial,
    "name" text NOT NULL,
    "project_id" bigint,
    CONSTRAINT "namespaces_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_namespaces_project" FOREIGN KEY ("project_id") REFERENCES "public"."projects" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."projects" (
    "id" bigserial,
    "name" text NOT NULL,
    "namespace_id" bigint NOT NULL,
    CONSTRAINT "projects_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_projects_namespace" FOREIGN KEY ("namespace_id") REFERENCES "public"."namespaces" ("id") ON DELETE CASCADE
);

CREATE VIEW "public"."v2_project_summary" AS
SELECT p.id, p.name, n.name as namespace_name
FROM "public"."v1_active_projects" p
JOIN "public"."namespaces" n ON p.namespace_id = n.id;

CREATE VIEW "public"."v1_active_projects" AS
SELECT id, name, namespace_id FROM "public"."projects" WHERE name IS NOT NULL;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "namespaces"))
				a.True(stc.verifyTableExists(t, database, "public", "projects"))
				a.True(stc.verifyViewExists(t, database, "v1_active_projects"))
				a.True(stc.verifyViewExists(t, database, "v2_project_summary"))
				// Verify FK constraints were created
				a.True(stc.verifyForeignKeyExists(t, database, "public", "namespaces", "fk_namespaces_project"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "projects", "fk_projects_namespace"))
			})

			t.Run("CyclicFKWithMaterializedViewChain", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_cycle_mv")

				// Create tables with cyclic FK and materialized views with dependencies
				// mv2 depends on mv1, and both depend on tables with FK cycle
				sdl := `CREATE TABLE "public"."orders" (
    "id" bigserial,
    "customer_id" bigint,
    "total" numeric(10, 2),
    CONSTRAINT "orders_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_orders_customer" FOREIGN KEY ("customer_id") REFERENCES "public"."customers" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."customers" (
    "id" bigserial,
    "name" text NOT NULL,
    "last_order_id" bigint,
    CONSTRAINT "customers_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_customers_last_order" FOREIGN KEY ("last_order_id") REFERENCES "public"."orders" ("id") ON DELETE SET NULL
);

CREATE MATERIALIZED VIEW "public"."mv2_customer_order_summary" AS
SELECT c.id, c.name, o.order_count, o.total_amount
FROM "public"."mv1_order_stats" o
JOIN "public"."customers" c ON o.customer_id = c.id;

CREATE MATERIALIZED VIEW "public"."mv1_order_stats" AS
SELECT customer_id, COUNT(*) as order_count, SUM(total) as total_amount
FROM "public"."orders"
GROUP BY customer_id;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
				a.True(stc.verifyTableExists(t, database, "public", "customers"))
				a.True(stc.verifyMaterializedViewExists(t, database, "mv1_order_stats"))
				a.True(stc.verifyMaterializedViewExists(t, database, "mv2_customer_order_summary"))
				// Verify FK constraints were created
				a.True(stc.verifyForeignKeyExists(t, database, "public", "orders", "fk_orders_customer"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "customers", "fk_customers_last_order"))
			})

			t.Run("CyclicFKWithLongViewChain", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_cycle_view_chain")

				// Create tables with cyclic FK and a long chain of views: v4 -> v3 -> v2 -> v1
				// Views are intentionally defined in reverse order to test topological sorting
				sdl := `CREATE TABLE "public"."departments" (
    "id" bigserial,
    "name" text NOT NULL,
    "manager_id" bigint,
    CONSTRAINT "departments_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_dept_manager" FOREIGN KEY ("manager_id") REFERENCES "public"."employees" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."employees" (
    "id" bigserial,
    "name" text NOT NULL,
    "department_id" bigint,
    "salary" numeric(10, 2),
    CONSTRAINT "employees_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_emp_dept" FOREIGN KEY ("department_id") REFERENCES "public"."departments" ("id") ON DELETE SET NULL
);

CREATE VIEW "public"."v4_top_departments" AS
SELECT department_name, avg_salary FROM "public"."v3_dept_summary" WHERE avg_salary > 50000;

CREATE VIEW "public"."v3_dept_summary" AS
SELECT department_name, AVG(salary) as avg_salary FROM "public"."v2_employee_details" GROUP BY department_name;

CREATE VIEW "public"."v2_employee_details" AS
SELECT e.id, e.name, e.salary, d.name as department_name
FROM "public"."v1_active_employees" e
JOIN "public"."departments" d ON e.department_id = d.id;

CREATE VIEW "public"."v1_active_employees" AS
SELECT id, name, department_id, salary FROM "public"."employees" WHERE salary > 0;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "departments"))
				a.True(stc.verifyTableExists(t, database, "public", "employees"))
				a.True(stc.verifyViewExists(t, database, "v1_active_employees"))
				a.True(stc.verifyViewExists(t, database, "v2_employee_details"))
				a.True(stc.verifyViewExists(t, database, "v3_dept_summary"))
				a.True(stc.verifyViewExists(t, database, "v4_top_departments"))
				// Verify FK constraints were created
				a.True(stc.verifyForeignKeyExists(t, database, "public", "departments", "fk_dept_manager"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "employees", "fk_emp_dept"))
			})

			t.Run("CyclicFKWithMixedViewsAndMaterializedViews", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fk_cycle_mixed")

				// Create tables with cyclic FK and mixed views/materialized views with dependencies
				// mv depends on view, view depends on another view
				sdl := `CREATE TABLE "public"."products" (
    "id" bigserial,
    "name" text NOT NULL,
    "category_id" bigint,
    "price" numeric(10, 2),
    CONSTRAINT "products_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_prod_cat" FOREIGN KEY ("category_id") REFERENCES "public"."categories" ("id") ON DELETE SET NULL
);

CREATE TABLE "public"."categories" (
    "id" bigserial,
    "name" text NOT NULL,
    "featured_product_id" bigint,
    CONSTRAINT "categories_pkey" PRIMARY KEY (id),
    CONSTRAINT "fk_cat_featured" FOREIGN KEY ("featured_product_id") REFERENCES "public"."products" ("id") ON DELETE SET NULL
);

CREATE MATERIALIZED VIEW "public"."mv_category_stats" AS
SELECT category_name, product_count, avg_price
FROM "public"."v2_category_products";

CREATE VIEW "public"."v2_category_products" AS
SELECT c.name as category_name, COUNT(*) as product_count, AVG(p.price) as avg_price
FROM "public"."v1_available_products" p
JOIN "public"."categories" c ON p.category_id = c.id
GROUP BY c.name;

CREATE VIEW "public"."v1_available_products" AS
SELECT id, name, category_id, price FROM "public"."products" WHERE price > 0;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "products"))
				a.True(stc.verifyTableExists(t, database, "public", "categories"))
				a.True(stc.verifyViewExists(t, database, "v1_available_products"))
				a.True(stc.verifyViewExists(t, database, "v2_category_products"))
				a.True(stc.verifyMaterializedViewExists(t, database, "mv_category_stats"))
				// Verify FK constraints were created
				a.True(stc.verifyForeignKeyExists(t, database, "public", "products", "fk_prod_cat"))
				a.True(stc.verifyForeignKeyExists(t, database, "public", "categories", "fk_cat_featured"))
			})
		})

		// -------------------- ViewTable --------------------
		t.Run("ViewTable", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateViewDependsOnTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vt_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "active" boolean DEFAULT true,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."active_users" AS
SELECT id, name FROM "public"."users" WHERE active = true;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyViewExists(t, database, "active_users"))
			})

			t.Run("DropTableWithDependentView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vt_drop")

				// Create table and dependent view
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop both (view must be dropped first)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
				a.False(stc.verifyViewExists(t, database, "user_names"))
			})

			t.Run("AlterTableUsedByView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vt_alter")

				// Create table and view
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Add column to table (view should still work)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
			})
		})

		// -------------------- ViewChain --------------------
		t.Run("ViewChain", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateViewChain", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vc_create")

				// Create view chain: View C → View B → Table A
				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "active" boolean DEFAULT true,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."active_users" AS
SELECT id, name, email FROM "public"."users" WHERE active = true;

CREATE VIEW "public"."active_user_emails" AS
SELECT id, email FROM "public"."active_users";`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyViewExists(t, database, "active_users"))
				a.True(stc.verifyViewExists(t, database, "active_user_emails"))
			})

			t.Run("DropViewChain", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vc_drop")

				// Create view chain
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_view1" AS
SELECT id, name FROM "public"."users";

CREATE VIEW "public"."user_view2" AS
SELECT id FROM "public"."user_view1";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop all (verify correct order)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
			})
		})

		// -------------------- MaterializedView --------------------
		t.Run("MaterializedView", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateMViewDependsOnTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mv_table")

				sdl := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT user_id, SUM(amount) as total FROM "public"."orders" GROUP BY user_id;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
			})

			t.Run("CreateMViewDependsOnView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mv_view")

				sdl := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "amount" decimal(10,2) NOT NULL,
    "status" varchar(20) DEFAULT 'pending',
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE VIEW "public"."completed_orders" AS
SELECT id, user_id, amount FROM "public"."orders" WHERE status = 'completed';

CREATE MATERIALIZED VIEW "public"."completed_totals" AS
SELECT user_id, SUM(amount) as total FROM "public"."completed_orders" GROUP BY user_id;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
				a.True(stc.verifyViewExists(t, database, "completed_orders"))
			})
		})

		// -------------------- TriggerFunction --------------------
		t.Run("TriggerFunction", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateTriggerWithFunction", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tf_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_updated_at"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_updated"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_updated_at"();`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyFunctionExists(t, database, "set_updated_at"))
			})

			t.Run("DropFunctionUsedByTrigger", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tf_drop")

				// Create trigger with function
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_updated_at"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_updated"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_updated_at"();`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop function and trigger (keep table)
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifyFunctionExists(t, database, "set_updated_at"))
			})

			t.Run("AlterTriggerFunction", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tf_alter")

				// Create trigger with function
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_updated_at"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_updated"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_updated_at"();`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify function body
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_updated_at"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_updated"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_updated_at"();`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- SequenceColumn --------------------
		t.Run("SequenceColumn", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateSequenceOwnedByColumn", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sc_create")

				sdl := `CREATE SEQUENCE "public"."order_num_seq"
    START WITH 1000
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "order_num" integer DEFAULT nextval('public.order_num_seq'),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

ALTER SEQUENCE "public"."order_num_seq" OWNED BY "public"."orders"."order_num";`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
				a.True(stc.verifySequenceExists(t, database, "order_num_seq"))
			})

			t.Run("DropColumnWithOwnedSequence", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sc_drop")

				// Create sequence owned by column
				sdl1 := `CREATE SEQUENCE "public"."order_num_seq"
    START WITH 1000
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "order_num" integer DEFAULT nextval('public.order_num_seq'),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

ALTER SEQUENCE "public"."order_num_seq" OWNED BY "public"."orders"."order_num";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop table (sequence should be dropped too due to OWNED BY)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "orders"))
			})
		})

		// -------------------- IndexTable --------------------
		t.Run("IndexTable", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateIndexOnTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "it_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");
CREATE INDEX "idx_users_name" ON "public"."users" ("name");`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("DropTableWithIndexes", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "it_drop")

				// Create table with indexes
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop table (indexes should be dropped automatically)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
			})
		})

		// -------------------- EnumTable --------------------
		t.Run("EnumTable", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateTableUsingEnum", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "et_create")

				sdl := `CREATE TYPE "public"."order_status" AS ENUM ('pending', 'processing', 'completed', 'cancelled');

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "status" "public"."order_status" DEFAULT 'pending',
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
			})

			t.Run("DropEnumUsedByTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "et_drop")

				// Create enum and table using it
				sdl1 := `CREATE TYPE "public"."order_status" AS ENUM ('pending', 'completed');

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "status" "public"."order_status" DEFAULT 'pending',
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop both (table must be dropped/altered before enum)
				sdl2 := ``

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "orders"))
			})
		})

		// ==================== Comment Combinations ====================

		// -------------------- TableComment --------------------
		t.Run("TableComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateTableWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tc_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'Store user information';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("DropTableWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tc_drop")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'User table';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
			})

			t.Run("AddCommentToExistingTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tc_add")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "tc_dropcmt")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'Table comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})
		})

		// -------------------- ColumnComment --------------------
		t.Run("ColumnComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateColumnWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cc_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON COLUMN "public"."users"."email" IS 'User email address';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
			})

			t.Run("DropColumnWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cc_drop")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON COLUMN "public"."users"."email" IS 'Email column';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifyColumnExists(t, database, "users", "email"))
			})

			t.Run("AddCommentToExistingColumn", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cc_add")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON COLUMN "public"."users"."email" IS 'Email address';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromColumn", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cc_dropcmt")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON COLUMN "public"."users"."email" IS 'Email';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- IndexComment --------------------
		t.Run("IndexComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateIndexWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ic_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");

COMMENT ON INDEX "public"."idx_users_email" IS 'Index for email lookup';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropIndexWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ic_drop")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");

COMMENT ON INDEX "public"."idx_users_email" IS 'Email index';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingIndex", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ic_add")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");

COMMENT ON INDEX "public"."idx_users_email" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromIndex", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ic_dropcmt")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");

COMMENT ON INDEX "public"."idx_users_email" IS 'Index comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- ConstraintComment --------------------
		t.Run("ConstraintComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateConstraintWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "csc_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);

COMMENT ON CONSTRAINT "uk_users_email" ON "public"."users" IS 'Unique email constraint';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropConstraintWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "csc_drop")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);

COMMENT ON CONSTRAINT "uk_users_email" ON "public"."users" IS 'Unique constraint';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingConstraint", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "csc_add")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);

COMMENT ON CONSTRAINT "uk_users_email" ON "public"."users" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromConstraint", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "csc_dropcmt")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);

COMMENT ON CONSTRAINT "uk_users_email" ON "public"."users" IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- ViewComment --------------------
		t.Run("ViewComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateViewWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vc_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";

COMMENT ON VIEW "public"."user_names" IS 'View for user names';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyViewExists(t, database, "user_names"))
			})

			t.Run("DropViewWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vc_drop")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";

COMMENT ON VIEW "public"."user_names" IS 'View comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifyViewExists(t, database, "user_names"))
			})

			t.Run("AddCommentToExistingView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vc_add")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";

COMMENT ON VIEW "public"."user_names" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vc_dropcmt")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";

COMMENT ON VIEW "public"."user_names" IS 'View comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- MaterializedViewComment --------------------
		t.Run("MaterializedViewComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateMViewWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mvc_create")

				sdl := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT SUM(amount) as total FROM "public"."orders";

COMMENT ON MATERIALIZED VIEW "public"."order_totals" IS 'Order totals mview';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropMViewWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mvc_drop")

				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT SUM(amount) as total FROM "public"."orders";

COMMENT ON MATERIALIZED VIEW "public"."order_totals" IS 'MView comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingMView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mvc_add")

				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT SUM(amount) as total FROM "public"."orders";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT SUM(amount) as total FROM "public"."orders";

COMMENT ON MATERIALIZED VIEW "public"."order_totals" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromMView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mvc_dropcmt")

				sdl1 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT SUM(amount) as total FROM "public"."orders";

COMMENT ON MATERIALIZED VIEW "public"."order_totals" IS 'MView comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "amount" decimal(10,2),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE MATERIALIZED VIEW "public"."order_totals" AS
SELECT SUM(amount) as total FROM "public"."orders";`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- FunctionComment --------------------
		t.Run("FunctionComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateFunctionWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fc_create")

				sdl := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_nums"(integer, integer) IS 'Add two numbers';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyFunctionExists(t, database, "add_nums"))
			})

			t.Run("DropFunctionWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fc_drop")

				sdl1 := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_nums"(integer, integer) IS 'Function comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifyFunctionExists(t, database, "add_nums"))
			})

			t.Run("AddCommentToExistingFunction", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fc_add")

				sdl1 := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_nums"(integer, integer) IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromFunction", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fc_dropcmt")

				sdl1 := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_nums"(integer, integer) IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- ProcedureComment --------------------
		t.Run("ProcedureComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateProcedureWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "pc_create")

				sdl := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_msg"(m text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("msg") VALUES (m);
END;
$$;

COMMENT ON PROCEDURE "public"."log_msg"(text) IS 'Log a message';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropProcedureWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "pc_drop")

				sdl1 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_msg"(m text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("msg") VALUES (m);
END;
$$;

COMMENT ON PROCEDURE "public"."log_msg"(text) IS 'Procedure comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingProcedure", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "pc_add")

				sdl1 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_msg"(m text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("msg") VALUES (m);
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_msg"(m text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("msg") VALUES (m);
END;
$$;

COMMENT ON PROCEDURE "public"."log_msg"(text) IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromProcedure", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "pc_dropcmt")

				sdl1 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_msg"(m text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("msg") VALUES (m);
END;
$$;

COMMENT ON PROCEDURE "public"."log_msg"(text) IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."logs" (
    "id" serial NOT NULL,
    "msg" text,
    CONSTRAINT "pk_logs" PRIMARY KEY ("id")
);

CREATE PROCEDURE "public"."log_msg"(m text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO "public"."logs" ("msg") VALUES (m);
END;
$$;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- TriggerComment --------------------
		t.Run("TriggerComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateTriggerWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trc_create")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();

COMMENT ON TRIGGER "trg_users_ts" ON "public"."users" IS 'Update timestamp trigger';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropTriggerWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trc_drop")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();

COMMENT ON TRIGGER "trg_users_ts" ON "public"."users" IS 'Trigger comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingTrigger", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trc_add")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();

COMMENT ON TRIGGER "trg_users_ts" ON "public"."users" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromTrigger", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trc_dropcmt")

				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();

COMMENT ON TRIGGER "trg_users_ts" ON "public"."users" IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- SequenceComment --------------------
		t.Run("SequenceComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateSequenceWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sqc_create")

				sdl := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE "public"."order_seq" IS 'Order number sequence';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifySequenceExists(t, database, "order_seq"))
			})

			t.Run("DropSequenceWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sqc_drop")

				sdl1 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE "public"."order_seq" IS 'Sequence comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.False(stc.verifySequenceExists(t, database, "order_seq"))
			})

			t.Run("AddCommentToExistingSequence", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sqc_add")

				sdl1 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE "public"."order_seq" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromSequence", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sqc_dropcmt")

				sdl1 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE "public"."order_seq" IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- EnumComment --------------------
		t.Run("EnumComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateEnumWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_create")

				sdl := `CREATE TYPE "public"."status" AS ENUM ('active', 'inactive');

COMMENT ON TYPE "public"."status" IS 'Status enum type';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropEnumWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_drop")

				sdl1 := `CREATE TYPE "public"."status" AS ENUM ('active', 'inactive');

COMMENT ON TYPE "public"."status" IS 'Enum comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingEnum", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_add")

				sdl1 := `CREATE TYPE "public"."status" AS ENUM ('active', 'inactive');`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TYPE "public"."status" AS ENUM ('active', 'inactive');

COMMENT ON TYPE "public"."status" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromEnum", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_dropcmt")

				sdl1 := `CREATE TYPE "public"."status" AS ENUM ('active', 'inactive');

COMMENT ON TYPE "public"."status" IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE TYPE "public"."status" AS ENUM ('active', 'inactive');`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})

		// -------------------- SchemaComment --------------------
		t.Run("SchemaComment", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateSchemaWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "schc_create")

				sdl := `CREATE SCHEMA "myschema";

COMMENT ON SCHEMA "myschema" IS 'Custom schema for application';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DropSchemaWithComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "schc_drop")

				sdl1 := `CREATE SCHEMA "myschema";

COMMENT ON SCHEMA "myschema" IS 'Schema comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("AddCommentToExistingSchema", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "schc_add")

				sdl1 := `CREATE SCHEMA "myschema";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE SCHEMA "myschema";

COMMENT ON SCHEMA "myschema" IS 'Added comment';`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})

			t.Run("DropCommentFromSchema", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "schc_dropcmt")

				sdl1 := `CREATE SCHEMA "myschema";

COMMENT ON SCHEMA "myschema" IS 'Comment';`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				sdl2 := `CREATE SCHEMA "myschema";`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
			})
		})
	})

	// ==================== ComplexScenarios ====================
	t.Run("ComplexScenarios", func(t *testing.T) {
		t.Parallel()

		// -------------------- MultipleObjects --------------------
		t.Run("MultipleObjects", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateMultipleObjectsAtOnce", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mo_create")

				// Single SDL creates table + view + function + trigger
				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyViewExists(t, database, "user_names"))
				a.True(stc.verifyFunctionExists(t, database, "set_ts"))
			})

			t.Run("AlterMultipleObjectsAtOnce", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mo_alter")

				// Create multiple objects
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name FROM "public"."users";

CREATE FUNCTION "public"."get_count"()
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 0;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Modify multiple objects at once
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_names" AS
SELECT id, name, email FROM "public"."users";

CREATE FUNCTION "public"."get_count"()
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 1;
END;
$$;`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
			})

			t.Run("DropMultipleObjectsAtOnce", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mo_drop")

				// Create multiple objects
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_view" AS
SELECT id FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Drop all
				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
				a.True(stc.verifyTableNotExists(t, database, "orders"))
			})

			t.Run("MixedOperations", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "mo_mixed")

				// Create initial objects
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."to_drop" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_to_drop" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Mixed: alter users, drop to_drop, create orders
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
				a.True(stc.verifyTableNotExists(t, database, "to_drop"))
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
			})
		})

		// -------------------- SchemaEvolution --------------------
		t.Run("SchemaEvolution", func(t *testing.T) {
			t.Parallel()

			t.Run("MultiStepEvolution", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "se_multi")

				// V1: Basic table
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.Equal(2, stc.getTableColumnCount(t, database, "users"))

				// V2: Add columns
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    "created_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.Equal(4, stc.getTableColumnCount(t, database, "users"))

				// V3: Add index and view
				sdl3 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    "created_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");

CREATE VIEW "public"."recent_users" AS
SELECT id, name FROM "public"."users" WHERE created_at > NOW() - INTERVAL '7 days';`

				err = stc.executeSDLRollout(t, database, sdl3)
				a.NoError(err)
				a.True(stc.verifyViewExists(t, database, "recent_users"))
			})

			t.Run("Idempotency", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "se_idempotent")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				// First application - should create table
				result1, err := stc.executeSDLRolloutWithResult(t, database, sdl)
				a.NoError(err)

				// Verify first rollout executed CREATE TABLE - SDL engine uses SDL as-is
				verifyExecutedSQL(t, result1, []string{sdl})
				a.True(stc.verifyTableExists(t, database, "public", "users"))

				// Second application (should be no-op - no SQL executed)
				result2, err := stc.executeSDLRolloutWithResult(t, database, sdl)
				a.NoError(err)

				// Verify second rollout executed NO SQL (idempotent)
				verifyNoSQL(t, result2)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("RollbackScenario", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "se_rollback")

				// V1: Initial schema
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.Equal(2, stc.getTableColumnCount(t, database, "users"))

				// V2: Add column
				sdl2 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.Equal(3, stc.getTableColumnCount(t, database, "users"))

				// Rollback to V1
				err = stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.Equal(2, stc.getTableColumnCount(t, database, "users"))
			})
		})

		// -------------------- CrossSchema --------------------
		t.Run("CrossSchema", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateMultipleSchemas", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cs_multi")

				sdl := `CREATE SCHEMA "admin";
CREATE SCHEMA "app";

CREATE TABLE "admin"."settings" (
    "id" serial NOT NULL,
    "key" varchar(255),
    "value" text,
    CONSTRAINT "pk_settings" PRIMARY KEY ("id")
);

CREATE TABLE "app"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "admin", "settings"))
				a.True(stc.verifyTableExists(t, database, "app", "users"))
			})

			t.Run("CrossSchemaReference", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cs_ref")

				sdl := `CREATE SCHEMA "core";

CREATE TABLE "core"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_core_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id"),
    CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "core"."users" ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "core", "users"))
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
			})

			t.Run("MoveObjectBetweenSchemas", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cs_move")

				// Create in public schema
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))

				// Move to new schema (drop from public, create in admin)
				sdl2 := `CREATE SCHEMA "admin";

CREATE TABLE "admin"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
				a.True(stc.verifyTableExists(t, database, "admin", "users"))
			})
		})

		// -------------------- LargeScale --------------------
		t.Run("LargeScale", func(t *testing.T) {
			t.Parallel()

			t.Run("CreateManyTables", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ls_many")

				// Create 20 tables at once
				var sdlBuilder strings.Builder
				for i := 1; i <= 20; i++ {
					sdlBuilder.WriteString(fmt.Sprintf(`CREATE TABLE "public"."table_%02d" (
    "id" serial NOT NULL,
    "data" varchar(255),
    CONSTRAINT "pk_table_%02d" PRIMARY KEY ("id")
);

`, i, i))
				}

				err := stc.executeSDLRollout(t, database, sdlBuilder.String())
				a.NoError(err)

				// Verify all tables exist
				for i := 1; i <= 20; i++ {
					a.True(stc.verifyTableExists(t, database, "public", fmt.Sprintf("table_%02d", i)))
				}
			})

			t.Run("ComplexDependencyGraph", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ls_complex")

				// Complex dependency: users -> orders -> order_items
				//                    users -> comments
				//                    products -> order_items
				//                    views on multiple tables
				// Intentionally scramble the order to test dependency sorting:
				// order_items depends on orders and products, but defined first
				// comments depends on users, but defined before users
				// orders depends on users, but defined before users
				// views depend on tables, but some defined before their dependencies
				sdl := `CREATE TABLE "public"."order_items" (
    "id" serial NOT NULL,
    "order_id" integer,
    "product_id" integer,
    "quantity" integer,
    CONSTRAINT "pk_order_items" PRIMARY KEY ("id"),
    CONSTRAINT "fk_items_order" FOREIGN KEY ("order_id") REFERENCES "public"."orders" ("id"),
    CONSTRAINT "fk_items_product" FOREIGN KEY ("product_id") REFERENCES "public"."products" ("id")
);

CREATE TABLE "public"."comments" (
    "id" serial NOT NULL,
    "user_id" integer,
    "content" text,
    CONSTRAINT "pk_comments" PRIMARY KEY ("id"),
    CONSTRAINT "fk_comments_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id")
);

CREATE VIEW "public"."order_details" AS
SELECT o.id, oi.quantity, p.name, p.price
FROM "public"."orders" o
JOIN "public"."order_items" oi ON o.id = oi.order_id
JOIN "public"."products" p ON oi.product_id = p.id;

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    "user_id" integer,
    "created_at" timestamp DEFAULT NOW(),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id"),
    CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id")
);

CREATE VIEW "public"."user_orders" AS
SELECT u.name, o.id as order_id, o.created_at
FROM "public"."users" u
JOIN "public"."orders" o ON u.id = o.user_id;

CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."products" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "price" decimal(10,2),
    CONSTRAINT "pk_products" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "users"))
				a.True(stc.verifyTableExists(t, database, "public", "products"))
				a.True(stc.verifyTableExists(t, database, "public", "orders"))
				a.True(stc.verifyTableExists(t, database, "public", "order_items"))
				a.True(stc.verifyTableExists(t, database, "public", "comments"))
				a.True(stc.verifyViewExists(t, database, "user_orders"))
				a.True(stc.verifyViewExists(t, database, "order_details"))
			})
		})

		// -------------------- EdgeCases --------------------
		t.Run("EdgeCases", func(t *testing.T) {
			t.Parallel()

			t.Run("EmptySDL", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_empty")

				// Create some objects first
				sdl1 := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl1)
				a.NoError(err)

				// Apply empty SDL to clear all
				sdl2 := ``
				err = stc.executeSDLRollout(t, database, sdl2)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "users"))
			})

			t.Run("NoChanges", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_nochange")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				// Apply same SDL twice
				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				a.True(stc.verifyTableExists(t, database, "public", "users"))
			})

			t.Run("SpecialCharacters", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "ec_special")

				// Test with special characters in comments and identifiers
				sdl := `CREATE TABLE "public"."user_data" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_user_data" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."user_data" IS 'Table with special chars: !@#$%^&*()';
COMMENT ON COLUMN "public"."user_data"."name" IS 'Unicode: 用户名称 - Имя пользователя';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableExists(t, database, "public", "user_data"))
			})
		})
	})

	// ==================== DriftHandling ====================
	t.Run("DriftHandling", func(t *testing.T) {
		t.Parallel()

		// -------------------- TableDrift --------------------
		t.Run("TableDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftAddColumn", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "td_addcol")

				// Apply initial SDL
				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Direct database modification (drift)
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" ADD COLUMN "extra" varchar(100)`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove the drift column)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.False(stc.verifyColumnExists(t, database, "users", "extra"))
			})

			t.Run("DriftDropColumn", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "td_dropcol")

				// Apply initial SDL with email column
				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Direct database modification (drift) - drop email
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" DROP COLUMN "email"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should re-add email column)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
			})

			t.Run("DriftModifyColumn", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "td_modcol")

				// Apply initial SDL
				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Direct database modification (drift) - change column type
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" ALTER COLUMN "name" TYPE text`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should restore original type)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftAddTable", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "td_addtbl")

				// Apply initial SDL
				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Direct database modification (drift) - add extra table
				stc.directExecuteSQL(t, database, `CREATE TABLE "public"."drift_table" ("id" serial PRIMARY KEY)`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift table)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyTableNotExists(t, database, "drift_table"))
			})
		})

		// -------------------- IndexDrift --------------------
		t.Run("IndexDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftAddIndex", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "id_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add index
				stc.directExecuteSQL(t, database, `CREATE INDEX "idx_drift" ON "public"."users" ("email")`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift index)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftDropIndex", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "id_drop")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: drop index
				stc.directExecuteSQL(t, database, `DROP INDEX "public"."idx_users_email"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should re-create index)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})
		})

		// -------------------- ConstraintDrift --------------------
		t.Run("ConstraintDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftAddConstraint", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cd_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add unique constraint
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" ADD CONSTRAINT "uk_drift" UNIQUE ("email")`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift constraint)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftDropConstraint", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cd_drop")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id"),
    CONSTRAINT "uk_users_email" UNIQUE ("email")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: drop constraint
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" DROP CONSTRAINT "uk_users_email"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should re-create constraint)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})
		})

		// -------------------- ViewDrift --------------------
		t.Run("ViewDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftModifyView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vd_modify")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_view" AS
SELECT id, name FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: modify view
				stc.directExecuteSQL(t, database, `CREATE OR REPLACE VIEW "public"."user_view" AS SELECT id, name, email FROM "public"."users"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should restore original view)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftAddView", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "vd_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add view
				stc.directExecuteSQL(t, database, `CREATE VIEW "public"."drift_view" AS SELECT id FROM "public"."users"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift view)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.False(stc.verifyViewExists(t, database, "drift_view"))
			})
		})

		// -------------------- FunctionDrift --------------------
		t.Run("FunctionDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftModifyFunction", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fd_modify")

				sdl := `CREATE FUNCTION "public"."add_nums"(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: modify function
				stc.directExecuteSQL(t, database, `CREATE OR REPLACE FUNCTION "public"."add_nums"(a integer, b integer) RETURNS integer LANGUAGE plpgsql AS $$ BEGIN RETURN a + b + 100; END; $$`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should restore original function)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftAddFunction", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "fd_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add function
				stc.directExecuteSQL(t, database, `CREATE FUNCTION "public"."drift_func"() RETURNS integer LANGUAGE plpgsql AS $$ BEGIN RETURN 0; END; $$`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift function)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.False(stc.verifyFunctionExists(t, database, "drift_func"))
			})
		})

		// -------------------- TriggerDrift --------------------
		t.Run("TriggerDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftAddTrigger", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trd_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add trigger
				stc.directExecuteSQL(t, database, `CREATE TRIGGER "drift_trigger" BEFORE UPDATE ON "public"."users" FOR EACH ROW EXECUTE FUNCTION "public"."set_ts"()`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift trigger)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftDropTrigger", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "trd_drop")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "updated_at" timestamp,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE FUNCTION "public"."set_ts"()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER "trg_users_ts"
BEFORE UPDATE ON "public"."users"
FOR EACH ROW
EXECUTE FUNCTION "public"."set_ts"();`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: drop trigger
				stc.directExecuteSQL(t, database, `DROP TRIGGER "trg_users_ts" ON "public"."users"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should re-create trigger)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})
		})

		// -------------------- SequenceDrift --------------------
		t.Run("SequenceDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftModifySequence", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sqd_modify")

				sdl := `CREATE SEQUENCE "public"."order_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: modify sequence
				stc.directExecuteSQL(t, database, `ALTER SEQUENCE "public"."order_seq" INCREMENT BY 10`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should restore original)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftAddSequence", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "sqd_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add sequence
				stc.directExecuteSQL(t, database, `CREATE SEQUENCE "public"."drift_seq"`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift sequence)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.False(stc.verifySequenceExists(t, database, "drift_seq"))
			})
		})

		// -------------------- CommentDrift --------------------
		t.Run("CommentDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftAddComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cmd_add")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add comment
				stc.directExecuteSQL(t, database, `COMMENT ON TABLE "public"."users" IS 'Drift comment'`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift comment)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})

			t.Run("DriftModifyComment", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cmd_modify")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

COMMENT ON TABLE "public"."users" IS 'Original comment';`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: modify comment
				stc.directExecuteSQL(t, database, `COMMENT ON TABLE "public"."users" IS 'Modified comment'`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should restore original comment)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
			})
		})

		// -------------------- CompositeDrift --------------------
		t.Run("CompositeDrift", func(t *testing.T) {
			t.Parallel()

			t.Run("DriftMultipleObjects", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cpd_multi")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."orders" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id")
);`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Multiple drifts
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" ADD COLUMN "extra" varchar(100)`)
				stc.directExecuteSQL(t, database, `CREATE TABLE "public"."drift_table" ("id" serial PRIMARY KEY)`)
				stc.directExecuteSQL(t, database, `CREATE INDEX "idx_drift" ON "public"."orders" ("id")`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should fix all drifts)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.False(stc.verifyColumnExists(t, database, "users", "extra"))
				a.True(stc.verifyTableNotExists(t, database, "drift_table"))
			})

			t.Run("DriftWithDependencies", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cpd_deps")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE VIEW "public"."user_view" AS
SELECT id, name FROM "public"."users";`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Drift: add column to table (view doesn't include it)
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" ADD COLUMN "email" varchar(255)`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should remove drift column, view should remain valid)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.False(stc.verifyColumnExists(t, database, "users", "email"))
				a.True(stc.verifyViewExists(t, database, "user_view"))
			})

			t.Run("DriftPartialMatch", func(t *testing.T) {
				t.Parallel()
				a := require.New(t)
				stc := setupSDLTestContext(t)
				defer stc.cleanup()

				database := stc.createTestPgDatabase(t, "cpd_partial")

				sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255),
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE INDEX "idx_users_email" ON "public"."users" ("email");`

				err := stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)

				// Partial drift: table OK, but index removed and extra column added
				stc.directExecuteSQL(t, database, `DROP INDEX "public"."idx_users_email"`)
				stc.directExecuteSQL(t, database, `ALTER TABLE "public"."users" ADD COLUMN "extra" varchar(100)`)
				stc.syncDatabase(t, database)

				// Re-apply SDL (should fix partial drifts)
				err = stc.executeSDLRollout(t, database, sdl)
				a.NoError(err)
				a.True(stc.verifyColumnExists(t, database, "users", "email"))
				a.False(stc.verifyColumnExists(t, database, "users", "extra"))
			})
		})
	})
}
