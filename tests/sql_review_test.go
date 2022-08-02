//go:build mysql
// +build mysql

package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	// Import pg driver.
	// init() in pgx/v4/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
)

var (
	sqlReviewAccessErr = fmt.Sprintf(`http response error code %d body "{\"message\":\"%s\"}\n"`, http.StatusForbidden, api.FeatureSQLReviewPolicy.AccessErrorMessage())
	noSQLReviewPolicy  = []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusWarn,
			Namespace: api.AdvisorNamespace,
			Code:      advisor.NotFound.Int(),
			Title:     "Empty SQL review policy or disabled",
			Content:   "",
		},
	}
)

type test struct {
	statement string
	result    []api.TaskCheckResult
}

func TestSQLReviewForPostgreSQL(t *testing.T) {
	var (
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
		databaseName = "testSQLReview"
		tests        = []test{
			{
				statement: statements[0],
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
			},
			{
				statement: "CREATE TABLE user(id);",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementSyntaxError.Int(),
						Title:     advisor.SyntaxErrorTitle,
						Content:   `syntax error at or near "user"`,
					},
				},
			},
			{
				statement: statements[1],
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingTableConventionMismatch.Int(),
						Title:     "naming.table",
						Content:   `"userTable" mismatches table naming convention, naming format should be "^[a-z]+(_[a-z]+)*$"`,
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingColumnConventionMismatch.Int(),
						Title:     "naming.column",
						Content:   `"userTable"."roomId" mismatches column naming convention, naming format should be "^[a-z]+(_[a-z]+)*$"`,
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingUKConventionMismatch.Int(),
						Title:     "naming.index.uk",
						Content:   `Unique key in table "userTable" mismatches the naming convention, expect "^uk_userTable_id_name$" but found "uk1"`,
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingFKConventionMismatch.Int(),
						Title:     "naming.index.fk",
						Content:   `Foreign key in table "userTable" mismatches the naming convention, expect "^fk_userTable_roomId_room_id$" but found "fk1"`,
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.TableNoPK.Int(),
						Title:     "table.require-pk",
						Content:   fmt.Sprintf(`Table "public"."userTable" requires PRIMARY KEY, related statement: %q`, statements[1]),
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoRequiredColumn.Int(),
						Title:     "column.required",
						Content:   `Table "userTable" requires columns: created_ts, creator_id, updated_ts, updater_id`,
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCanNotNull.Int(),
						Title:     "column.no-null",
						Content:   `Column "id" in "public"."userTable" can not have NULL value`,
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCanNotNull.Int(),
						Title:     "column.no-null",
						Content:   `Column "name" in "public"."userTable" can not have NULL value`,
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCanNotNull.Int(),
						Title:     "column.no-null",
						Content:   `Column "roomId" in "public"."userTable" can not have NULL value`,
					},
				},
			},
			{
				statement: "DELETE FROM t",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementNoWhere.Int(),
						Title:     "statement.where.require",
						Content:   `"DELETE FROM t" requires WHERE clause`,
					},
				},
			},
			{
				statement: "DELETE FROM t WHERE a like '%abc'",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementLeadingWildcardLike.Int(),
						Title:     "statement.where.no-leading-wildcard-like",
						Content:   `"DELETE FROM t WHERE a like '%abc'" uses leading wildcard LIKE`,
					},
				},
			},
			{
				statement: `DELETE FROM t WHERE a = (SELECT max(id) FROM "user" WHERE name = 'bytebase')`,
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
			},
		}
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	port := getTestPort(t.Name()) + 3
	err := ctl.StartServer(ctx, dataDir, fake.NewGitLab, getTestPort(t.Name()))
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)

	// Create a PostgreSQL instance.
	_, stopInstance := postgres.SetupTestInstance(t, port)
	defer stopInstance()

	pgDB, err := sql.Open("pgx", fmt.Sprintf("host=127.0.0.1 port=%d user=root database=postgres", port))
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

	policyPayload, err := prodTemplateSQLReviewPolicy()
	a.NoError(err)

	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
		Payload:       &policyPayload,
	})
	a.EqualError(err, sqlReviewAccessErr)

	err = ctl.setLicense()
	a.NoError(err)

	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
		Payload:       &policyPayload,
	})
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "pgInstance",
		Engine:        db.Postgres,
		Host:          "127.0.0.1",
		Port:          strconv.Itoa(port),
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

	err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

	for _, t := range tests {
		result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, project.Creator.ID, t.statement)
		a.Equal(t.result, result)
	}

	// disable the SQL review policy
	disable := string(api.Archived)
	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
		Payload:       &policyPayload,
		RowStatus:     &disable,
	})
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, project.Creator.ID, statements[0])
	a.Equal(noSQLReviewPolicy, result)

	// delete the SQL review policy
	err = ctl.deletePolicy(api.PolicyDelete{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
	})
	a.NoError(err)

	result = createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, project.Creator.ID, statements[0])
	a.Equal(noSQLReviewPolicy, result)
}

func TestSQLReviewForMySQL(t *testing.T) {
	var (
		databaseName = "testSQLReview"
		statements   = []string{
			"CREATE TABLE user(" +
				"id INT PRIMARY KEY," +
				"name VARCHAR(255) NOT NULL," +
				"room_id INT NOT NULL," +
				"creator_id INT NOT NULL," +
				"created_ts TIMESTAMP NOT NULL," +
				"updater_id INT NOT NULL," +
				"updated_ts TIMESTAMP NOT NULL," +
				"INDEX idx_user_name(name)," +
				"UNIQUE KEY uk_user_id_name(id, name)" +
				") ENGINE = INNODB",
			"CREATE TABLE userTable(" +
				"id INT," +
				"name VARCHAR(255)," +
				"roomId INT," +
				"INDEX idx1(name)," +
				"UNIQUE KEY uk1(id, name)," +
				"FOREIGN KEY fk1(roomId) REFERENCES room(id)" +
				") ENGINE = CSV",
		}
		tests = []test{
			{
				statement: statements[0],
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
			},
			{
				statement: "CREATE TABLE user(id);",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementSyntaxError.Int(),
						Title:     advisor.SyntaxErrorTitle,
						Content:   "line 1 column 21 near \");\" ",
					},
				},
			},
			{
				statement: statements[1],
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NotInnoDBEngine.Int(),
						Title:     "engine.mysql.use-innodb",
						Content:   fmt.Sprintf("%q doesn't use InnoDB engine", statements[1]),
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingTableConventionMismatch.Int(),
						Title:     "naming.table",
						Content:   "`userTable` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingColumnConventionMismatch.Int(),
						Title:     "naming.column",
						Content:   "`userTable`.`roomId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingIndexConventionMismatch.Int(),
						Title:     "naming.index.idx",
						Content:   "Index in table `userTable` mismatches the naming convention, expect \"^idx_userTable_name$\" but found `idx1`",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingUKConventionMismatch.Int(),
						Title:     "naming.index.uk",
						Content:   "Unique key in table `userTable` mismatches the naming convention, expect \"^uk_userTable_id_name$\" but found `uk1`",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingFKConventionMismatch.Int(),
						Title:     "naming.index.fk",
						Content:   "Foreign key in table `userTable` mismatches the naming convention, expect \"^fk_userTable_roomId_room_id$\" but found `fk1`",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.TableNoPK.Int(),
						Title:     "table.require-pk",
						Content:   "Table `userTable` requires PRIMARY KEY",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.TableHasFK.Int(),
						Title:     "table.no-foreign-key",
						Content:   "Foreign key is not allowed in the table `userTable`",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoRequiredColumn.Int(),
						Title:     "column.required",
						Content:   "Table `userTable` requires columns: created_ts, creator_id, updated_ts, updater_id",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCanNotNull.Int(),
						Title:     "column.no-null",
						Content:   "`userTable`.`id` can not have NULL value",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCanNotNull.Int(),
						Title:     "column.no-null",
						Content:   "`userTable`.`name` can not have NULL value",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCanNotNull.Int(),
						Title:     "column.no-null",
						Content:   "`userTable`.`roomId` can not have NULL value",
					},
				},
			},
			{
				statement: "DELETE FROM t",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementNoWhere.Int(),
						Title:     "statement.where.require",
						Content:   "\"DELETE FROM t\" requires WHERE clause",
					},
				},
			},
			{
				statement: "DELETE FROM t WHERE a like `%abc`",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementLeadingWildcardLike.Int(),
						Title:     "statement.where.no-leading-wildcard-like",
						Content:   "\"DELETE FROM t WHERE a like `%abc`\" uses leading wildcard LIKE",
					},
				},
			},
			{
				statement: "INSERT INTO t_copy SELECT * FROM t",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementSelectAll.Int(),
						Title:     "statement.select.no-select-all",
						Content:   "\"INSERT INTO t_copy SELECT * FROM t\" uses SELECT all",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementNoWhere.Int(),
						Title:     "statement.where.require",
						Content:   "\"INSERT INTO t_copy SELECT * FROM t\" requires WHERE clause",
					},
				},
			},
			{
				statement: "DELETE FROM t WHERE a = (SELECT max(id) FROM user WHERE name = 'bytebase')",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
			},
			{
				statement: "DROP TABLE user",
				result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.TableDropNamingConventionMismatch.Int(),
						Title:     "table.drop-naming-convention",
						Content:   "`user` mismatches drop table naming convention, naming format should be \"_delete$\"",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.CompatibilityDropTable.Int(),
						Title:     "schema.backward-compatibility",
						Content:   "\"DROP TABLE user\" may cause incompatibility with the existing data and code",
					},
				},
			},
		}
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	port := getTestPort(t.Name()) + 3
	err := ctl.StartServer(ctx, dataDir, fake.NewGitLab, getTestPort(t.Name()))
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)

	// Create a MySQL instance.
	_, stopInstance := mysql.SetupTestInstance(t, port)
	defer stopInstance()

	mysqlDB, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", port))
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

	policyPayload, err := prodTemplateSQLReviewPolicy()
	a.NoError(err)

	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
		Payload:       &policyPayload,
	})
	a.EqualError(err, sqlReviewAccessErr)

	err = ctl.setLicense()
	a.NoError(err)

	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
		Payload:       &policyPayload,
	})
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "mysqlInstance",
		Engine:        db.MySQL,
		Host:          "127.0.0.1",
		Port:          strconv.Itoa(port),
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

	for _, t := range tests {
		result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, project.Creator.ID, t.statement)
		a.Equal(t.result, result)
	}

	// disable the SQL review policy
	disable := string(api.Archived)
	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
		Payload:       &policyPayload,
		RowStatus:     &disable,
	})
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, project.Creator.ID, statements[0])
	a.Equal(noSQLReviewPolicy, result)

	// delete the SQL review policy
	err = ctl.deletePolicy(api.PolicyDelete{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSQLReview,
	})
	a.NoError(err)

	result = createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, project.Creator.ID, statements[0])
	a.Equal(noSQLReviewPolicy, result)
}

func createIssueAndReturnSQLReviewResult(a *require.Assertions, ctl *controller, databaseID int, projectID int, assigneeID int, statement string) []api.TaskCheckResult {
	createContext, err := json.Marshal(&api.UpdateSchemaContext{
		MigrationType: db.Migrate,
		DetailList: []*api.UpdateSchemaDetail{
			{
				DatabaseID: databaseID,
				Statement:  statement,
			},
		},
	})
	a.NoError(err)

	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectID,
		Name:          "update schema for database",
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   "This updates the schema of database",
		AssigneeID:    assigneeID,
		CreateContext: string(createContext),
	})
	a.NoError(err)

	result, err := ctl.GetSQLReviewResult(issue.ID)
	a.NoError(err)

	return result
}
