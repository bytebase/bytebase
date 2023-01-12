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
	"github.com/bytebase/bytebase/plugin/advisor"
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

	yamlFile.Close()
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
	var (
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
		tests = []test{
			{
				Statement: statements[0],
				Result: []api.TaskCheckResult{
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
				Statement: "CREATE TABLE user(id);",
				Result: []api.TaskCheckResult{
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
				Statement: statements[1],
				Result: []api.TaskCheckResult{
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
						Content:   "Index in table `userTable` mismatches the naming convention, expect \"^$|^idx_userTable_name$\" but found `idx1`",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingUKConventionMismatch.Int(),
						Title:     "naming.index.uk",
						Content:   "Unique key in table `userTable` mismatches the naming convention, expect \"^$|^uk_userTable_id_name$\" but found `uk1`",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingFKConventionMismatch.Int(),
						Title:     "naming.index.fk",
						Content:   "Foreign key in table `userTable` mismatches the naming convention, expect \"^$|^fk_userTable_roomId_room_id$\" but found `fk1`",
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
						Code:      advisor.NoTableComment.Int(),
						Title:     "table.comment",
						Content:   "Table `userTable` requires comments",
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
						Code:      advisor.ColumnCannotNull.Int(),
						Title:     "column.no-null",
						Content:   "`userTable`.`name` cannot have NULL value",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ColumnCannotNull.Int(),
						Title:     "column.no-null",
						Content:   "`userTable`.`roomId` cannot have NULL value",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NotNullColumnWithNoDefault.Int(),
						Title:     "column.set-default-for-not-null",
						Content:   "Column `userTable`.`id` is NOT NULL but doesn't have DEFAULT",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoColumnComment.Int(),
						Title:     "column.comment",
						Content:   "Column `userTable`.`id` requires comments",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoColumnComment.Int(),
						Title:     "column.comment",
						Content:   "Column `userTable`.`name` requires comments",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoColumnComment.Int(),
						Title:     "column.comment",
						Content:   "Column `userTable`.`roomId` requires comments",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.DisabledColumnType.Int(),
						Title:     "column.type-disallow-list",
						Content:   "Disallow column type JSON but column `userTable`.`json_content` is",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.SetColumnCharset.Int(),
						Title:     "column.disallow-set-charset",
						Content:   fmt.Sprintf("Disallow set column charset but \"%s\" does", statements[1]),
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.OnUpdateCurrentTimeColumnCountExceedsLimit.Int(),
						Title:     "column.current-time-count-limit",
						Content:   "Table `userTable` has 2 ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than 1.",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoDefault.Int(),
						Title:     "column.require-default",
						Content:   "Column `userTable`.`id` doesn't have DEFAULT.",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoDefault.Int(),
						Title:     "column.require-default",
						Content:   "Column `userTable`.`name` doesn't have DEFAULT.",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoDefault.Int(),
						Title:     "column.require-default",
						Content:   "Column `userTable`.`roomId` doesn't have DEFAULT.",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.IndexTypeNoBlob.Int(),
						Title:     "index.type-no-blob",
						Content:   "Columns in index must not be BLOB but `userTable`.`content` is blob",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.DisabledCharset.Int(),
						Title:     "system.charset.allowlist",
						Content:   fmt.Sprintf("\"%s\" used disabled charset 'ascii'", statements[1]),
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.DisabledCollation.Int(),
						Title:     "system.collation.allowlist",
						Content:   fmt.Sprintf("\"%s\" used disabled collation 'latin1_bin'", statements[1]),
					},
				},
			},
			{
				Statement: `CREATE TABLE t_auto(auto_id varchar(20) AUTO_INCREMENT PRIMARY KEY COMMENT 'COMMENT') auto_increment = 2 COMMENT 'comment'`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NamingAutoIncrementColumnConventionMismatch.Int(),
						Title:     "naming.column.auto-increment",
						Content:   "`t_auto`.`auto_id` mismatches auto_increment column naming convention, naming format should be \"^id$\"",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.NoRequiredColumn.Int(),
						Title:     "column.required",
						Content:   "Table `t_auto` requires columns: created_ts, creator_id, id, updated_ts, updater_id",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.AutoIncrementColumnNotInteger.Int(),
						Title:     "column.auto-increment-must-integer",
						Content:   "Auto-increment column `t_auto`.`auto_id` requires integer type",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.AutoIncrementColumnInitialValueNotMatch.Int(),
						Title:     "column.auto-increment-initial-value",
						Content:   "The initial auto-increment value in table `t_auto` is 2, which doesn't equal 20",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.AutoIncrementColumnSigned.Int(),
						Title:     "column.auto-increment-must-unsigned",
						Content:   "Auto-increment column `t_auto`.`auto_id` is not UNSIGNED type",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.IndexPKType.Int(),
						Title:     "index.pk-type-limit",
						Content:   "Columns in primary key must be INT/BIGINT but `t_auto`.`auto_id` is varchar(20)",
					},
				},
			},
			{
				Statement: `
					DELETE FROM tech_book`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementNoWhere.Int(),
						Title:     "statement.where.require",
						Content:   "\"DELETE FROM tech_book\" requires WHERE clause",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementAffectedRowExceedsLimit.Int(),
						Title:     "statement.affected-row-limit",
						Content:   "\"DELETE FROM tech_book\" dry runs failed: Error 1146: Table 'testsqlreview.tech_book' doesn't exist",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementDMLDryRunFailed.Int(),
						Title:     "statement.dml-dry-run",
						Content:   "\"DELETE FROM tech_book\" dry runs failed: Error 1146: Table 'testsqlreview.tech_book' doesn't exist",
					},
				},
			},
			{
				Statement: "DELETE FROM tech_book WHERE name like `%abc`",
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementLeadingWildcardLike.Int(),
						Title:     "statement.where.no-leading-wildcard-like",
						Content:   "\"DELETE FROM tech_book WHERE name like `%abc`\" uses leading wildcard LIKE",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementAffectedRowExceedsLimit.Int(),
						Title:     "statement.affected-row-limit",
						Content:   "\"DELETE FROM tech_book WHERE name like `%abc`\" dry runs failed: Error 1146: Table 'testsqlreview.tech_book' doesn't exist",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementDMLDryRunFailed.Int(),
						Title:     "statement.dml-dry-run",
						Content:   "\"DELETE FROM tech_book WHERE name like `%abc`\" dry runs failed: Error 1146: Table 'testsqlreview.tech_book' doesn't exist",
					},
				},
			},
			{
				Statement: `
					INSERT INTO t_copy SELECT * FROM t`,
				Result: []api.TaskCheckResult{
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

					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.InsertTooManyRows.Int(),
						Title:     "statement.insert.row-limit",
						Content:   "\"INSERT INTO t_copy SELECT * FROM t\" dry runs failed: Error 1146: Table 'testsqlreview.t_copy' doesn't exist",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.InsertNotSpecifyColumn.Int(),
						Title:     "statement.insert.must-specify-column",
						Content:   "The INSERT statement must specify columns but \"INSERT INTO t_copy SELECT * FROM t\" does not",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementDMLDryRunFailed.Int(),
						Title:     "statement.dml-dry-run",
						Content:   "\"INSERT INTO t_copy SELECT * FROM t\" dry runs failed: Error 1146: Table 'testsqlreview.t_copy' doesn't exist",
					},
				},
			},
			{
				Statement: `
					INSERT INTO t VALUES (1, 1, now(), 1, now())`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.InsertNotSpecifyColumn.Int(),
						Title:     "statement.insert.must-specify-column",
						Content:   "The INSERT statement must specify columns but \"INSERT INTO t VALUES (1, 1, now(), 1, now())\" does not",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementDMLDryRunFailed.Int(),
						Title:     "statement.dml-dry-run",
						Content:   "\"INSERT INTO t VALUES (1, 1, now(), 1, now())\" dry runs failed: Error 1146: Table 'testsqlreview.t' doesn't exist",
					},
				},
			},
			{
				Statement: "DELETE FROM tech_book WHERE id = (SELECT max(id) FROM tech_book WHERE name = 'bytebase')",
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementAffectedRowExceedsLimit.Int(),
						Title:     "statement.affected-row-limit",
						Content:   "\"DELETE FROM tech_book WHERE id = (SELECT max(id) FROM tech_book WHERE name = 'bytebase')\" dry runs failed: Error 1146: Table 'testsqlreview.tech_book' doesn't exist",
					},
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementDMLDryRunFailed.Int(),
						Title:     "statement.dml-dry-run",
						Content:   "\"DELETE FROM tech_book WHERE id = (SELECT max(id) FROM tech_book WHERE name = 'bytebase')\" dry runs failed: Error 1146: Table 'testsqlreview.tech_book' doesn't exist",
					},
				},
			},
			{
				Statement: `COMMIT;`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusError,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementDisallowCommit.Int(),
						Title:     "statement.disallow-commit",
						Content:   "Commit is not allowed, related statement: \"COMMIT;\"",
					},
				},
			},
			{
				Statement: statements[0],
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
				Run: true,
			},
			{
				Statement: `INSERT INTO user(id, name) values (1, 'a'), (2, 'b'), (3, 'c'), (4, 'd'), (5, 'e')`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
				Run: true,
			},
			{
				Statement: `DELETE FROM user WHERE id < 10`,
				Result: []api.TaskCheckResult{
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
				Statement: `INSERT INTO user(id, name) values (6, 'f'), (7, 'g'), (8, 'h'), (9, 'i'), (10, 'j')`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusSuccess,
						Namespace: api.BBNamespace,
						Code:      common.Ok.Int(),
						Title:     "OK",
						Content:   "",
					},
				},
				Run: true,
			},
			{
				Statement: `DELETE FROM user WHERE id <= 10`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementAffectedRowExceedsLimit.Int(),
						Title:     "statement.affected-row-limit",
						Content:   "\"DELETE FROM user WHERE id <= 10\" affected 10 rows. The count exceeds 5.",
					},
				},
			},
			{
				Statement: `INSERT INTO user(id, name) SELECT id, name FROM ` + valueTable + ` WHERE 1=1`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.InsertTooManyRows.Int(),
						Title:     "statement.insert.row-limit",
						Content:   "\"INSERT INTO user(id, name) SELECT id, name FROM " + valueTable + " WHERE 1=1\" inserts 10 rows. The count exceeds 5.",
					},
				},
			},
			{
				Statement: "INSERT INTO user(id, name) SELECT id, name FROM user WHERE id=1 LIMIT 1",
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.InsertUseLimit.Int(),
						Title:     "statement.disallow-limit",
						Content:   "LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"INSERT INTO user(id, name) SELECT id, name FROM user WHERE id=1 LIMIT 1\" uses",
					},
				},
			},
			{
				Statement: `
					ALTER TABLE user PARTITION BY HASH(id) PARTITIONS 8;
				`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.CreateTablePartition.Int(),
						Title:     "table.disallow-partition",
						Content:   "Table partition is forbidden, but \"ALTER TABLE user PARTITION BY HASH(id) PARTITIONS 8;\" creates",
					},
				},
			},
			{
				Statement: `
					ALTER TABLE user CHANGE name name varchar(320) NOT NULL DEFAULT '' COMMENT 'COMMENT' FIRST;
					ALTER TABLE user ADD COLUMN c_column int NOT NULL DEFAULT 0 COMMENT 'comment';
				`,
				Result: []api.TaskCheckResult{
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.StatementRedundantAlterTable.Int(),
						Title:     "statement.merge-alter-table",
						Content:   "There are 2 statements to modify table `user`",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ChangeColumnType.Int(),
						Title:     "column.disallow-change-type",
						Content:   "\"ALTER TABLE user CHANGE name name varchar(320) NOT NULL DEFAULT '' COMMENT 'COMMENT' FIRST;\" changes column type",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.UseChangeColumnStatement.Int(),
						Title:     "column.disallow-change",
						Content:   "\"ALTER TABLE user CHANGE name name varchar(320) NOT NULL DEFAULT '' COMMENT 'COMMENT' FIRST;\" contains CHANGE COLUMN statement",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.ChangeColumnOrder.Int(),
						Title:     "column.disallow-changing-order",
						Content:   "\"ALTER TABLE user CHANGE name name varchar(320) NOT NULL DEFAULT '' COMMENT 'COMMENT' FIRST;\" changes column order",
					},
					{
						Status:    api.TaskCheckStatusWarn,
						Namespace: api.AdvisorNamespace,
						Code:      advisor.CompatibilityAlterColumn.Int(),
						Title:     "schema.backward-compatibility",
						Content:   "\"ALTER TABLE user CHANGE name name varchar(320) NOT NULL DEFAULT '' COMMENT 'COMMENT' FIRST;\" may cause incompatibility with the existing data and code",
					},
				},
			},
			{
				Statement: `
					DROP TABLE user;
					`,
				Result: []api.TaskCheckResult{
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
						Content:   "\"DROP TABLE user;\" may cause incompatibility with the existing data and code",
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

	for _, t := range tests {
		result := createIssueAndReturnSQLReviewResult(a, ctl, database.ID, project.ID, t.Statement, t.Run)
		a.Equal(t.Result, result, t.Statement)
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
		a.Equal(common.Ok.Int(), result[0].Code)
		status, err := ctl.waitIssuePipeline(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)
	}

	return result
}
