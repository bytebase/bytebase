package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestSyncerForPostgreSQL(t *testing.T) {
	const (
		databaseName = "test_sync_postgresql_schema_db"
		createSchema = `
		CREATE SCHEMA schema1;
		CREATE TABLE schema1.trd (
			"A" int DEFAULT NULL,
			"B" int DEFAULT NULL,
			c int DEFAULT NULL,
			UNIQUE ("A","B",c)
		  );
		  CREATE TABLE "TFK" (
			a int DEFAULT NULL,
			b int DEFAULT NULL,
			c int DEFAULT NULL,
			CONSTRAINT tfk_ibfk_1 FOREIGN KEY (a, b, c) REFERENCES schema1.trd ("A", "B", c)
		  );
		`
		expectedSchema = `
		{
			"name":"test_sync_postgresql_schema_db",
			"schemas":[
			   {
				  "name":"public",
				  "tables":[
					 {
						"name":"TFK",
						"columns":[
						   {
							  "name":"a",
							  "position":1,
							  "default":"",
							  "nullable":true,
							  "type":"integer"
						   },
						   {
							  "name":"b",
							  "position":2,
							  "default":"",
							  "nullable":true,
							  "type":"integer"
						   },
						   {
							  "name":"c",
							  "position":3,
							  "default":"",
							  "nullable":true,
							  "type":"integer"
						   }
						],
						"foreignKeys":[
						   {
							  "name":"tfk_ibfk_1",
							  "columns":[
								 "a",
								 "b",
								 "c"
							  ],
							  "referencedSchema":"schema1",
							  "referencedTable":"trd",
							  "referencedColumns":[
								 "A",
								 "B",
								 "c"
							  ],
							  "onDelete":"NO ACTION",
							  "onUpdate":"NO ACTION",
							  "matchType":"SIMPLE"
						   }
						]
					 }
				  ]
			   },
			   {
				  "name":"schema1",
				  "tables":[
					 {
						"name":"trd",
						"columns":[
						   {
							  "name":"A",
							  "position":1,
							  "default":"",
							  "nullable":true,
							  "type":"integer"
						   },
						   {
							  "name":"B",
							  "position":2,
							  "default":"",
							  "nullable":true,
							  "type":"integer"
						   },
						   {
							  "name":"c",
							  "position":3,
							  "default":"",
							  "nullable":true,
							  "type":"integer"
						   }
						],
						"indexes":[
						   {
							  "name":"trd_A_B_c_key",
							  "expressions":[
								 "\"A\"",
								 "\"B\"",
								 "c"
							  ],
							  "type":"btree",
							  "unique":true
						   }
						],
						"indexSize":"8192"
					 }
				  ]
			   }
			],
			"characterSet":"UTF8",
			"collation":"en_US.UTF-8"
		 }
		`
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	stopInstance := postgres.SetupTestInstance(t, pgPort, resourceDirOverride)
	defer stopInstance()

	pgDB, err := sql.Open("pgx", fmt.Sprintf("host=/tmp port=%d user=root database=postgres", pgPort))
	a.NoError(err)
	defer pgDB.Close()

	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)

	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Syncer For PostgreSQL",
		Key:  "TestSyncerForPostgreSQL",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "pgInstance",
		Engine:        db.Postgres,
		Host:          "/tmp",
		Port:          strconv.Itoa(pgPort),
		Username:      "bytebase",
		Password:      "bytebase",
	})
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Nil(databases)

	err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil)
	a.NoError(err)

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
				Statement:     createSchema,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("Create sequence for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("Create sequence of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	metadata, err := ctl.getLatestSchemaMetadata(database.ID)
	a.NoError(err)

	var latestSchemaMetadata storepb.DatabaseMetadata
	err = protojson.Unmarshal([]byte(metadata), &latestSchemaMetadata)
	a.NoError(err)

	var expectedSchemaMetadata storepb.DatabaseMetadata
	err = protojson.Unmarshal([]byte(expectedSchema), &expectedSchemaMetadata)
	a.NoError(err)
	a.Equal(&expectedSchemaMetadata, &latestSchemaMetadata)
}

func TestSyncerForMySQL(t *testing.T) {
	const (
		databaseName = "test_sync_mysql_schema_db"
		createSchema = `
		CREATE TABLE trd (
			a int DEFAULT NULL,
			b int DEFAULT NULL,
			c int DEFAULT NULL,
			UNIQUE KEY a (a,b,c)
		  );
		  CREATE TABLE tfk (
			a int DEFAULT NULL,
			b int DEFAULT NULL,
			c int DEFAULT NULL,
			KEY a (a,b,c),
			CONSTRAINT tfk_ibfk_1 FOREIGN KEY (a, b, c) REFERENCES trd (a, b, c)
		  );
		`
		expectedSchema = `{
			"name":"test_sync_mysql_schema_db",
			"schemas":[
			   {
				  "tables":[
					 {
						"name":"tfk",
						"columns":[
						   {
							  "name":"a",
							  "position":1,
							  "nullable":true,
							  "type":"int"
						   },
						   {
							  "name":"b",
							  "position":2,
							  "nullable":true,
							  "type":"int"
						   },
						   {
							  "name":"c",
							  "position":3,
							  "nullable":true,
							  "type":"int"
						   }
						],
						"indexes":[
						   {
							  "name":"a",
							  "expressions":[
								 "a",
								 "b",
								 "c"
							  ],
							  "type":"BTREE",
							  "visible":true
						   }
						],
						"engine":"InnoDB",
						"collation":"utf8mb4_general_ci",
						"dataSize":"16384",
						"indexSize":"16384",
						"foreignKeys":[
						   {
							  "name":"tfk_ibfk_1",
							  "columns":[
								 "a",
								 "b",
								 "c"
							  ],
							  "referencedTable":"trd",
							  "referencedColumns":[
								 "a",
								 "b",
								 "c"
							  ],
							  "onDelete":"NO ACTION",
							  "onUpdate":"NO ACTION",
							  "matchType":"NONE"
						   }
						]
					 },
					 {
						"name":"trd",
						"columns":[
						   {
							  "name":"a",
							  "position":1,
							  "nullable":true,
							  "type":"int"
						   },
						   {
							  "name":"b",
							  "position":2,
							  "nullable":true,
							  "type":"int"
						   },
						   {
							  "name":"c",
							  "position":3,
							  "nullable":true,
							  "type":"int"
						   }
						],
						"indexes":[
						   {
							  "name":"a",
							  "expressions":[
								 "a",
								 "b",
								 "c"
							  ],
							  "type":"BTREE",
							  "unique":true,
							  "visible":true
						   }
						],
						"engine":"InnoDB",
						"collation":"utf8mb4_general_ci",
						"dataSize":"16384",
						"indexSize":"16384"
					 }
				  ]
			   }
			],
			"characterSet":"utf8mb4",
			"collation":"utf8mb4_general_ci"
		 }`
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	mysqlDB, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", mysqlPort))
	a.NoError(err)
	defer mysqlDB.Close()

	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Sync MySQL Schema",
		Key:  "TestSyncMySQLSchema",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "mysqlInstance",
		Engine:        db.MySQL,
		Host:          "127.0.0.1",
		Port:          strconv.Itoa(mysqlPort),
		Username:      "bytebase",
		Password:      "bytebase",
	})
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Nil(databases)

	err = ctl.createDatabase(project, instance, databaseName, "", nil)
	a.NoError(err)

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
				Statement:     createSchema,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("Create sequence for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("Create sequence of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	metadata, err := ctl.getLatestSchemaMetadata(database.ID)
	a.NoError(err)

	var latestSchemaMetadata storepb.DatabaseMetadata
	err = protojson.Unmarshal([]byte(metadata), &latestSchemaMetadata)
	a.NoError(err)

	var expectedSchemaMetadata storepb.DatabaseMetadata
	err = protojson.Unmarshal([]byte(expectedSchema), &expectedSchemaMetadata)
	a.NoError(err)
	a.Equal(&expectedSchemaMetadata, &latestSchemaMetadata)
}
