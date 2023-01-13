//go:build mysql
// +build mysql

package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	// Import pg driver.
	// init() in pgx will register it's pgx driver.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
)

var noSQLReviewPolicy = []api.TaskCheckResult{
	{
		Status:    api.TaskCheckStatusSuccess,
		Namespace: api.AdvisorNamespace,
		Code:      common.Ok.Int(),
		Title:     "Empty SQL review policy or disabled",
		Content:   "",
	},
}

type test struct {
	Statement string
	Result    []api.TaskCheckResult
	Run       bool
}

func TestSQLReviewForPostgreSQL(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath   = filepath.Join("test-data", "sql_review_pg.yaml")
		statements = []string{
			`CREATE TABLE "user"(
				id INT,
				name VARCHAR(255) NOT NULL,
				room_id INT NOT NULL,
				creator_id INT NOT NULL,
				created_ts TIMESTAMP NOT NULL,
				updater_id INT NOT NULL,
				updated_ts TIMESTAMP NOT NULL,
				CONSTRAINT pk_user_id PRIMARY KEY (id),
				CONSTRAINT uk_user_id_name UNIQUE (id, name)
				)`,
			`CREATE TABLE "userTable"(
				id INT,
				name VARCHAR(255),
				"roomId" INT,
				CONSTRAINT uk1 UNIQUE (id, name),
				CONSTRAINT fk1 FOREIGN KEY ("roomId") REFERENCES room(id)
				)`,
		}
		databaseName = "testsqlreview"
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
		Name: "Test SQL Review Project",
		Key:  "TestSQLReview",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	policyPayload, err := prodTemplateSQLReviewPolicyForPostgreSQL()
	a.NoError(err)

	_, err = ctl.upsertPolicy(api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
		Payload:      &policyPayload,
	})
	a.NoError(err)

	policy, err := ctl.upsertPolicy(api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
		Payload:      &policyPayload,
	})
	a.NoError(err)
	a.NotNil(policy.Environment)

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

	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []test{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	err = yaml.Unmarshal(byteValue, &tests)
	a.NoError(err)

	for i, t := range tests {
		result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, t.Statement, t.Run)
		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}

	// disable the SQL review policy
	disable := string(api.Archived)
	_, err = ctl.upsertPolicy(api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
		Payload:      &policyPayload,
		RowStatus:    &disable,
	})
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, statements[0], false)
	a.Equal(noSQLReviewPolicy, result)

	// delete the SQL review policy
	err = ctl.deletePolicy(api.PolicyDelete{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
	})
	a.NoError(err)

	result = createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, statements[0], false)
	a.Equal(noSQLReviewPolicy, result)
}

func TestSQLReviewForMySQL(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath     = filepath.Join("test-data", "sql_review_mysql.yaml")
		databaseName = "testsqlreview"
		statements   = []string{
			"CREATE TABLE user(" +
				"id INT PRIMARY KEY COMMENT 'comment'," +
				"name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'comment'," +
				"room_id INT NOT NULL DEFAULT 0 COMMENT 'comment'," +
				"creator_id INT NOT NULL DEFAULT 0 COMMENT 'comment'," +
				"created_ts TIMESTAMP NOT NULL DEFAULT NOW() COMMENT 'comment'," +
				"updater_id INT NOT NULL DEFAULT 0 COMMENT 'comment'," +
				"updated_ts TIMESTAMP NOT NULL DEFAULT NOW() ON UPDATE NOW() COMMENT 'comment'," +
				"INDEX idx_user_name(name)," +
				"UNIQUE KEY uk_user_id_name(id, name)" +
				") ENGINE = INNODB COMMENT 'comment'",
			"CREATE TABLE userTable(" +
				"id INT NOT NULL," +
				"name VARCHAR(255) CHARSET ascii," +
				"roomId INT," +
				"time_created TIMESTAMP NOT NULL DEFAULT NOW() ON UPDATE NOW() COMMENT 'comment'," +
				"time_updated TIMESTAMP NOT NULL DEFAULT NOW() ON UPDATE NOW() COMMENT 'comment'," +
				"content BLOB NOT NULL COMMENT 'comment'," +
				"json_content JSON NOT NULL COMMENT 'comment'," +
				"INDEX idx1(name)," +
				"UNIQUE KEY uk1(id, name)," +
				"FOREIGN KEY fk1(roomId) REFERENCES room(id)," +
				"INDEX idx_userTable_content(content)" +
				") ENGINE = CSV COLLATE latin1_bin",
		}
		valueTable = `(SELECT 1 AS id, 'a' AS name WHERE 1=1 UNION ALL
			SELECT 2 AS id, 'b' AS name WHERE 1=1 UNION ALL
			SELECT 3 AS id, 'c' AS name WHERE 1=1 UNION ALL
			SELECT 4 AS id, 'd' AS name WHERE 1=1 UNION ALL
			SELECT 5 AS id, 'e' AS name WHERE 1=1 UNION ALL
			SELECT 6 AS id, 'f' AS name WHERE 1=1 UNION ALL
			SELECT 7 AS id, 'g' AS name WHERE 1=1 UNION ALL
			SELECT 8 AS id, 'h' AS name WHERE 1=1 UNION ALL
			SELECT 9 AS id, 'i' AS name WHERE 1=1 UNION ALL
			SELECT 10 AS id, 'j' AS name WHERE 1=1) value_table`
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
		Name: "Test SQL Review Project",
		Key:  "TestSQLReview",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	policyPayload, err := prodTemplateSQLReviewPolicyForMySQL()
	a.NoError(err)

	_, err = ctl.upsertPolicy(api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
		Payload:      &policyPayload,
	})
	a.NoError(err)

	policy, err := ctl.upsertPolicy(api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
		Payload:      &policyPayload,
	})
	a.NoError(err)
	a.NotNil(policy.Environment)

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
	databases, err = ctl.getDatabases(api.DatabaseFind{
		InstanceID: &instance.ID,
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

	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []test{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	err = yaml.Unmarshal(byteValue, &tests)
	a.NoError(err)

	for i, t := range tests {
		result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, t.Statement, t.Run)
		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result, t.Statement)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}

	// test for dry-run-dml
	initialStmts := []string{
		"CREATE TABLE test(" +
			"id INT PRIMARY KEY COMMENT 'comment'," +
			"name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'comment'," +
			"room_id INT NOT NULL DEFAULT 0 COMMENT 'comment'," +
			"creator_id INT NOT NULL DEFAULT 0 COMMENT 'comment'," +
			"created_ts TIMESTAMP NOT NULL DEFAULT NOW() COMMENT 'comment'," +
			"updater_id INT NOT NULL DEFAULT 0 COMMENT 'comment'," +
			"updated_ts TIMESTAMP NOT NULL DEFAULT NOW() ON UPDATE NOW() COMMENT 'comment'," +
			"INDEX idx_test_name(name)," +
			"UNIQUE KEY uk_test_id_name(id, name)" +
			") ENGINE = INNODB COMMENT 'comment';",
		`INSERT INTO test(id, name) VALUES (1, 'a'), (2, 'b'), (3, 'c'), (4, 'd');`,
	}
	for _, stmt := range initialStmts {
		createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, stmt, true /* wait */)
	}
	countSQL := "SELECT count(*) FROM test WHERE 1=1;"
	dmlSQL := "INSERT INTO test SELECT * FROM " + valueTable
	origin, err := ctl.query(instance, databaseName, countSQL)
	a.NoError(err)
	a.Equal("[[\"count(*)\"],[\"BIGINT\"],[[\"4\"]]]", origin)
	createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, dmlSQL, false /* wait */)
	finial, err := ctl.query(instance, databaseName, countSQL)
	a.NoError(err)
	a.Equal(origin, finial)

	// disable the SQL review policy
	disable := string(api.Archived)
	_, err = ctl.upsertPolicy(api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
		Payload:      &policyPayload,
		RowStatus:    &disable,
	})
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, statements[0], false)
	a.Equal(noSQLReviewPolicy, result)

	// delete the SQL review policy
	err = ctl.deletePolicy(api.PolicyDelete{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   prodEnvironment.ID,
		Type:         api.PolicyTypeSQLReview,
	})
	a.NoError(err)

	result = createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, statements[0], false)
	a.Equal(noSQLReviewPolicy, result)
}

func createIssueAndReturnSQLReviewResult(a *require.Assertions, ctl *controller, databaseID int, projectID int, statement string, wait bool) []api.TaskCheckResult {
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    databaseID,
				Statement:     statement,
			},
		},
	})
	a.NoError(err)

	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectID,
		Name:          "update schema for database",
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   "This updates the schema of database",
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	result, err := ctl.GetSQLReviewResult(issue.ID)
	a.NoError(err)

	if wait {
		a.Equal(1, len(result))
		a.Equal(common.Ok.Int(), result[0].Code, result[0])
		status, err := ctl.waitIssuePipeline(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)
	}

	return result
}
