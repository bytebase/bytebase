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

// sdlRolloutResult captures the result of an SDL rollout including executed SQL statements.
type sdlRolloutResult struct {
	// ExecutedStatements contains the SQL statements that were actually executed during the rollout.
	ExecutedStatements []string
}

// TestPgSDLRollout tests PostgreSQL SDL rollout workflow end-to-end.
// This test focuses on verifying the overall SDL rollout workflow works correctly.
// Detailed logic tests for specific SDL operations should be in backend/plugin/schema/pg tests.
func TestPgSDLRollout(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()

	// Create a single shared PostgreSQL container for all subtests
	pgContainer, err := getPgContainer(ctx)
	a.NoError(err)
	defer pgContainer.Close(ctx)

	// Create a single shared Bytebase controller and instance for all subtests
	ctl := &controller{}
	ctx, err = ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create shared instance in Bytebase
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst")[:8],
		Instance: &v1pb.Instance{
			Title:       "SDL Test Instance",
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
	instance := instanceResp.Msg

	t.Run("BasicWorkflow", func(t *testing.T) {
		a := require.New(t)

		// Create unique database name
		dbName := fmt.Sprintf("sdl_workflow_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

		// Create database directly in PostgreSQL
		_, err := pgContainer.db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		a.NoError(err)

		// Create database in Bytebase
		err = ctl.createDatabase(ctx, ctl.project, instance, nil, dbName, "postgres")
		a.NoError(err)

		// Get database
		databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbName),
		}))
		a.NoError(err)
		database := databaseResp.Msg

		// Step 1: Create initial schema with basic objects
		initialSDL := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."posts" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "title" varchar(500) NOT NULL,
    "content" text,
    CONSTRAINT "pk_posts" PRIMARY KEY ("id"),
    CONSTRAINT "fk_posts_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id")
);

CREATE INDEX "idx_posts_user_id" ON "public"."posts"("user_id");`

		result, err := executeSDLRolloutWithResult(ctx, ctl, database, initialSDL)
		a.NoError(err)
		a.NotEmpty(result.ExecutedStatements, "Expected SQL statements to be executed")

		// Verify initial schema was created
		a.True(verifyTableExists(ctx, ctl, database, "users"))
		a.True(verifyTableExists(ctx, ctl, database, "posts"))
		a.Equal(3, getTableColumnCount(ctx, ctl, database, "users"))
		a.Equal(4, getTableColumnCount(ctx, ctl, database, "posts"))

		// Step 2: Update schema - add column and new table
		updatedSDL := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "created_at" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."posts" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "title" varchar(500) NOT NULL,
    "content" text,
    CONSTRAINT "pk_posts" PRIMARY KEY ("id"),
    CONSTRAINT "fk_posts_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id")
);

CREATE INDEX "idx_posts_user_id" ON "public"."posts"("user_id");

CREATE TABLE "public"."comments" (
    "id" serial NOT NULL,
    "post_id" integer NOT NULL,
    "content" text NOT NULL,
    CONSTRAINT "pk_comments" PRIMARY KEY ("id"),
    CONSTRAINT "fk_comments_post" FOREIGN KEY ("post_id") REFERENCES "public"."posts"("id")
);`

		result, err = executeSDLRolloutWithResult(ctx, ctl, database, updatedSDL)
		a.NoError(err)
		a.NotEmpty(result.ExecutedStatements, "Expected ALTER/CREATE statements to be executed")

		// Verify schema was updated
		a.True(verifyTableExists(ctx, ctl, database, "users"))
		a.True(verifyTableExists(ctx, ctl, database, "posts"))
		a.True(verifyTableExists(ctx, ctl, database, "comments"))
		a.Equal(4, getTableColumnCount(ctx, ctl, database, "users"))
		a.True(verifyColumnExists(ctx, ctl, database, "users", "created_at"))

		// Step 3: Remove objects - drop comments table and posts index
		finalSDL := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    "email" varchar(255),
    "created_at" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);

CREATE TABLE "public"."posts" (
    "id" serial NOT NULL,
    "user_id" integer NOT NULL,
    "title" varchar(500) NOT NULL,
    "content" text,
    CONSTRAINT "pk_posts" PRIMARY KEY ("id"),
    CONSTRAINT "fk_posts_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id")
);`

		result, err = executeSDLRolloutWithResult(ctx, ctl, database, finalSDL)
		a.NoError(err)
		a.NotEmpty(result.ExecutedStatements, "Expected DROP statements to be executed")

		// Verify objects were removed
		a.True(verifyTableExists(ctx, ctl, database, "users"))
		a.True(verifyTableExists(ctx, ctl, database, "posts"))
		a.True(verifyTableNotExists(ctx, ctl, database, "comments"))
	})

	t.Run("EmptySDL", func(t *testing.T) {
		a := require.New(t)

		// Create unique database name
		dbName := fmt.Sprintf("sdl_empty_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

		// Create database directly in PostgreSQL
		_, err := pgContainer.db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		a.NoError(err)

		// Create database in Bytebase
		err = ctl.createDatabase(ctx, ctl.project, instance, nil, dbName, "postgres")
		a.NoError(err)

		// Get database
		databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbName),
		}))
		a.NoError(err)
		database := databaseResp.Msg

		// Create initial schema
		initialSDL := `CREATE TABLE "public"."test_table" (
    "id" serial NOT NULL,
    CONSTRAINT "pk_test" PRIMARY KEY ("id")
);`

		_, err = executeSDLRolloutWithResult(ctx, ctl, database, initialSDL)
		a.NoError(err)
		a.True(verifyTableExists(ctx, ctl, database, "test_table"))

		// Apply empty SDL to drop all objects
		result, err := executeSDLRolloutWithResult(ctx, ctl, database, "")
		a.NoError(err)
		a.NotEmpty(result.ExecutedStatements, "Expected DROP statements for empty SDL")

		// Verify table was dropped
		a.True(verifyTableNotExists(ctx, ctl, database, "test_table"))
	})

	t.Run("NoChangesNeeded", func(t *testing.T) {
		a := require.New(t)

		// Create unique database name
		dbName := fmt.Sprintf("sdl_nochange_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

		// Create database directly in PostgreSQL
		_, err := pgContainer.db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		a.NoError(err)

		// Create database in Bytebase
		err = ctl.createDatabase(ctx, ctl.project, instance, nil, dbName, "postgres")
		a.NoError(err)

		// Get database
		databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbName),
		}))
		a.NoError(err)
		database := databaseResp.Msg

		// Create initial schema
		sdl := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

		_, err = executeSDLRolloutWithResult(ctx, ctl, database, sdl)
		a.NoError(err)

		// Apply same SDL again - should be a no-op
		result, err := executeSDLRolloutWithResult(ctx, ctl, database, sdl)
		a.NoError(err)

		// Verify no SQL was executed (schema already matches)
		a.Empty(result.ExecutedStatements,
			"Expected no SQL statements to be executed, but got %d statements:\n%s",
			len(result.ExecutedStatements),
			strings.Join(result.ExecutedStatements, "\n---\n"))
	})

	t.Run("DriftDetection", func(t *testing.T) {
		a := require.New(t)

		// Create unique database name
		dbName := fmt.Sprintf("sdl_drift_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

		// Create database directly in PostgreSQL
		_, err := pgContainer.db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		a.NoError(err)

		// Create database in Bytebase
		err = ctl.createDatabase(ctx, ctl.project, instance, nil, dbName, "postgres")
		a.NoError(err)

		// Get database
		databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbName),
		}))
		a.NoError(err)
		database := databaseResp.Msg

		// Create initial schema via SDL
		initialSDL := `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "name" varchar(255) NOT NULL,
    CONSTRAINT "pk_users" PRIMARY KEY ("id")
);`

		_, err = executeSDLRolloutWithResult(ctx, ctl, database, initialSDL)
		a.NoError(err)

		// Introduce drift by directly modifying the database
		directExecuteSQL(pgContainer, database, `ALTER TABLE "public"."users" ADD COLUMN "email" varchar(255);`)
		_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{
			Name: database.Name,
		}))
		a.NoError(err)

		// Verify drift was introduced
		a.True(verifyColumnExists(ctx, ctl, database, "users", "email"))

		// Re-apply original SDL to fix drift
		result, err := executeSDLRolloutWithResult(ctx, ctl, database, initialSDL)
		a.NoError(err)
		a.NotEmpty(result.ExecutedStatements, "Expected statements to fix drift")

		// Verify drift was fixed (email column should be removed)
		a.False(verifyColumnExists(ctx, ctl, database, "users", "email"))
		a.Equal(2, getTableColumnCount(ctx, ctl, database, "users"))
	})
}

// executeSDLRolloutWithResult performs a complete SDL rollout and returns the result including executed SQL.
func executeSDLRolloutWithResult(ctx context.Context, ctl *controller, database *v1pb.Database, sdlContent string) (*sdlRolloutResult, error) {
	// Create a DECLARATIVE release with SDL content
	// Empty SDL content is allowed for DECLARATIVE releases (represents dropping all objects)
	releaseResp, err := ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent: ctl.project.Name,
		Release: &v1pb.Release{
			Type: v1pb.Release_DECLARATIVE,
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
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
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
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: plan.Name,
	}))
	if err != nil {
		return nil, err
	}
	rollout := rolloutResp.Msg

	// Wait for rollout to complete (without issue approval flow)
	err = ctl.waitRolloutWithoutApproval(ctx, rollout.Name)
	if err != nil {
		return nil, err
	}

	// Get executed SQL statements from task run logs
	executedStatements, err := getExecutedStatements(ctx, ctl, rollout.Name)
	if err != nil {
		return nil, err
	}

	return &sdlRolloutResult{
		ExecutedStatements: executedStatements,
	}, nil
}

// getExecutedStatements retrieves the SQL statements that were actually executed during the rollout.
func getExecutedStatements(ctx context.Context, ctl *controller, rolloutName string) ([]string, error) {
	var statements []string

	// Get the rollout to find all tasks
	rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
		Name: rolloutName,
	}))
	if err != nil {
		return nil, err
	}

	// Iterate through all stages and tasks to get task run logs
	for _, stage := range rolloutResp.Msg.Stages {
		for _, task := range stage.Tasks {
			// Get task runs for this task
			taskRunsResp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
				Parent: task.Name,
			}))
			if err != nil {
				return nil, err
			}

			// Get logs for each task run
			for _, taskRun := range taskRunsResp.Msg.TaskRuns {
				logResp, err := ctl.rolloutServiceClient.GetTaskRunLog(ctx, connect.NewRequest(&v1pb.GetTaskRunLogRequest{
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

// verifyTableExists checks if a table exists in the database schema.
func verifyTableExists(ctx context.Context, ctl *controller, database *v1pb.Database, tableName string) bool {
	metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	if err != nil {
		panic(err)
	}

	for _, schema := range metadata.Msg.Schemas {
		if schema.Name == "public" {
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
func verifyTableNotExists(ctx context.Context, ctl *controller, database *v1pb.Database, tableName string) bool {
	return !verifyTableExists(ctx, ctl, database, tableName)
}

// verifyColumnExists checks if a column exists in a table.
func verifyColumnExists(ctx context.Context, ctl *controller, database *v1pb.Database, tableName, columnName string) bool {
	metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	if err != nil {
		panic(err)
	}

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
func getTableColumnCount(ctx context.Context, ctl *controller, database *v1pb.Database, tableName string) int {
	metadata, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: database.Name + "/metadata",
	}))
	if err != nil {
		panic(err)
	}

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

// directExecuteSQL executes SQL directly on the database (for drift tests).
func directExecuteSQL(pgContainer *Container, database *v1pb.Database, sqlStmt string) {
	// Extract database name from resource name
	parts := strings.Split(database.Name, "/")
	dbName := parts[len(parts)-1]

	// Connect to the specific database
	connStr := fmt.Sprintf("host=%s port=%s user=postgres password=root-password dbname=%s sslmode=disable",
		pgContainer.host, pgContainer.port, dbName)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}
}
