package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestSchemaAndDataUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "migration statement sheet",
			Content: []byte(migrationStatement1),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
	a.NoError(err)

	// Query schema.
	dbMetadataResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)}))
	a.NoError(err)
	dbMetadata := dbMetadataResp.Msg
	a.Equal(wantBookSchema, dbMetadata.Schema)

	sheetResp, err = ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "dataUpdateStatement",
			Content: []byte(dataUpdateStatement),
		},
	}))
	a.NoError(err)
	sheet = sheetResp.Msg

	// Create an issue that updates database data.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_DATA)
	a.NoError(err)

	resp, err := ctl.databaseServiceClient.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{
		Parent: database.Name,
		View:   v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
	}))
	a.NoError(err)
	changelogs := resp.Msg.Changelogs
	wantChangelogs := []*v1pb.Changelog{
		{
			Type:       v1pb.Changelog_DATA,
			Status:     v1pb.Changelog_DONE,
			Schema:     "",
			PrevSchema: "",
			Version:    "",
		},
		{
			Type:       v1pb.Changelog_MIGRATE,
			Status:     v1pb.Changelog_DONE,
			Schema:     dumpedSchema,
			PrevSchema: "",
			Version:    "",
		},
	}
	a.Equal(len(wantChangelogs), len(changelogs))
	for i, changelog := range changelogs {
		got := &v1pb.Changelog{
			Type:       changelog.Type,
			Status:     changelog.Status,
			Schema:     changelog.Schema,
			PrevSchema: changelog.PrevSchema,
			Version:    changelog.Version,
		}
		want := wantChangelogs[i]
		a.Equal(want, got)
	}
}

func TestGetLatestSchema(t *testing.T) {
	tests := []struct {
		name                 string
		dbType               storepb.Engine
		instanceID           string
		databaseName         string
		ddl                  string
		wantRawSchema        string
		wantSDL              string
		wantDatabaseMetadata *v1pb.DatabaseMetadata
	}{
		{
			name:         "MySQL",
			dbType:       storepb.Engine_MYSQL,
			instanceID:   "latest-schema-mysql",
			databaseName: "latestSchema",
			ddl:          `CREATE TABLE book(id INT, name TEXT);`,
			wantRawSchema: "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\n" +
				"SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n" +
				"--\n" +
				"-- Table structure for `book`\n" +
				"--\n" +
				"CREATE TABLE `book` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  `name` text DEFAULT NULL\n" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n" +
				"SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\n" +
				"SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n",
			wantSDL: "CREATE TABLE `book` (\n" +
				"  `id` INT DEFAULT NULL,\n" +
				"  `name` TEXT DEFAULT NULL\n" +
				") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_GENERAL_CI;\n\n",
			wantDatabaseMetadata: &v1pb.DatabaseMetadata{
				Name:         "instances/latest-schema-mysql/databases/latestSchema/metadata",
				CharacterSet: "utf8mb4",
				Collation:    "utf8mb4_general_ci",
				Schemas: []*v1pb.SchemaMetadata{
					{
						Tables: []*v1pb.TableMetadata{
							{
								Name:      "book",
								Engine:    "InnoDB",
								Collation: "utf8mb4_general_ci",
								Charset:   "utf8mb4",
								DataSize:  16384,
								Columns: []*v1pb.ColumnMetadata{
									{
										Name:       "id",
										Position:   1,
										Nullable:   true,
										HasDefault: true,
										Default: &v1pb.ColumnMetadata_DefaultNull{
											DefaultNull: true,
										},
										Type: "int",
									},
									{
										Name:       "name",
										Position:   2,
										Nullable:   true,
										Type:       "text",
										HasDefault: true,
										Default: &v1pb.ColumnMetadata_DefaultNull{
											DefaultNull: true,
										},
										CharacterSet: "utf8mb4",
										Collation:    "utf8mb4_general_ci",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:         "PostgreSQL",
			dbType:       storepb.Engine_POSTGRES,
			instanceID:   "latest-schema-postgres",
			databaseName: "latestSchema",
			ddl:          `CREATE TABLE book(id INT, name TEXT);`,
			wantRawSchema: `
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

CREATE TABLE "public"."book" (
    "id" integer,
    "name" text
);

`,
			wantSDL: ``,
			wantDatabaseMetadata: &v1pb.DatabaseMetadata{
				Name:         "instances/latest-schema-postgres/databases/latestSchema/metadata",
				Owner:        "postgres",
				CharacterSet: "UTF8",
				Collation:    "en_US.UTF-8",
				Schemas: []*v1pb.SchemaMetadata{
					{
						Name:  "public",
						Owner: "pg_database_owner",
						Tables: []*v1pb.TableMetadata{
							{
								Name:     "book",
								Owner:    "postgres",
								DataSize: 8192,
								Columns: []*v1pb.ColumnMetadata{
									{Name: "id", Position: 1, Nullable: true, Type: "integer"},
									{Name: "name", Position: 2, Nullable: true, Type: "text"},
								},
							},
						},
					},
				},
			},
		},
	}
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer func() {
		_ = ctl.Close(ctx)
	}()
	environmentName := strings.ToLower(t.Name())
	environment, err := ctl.createEnvironment(ctx, environmentName, environmentName)
	a.NoError(err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := require.New(t)
			var instance *v1pb.Instance
			switch test.dbType {
			case storepb.Engine_POSTGRES:
				pgContainer, err := getPgContainer(ctx)
				defer func() {
					pgContainer.Close(ctx)
				}()
				a.NoError(err)
				instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
					InstanceId: test.instanceID,
					Instance: &v1pb.Instance{
						Title:       test.name,
						Engine:      v1pb.Engine_POSTGRES,
						Environment: environment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "postgres", Password: "root-password", Id: "admin"}},
					},
				}))
				a.NoError(err)
				instance = instanceResp.Msg
			case storepb.Engine_MYSQL:
				mysqlContainer, err := getMySQLContainer(ctx)
				defer func() {
					mysqlContainer.Close(ctx)
				}()
				a.NoError(err)

				instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
					InstanceId: test.instanceID,
					Instance: &v1pb.Instance{
						Title:       "mysqlInstance",
						Engine:      v1pb.Engine_MYSQL,
						Environment: environment.Name,
						Activation:  true,
						DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: mysqlContainer.host, Port: mysqlContainer.port, Username: "root", Password: "root-password", Id: "admin"}},
					},
				}))
				a.NoError(err)
				instance = instanceResp.Msg
			default:
				a.FailNow("unsupported db type")
			}

			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, test.databaseName, "postgres")
			a.NoError(err)

			databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, test.databaseName),
			}))
			a.NoError(err)
			database := databaseResp.Msg

			ddlSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
				Parent: ctl.project.Name,
				Sheet: &v1pb.Sheet{
					Title:   "test ddl",
					Content: []byte(test.ddl),
				},
			}))
			a.NoError(err)
			ddlSheet := ddlSheetResp.Msg

			// Create an issue that updates database schema.
			err = ctl.changeDatabase(ctx, ctl.project, database, ddlSheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
			a.NoError(err)

			latestSchemaResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
				Name: fmt.Sprintf("%s/schema", database.Name),
			}))
			a.NoError(err)
			latestSchema := latestSchemaResp.Msg
			a.Equal(test.wantRawSchema, latestSchema.Schema)
			if test.dbType == storepb.Engine_MYSQL {
				latestSchemaSDLResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
					Name:      fmt.Sprintf("%s/schema", database.Name),
					SdlFormat: true,
				}))
				a.NoError(err)
				latestSchemaSDL := latestSchemaSDLResp.Msg
				a.Equal(test.wantSDL, latestSchemaSDL.Schema)
			}
			latestSchemaMetadataResp, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
				Name: fmt.Sprintf("%s/metadata", database.Name),
			}))
			a.NoError(err)
			latestSchemaMetadata := latestSchemaMetadataResp.Msg
			diff := cmp.Diff(test.wantDatabaseMetadata, latestSchemaMetadata, protocmp.Transform())
			a.Empty(diff)
		})
	}
}

func TestMarkTaskAsDone(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Provision an instance.
	instanceRootDir := t.TempDir()
	instanceName := "testInstance1"
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, instanceName)
	a.NoError(err)

	// Add an instance.
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_SQLITE,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	// Create an issue that creates a database.
	databaseName := "testSchemaUpdate"
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil, databaseName, "")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "migration statement sheet",
			Content: []byte(migrationStatement1),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create an issue that updates database schema.
	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{database.Name},
							Sheet:   sheet.Name,
							Type:    v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
						},
					},
				},
			},
		},
	}))
	a.NoError(err)
	plan := planResp.Msg
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: ctl.project.Name,
		Issue: &v1pb.Issue{
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Title:       fmt.Sprintf("change database %s", database.Name),
			Description: fmt.Sprintf("change database %s", database.Name),
			Plan:        plan.Name,
		},
	}))
	a.NoError(err)
	issue := issueResp.Msg
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: ctl.project.Name, Rollout: &v1pb.Rollout{Plan: plan.Name}}))
	a.NoError(err)
	rollout := rolloutResp.Msg

	// Skip the task.
	for _, stage := range rollout.Stages {
		for _, task := range stage.Tasks {
			_, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
				Parent: stage.Name,
				Tasks:  []string{task.Name},
				Reason: "skip it!",
			}))
			a.NoError(err)
		}
	}

	err = ctl.waitRollout(ctx, issue.Name, rollout.Name)
	a.NoError(err)

	// Query schema.
	dbMetadataResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)}))
	a.NoError(err)
	dbMetadata := dbMetadataResp.Msg
	a.Equal("", dbMetadata.Schema)
}
