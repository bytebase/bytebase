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

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/stretchr/testify/require"
)

func TestSchemaSystem(t *testing.T) {
	type test struct {
		statement string
		result    []api.TaskCheckResult
	}

	var (
		databaseName          = "testSchemaSystem"
		schemaReviewAccessErr = fmt.Sprintf(`http response error code %d body "{\"message\":\"%s\"}\n"`, http.StatusForbidden, api.FeatureSchemaReviewPolicy.AccessErrorMessage())
		statements            = []string{
			"CREATE TABLE user(" +
				"id INT PRIMARY KEY," +
				"name VARCHAR(255) NOT NULL," +
				"room_id INT NOT NULL," +
				"creator_id INT NOT NULL," +
				"created_ts TIMESTAMP NOT NULL," +
				"updater_id INT NOT NULL," +
				"updated_ts TIMESTAMP NOT NULL," +
				"INDEX idx_user_name(name)," +
				"UNIQUE KEY uk_user_id_name(id, name)," +
				"FOREIGN KEY fk_user_room_id_room_id(room_id) REFERENCES room(id)" +
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
						Status:  api.TaskCheckStatusSuccess,
						Code:    common.Ok,
						Title:   "OK",
						Content: "",
					},
				},
			},
			{
				statement: "CREATE TABLE user(id);",
				result: []api.TaskCheckResult{
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.DbStatementSyntaxError,
						Title:   advisor.SyntaxErrorTitle,
						Content: "line 1 column 21 near \");\" ",
					},
				},
			},
			{
				statement: statements[1],
				result: []api.TaskCheckResult{
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.NotInnoDBEngine,
						Title:   "InnoDB engine is not used",
						Content: fmt.Sprintf("%q doesn't use InnoDB engine", statements[1]),
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.NamingTableConventionMismatch,
						Title:   "Mismatch table naming convention",
						Content: "`userTable` mismatches table naming convention, naming format should be \"^[a-z]+(_[a-z]+)?$\"",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.NamingColumnConventionMismatch,
						Title:   "Mismatch column naming convention",
						Content: "`userTable`.`roomId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)?$\"",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.NamingIndexConventionMismatch,
						Title:   "Mismatch index naming convention",
						Content: "Index in table `userTable` mismatches the naming convention, expect \"^idx_userTable_name$\" but found `idx1`",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.NamingUKConventionMismatch,
						Title:   "Mismatch unique key naming convention",
						Content: "Unique key in table `userTable` mismatches the naming convention, expect \"^uk_userTable_id_name$\" but found `uk1`",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.NamingFKConventionMismatch,
						Title:   "Mismatch foreign key naming convention",
						Content: "Foreign key in table `userTable` mismatches the naming convention, expect \"^fk_userTable_roomId_room_id$\" but found `fk1`",
					},
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.TableNoPK,
						Title:   "Require PRIMARY KEY",
						Content: "Table `userTable` requires PRIMARY KEY",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.NoRequiredColumn,
						Title:   "Require columns",
						Content: "Table `userTable` requires columns: created_ts, creator_id, updated_ts, updater_id",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.ColumnCanNull,
						Title:   "Column can have NULL value",
						Content: "`userTable`.`id` can have NULL value",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.ColumnCanNull,
						Title:   "Column can have NULL value",
						Content: "`userTable`.`name` can have NULL value",
					},
					{
						Status:  api.TaskCheckStatusWarn,
						Code:    common.ColumnCanNull,
						Title:   "Column can have NULL value",
						Content: "`userTable`.`roomId` can have NULL value",
					},
				},
			},
			{
				statement: "DELETE FROM t",
				result: []api.TaskCheckResult{
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.StatementNoWhere,
						Title:   "Require WHERE clause",
						Content: "\"DELETE FROM t\" requires WHERE clause",
					},
				},
			},
			{
				statement: "DELETE FROM t WHERE a like `%abc`",
				result: []api.TaskCheckResult{
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.StatementLeadingWildcardLike,
						Title:   "No leading wildcard LIKE",
						Content: "\"DELETE FROM t WHERE a like `%abc`\" uses leading wildcard LIKE",
					},
				},
			},
			{
				// TODO(rebelice): most advisors cannot get the SQL text with sub-queries, fix it
				statement: "INSERT INTO t_copy SELECT * FROM t",
				result: []api.TaskCheckResult{
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.StatementSelectAll,
						Title:   "No SELECT all",
						Content: "\"\" uses SELECT all",
					},
					{
						Status:  api.TaskCheckStatusError,
						Code:    common.StatementNoWhere,
						Title:   "Require WHERE clause",
						Content: "\"\" requires WHERE clause",
					},
				},
			},
			{
				statement: "DELETE FROM t WHERE a = (SELECT max(id) FROM user WHERE name = 'bytebase')",
				result: []api.TaskCheckResult{
					{
						Status:  api.TaskCheckStatusSuccess,
						Code:    common.Ok,
						Title:   "OK",
						Content: "",
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
	err := ctl.StartServer(ctx, dataDir, getTestPort(t.Name()))
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
		Name: "Test Schema System Project",
		Key:  "TestSchemaSystem",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	policyPayload, err := api.ProdTemplateSchemaReviewPolicy()
	a.NoError(err)

	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSchemaReview,
		Payload:       &policyPayload,
	})
	a.EqualError(err, schemaReviewAccessErr)

	err = ctl.setLicense()
	a.NoError(err)

	err = ctl.upsertPolicy(api.PolicyUpsert{
		EnvironmentID: prodEnvironment.ID,
		Type:          api.PolicyTypeSchemaReview,
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

	err = ctl.createDatabase(project, instance, databaseName, nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(database.Instance.ID, instance.ID)

	for _, t := range tests {
		createContext, err := json.Marshal(&api.UpdateSchemaContext{
			MigrationType: db.Migrate,
			DetailList: []*api.UpdateSchemaDetail{
				{
					DatabaseID: database.ID,
					Statement:  t.statement,
				},
			},
		})
		a.NoError(err)

		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("update schema for database %q", databaseName),
			Type:          api.IssueDatabaseSchemaUpdate,
			Description:   fmt.Sprintf("This updates the schema of database %q", databaseName),
			AssigneeID:    project.Creator.ID,
			CreateContext: string(createContext),
		})
		a.NoError(err)

		result, err := ctl.getSchemaReviewResult(issue.ID)
		a.NoError(err)

		a.Equal(t.result, result)
	}

}
