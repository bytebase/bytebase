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
	"strings"
	"testing"

	// Import pg driver.
	// init() in pgx will register it's pgx driver.
	"github.com/google/go-cmp/cmp"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	noSQLReviewPolicy = []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "",
		},
	}
)

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
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	stopInstance := postgres.SetupTestInstance(t, pgPort, resourceDir)
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
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_DEPLOYMENT_APPROVAL,
			Policy: &v1pb.Policy_DeploymentApprovalPolicy{
				DeploymentApprovalPolicy: &v1pb.DeploymentApprovalPolicy{
					DefaultStrategy: v1pb.ApprovalStrategy_MANUAL,
				},
			},
		},
	})
	a.NoError(err)

	reviewPolicy, err := prodTemplateSQLReviewPolicyForPostgreSQL()
	a.NoError(err)

	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_SQL_REVIEW,
			Policy: &v1pb.Policy_SqlReviewPolicy{
				SqlReviewPolicy: reviewPolicy,
			},
		},
	})
	a.NoError(err)

	policy, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_SQL_REVIEW,
			Policy: &v1pb.Policy_SqlReviewPolicy{
				SqlReviewPolicy: reviewPolicy,
			},
		},
	})
	a.NoError(err)
	a.NotNil(policy.Name)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "bytebase", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []test{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	err = yaml.Unmarshal(byteValue, &tests)
	a.NoError(err)

	for i, t := range tests {
		result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, t.Statement, t.Run)
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
	policy.Enforce = false
	_, err = ctl.orgPolicyServiceClient.UpdatePolicy(ctx, &v1pb.UpdatePolicyRequest{
		Policy: policy,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"enforce"},
		},
	})
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, statements[0], false)
	a.Equal(noSQLReviewPolicy, result)

	// delete the SQL review policy
	_, err = ctl.orgPolicyServiceClient.DeletePolicy(ctx, &v1pb.DeletePolicyRequest{
		Name: fmt.Sprintf("%s/policies/%s", prodEnvironment.Name, v1pb.PolicyType_SQL_REVIEW),
	})
	a.NoError(err)

	result = createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, statements[0], false)
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
		wantQueryResult = &v1pb.QueryResult{
			ColumnNames:     []string{"count(*)"},
			ColumnTypeNames: []string{"BIGINT"},
			Masked:          []bool{false},
			Sensitive:       []bool{false},
			Rows: []*v1pb.QueryRow{
				{
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "4"}},
					},
				},
			},
			Statement: "SELECT count(*) FROM test WHERE 1=1",
		}
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
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
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_DEPLOYMENT_APPROVAL,
			Policy: &v1pb.Policy_DeploymentApprovalPolicy{
				DeploymentApprovalPolicy: &v1pb.DeploymentApprovalPolicy{
					DefaultStrategy: v1pb.ApprovalStrategy_MANUAL,
				},
			},
		},
	})
	a.NoError(err)

	reviewPolicy, err := prodTemplateSQLReviewPolicyForMySQL()
	a.NoError(err)

	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_SQL_REVIEW,
			Policy: &v1pb.Policy_SqlReviewPolicy{
				SqlReviewPolicy: reviewPolicy,
			},
		},
	})
	a.NoError(err)

	policy, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: prodEnvironment.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_SQL_REVIEW,
			Policy: &v1pb.Policy_SqlReviewPolicy{
				SqlReviewPolicy: reviewPolicy,
			},
		},
	})
	a.NoError(err)
	a.NotNil(policy.Name)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(mysqlPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)
	databaseUID, err := strconv.Atoi(database.Uid)
	a.NoError(err)

	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []test{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	err = yaml.Unmarshal(byteValue, &tests)
	a.NoError(err)

	for i, t := range tests {
		result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, t.Statement, t.Run)
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
		createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, stmt, true /* wait */)
	}
	countSQL := "SELECT count(*) FROM test WHERE 1=1;"
	dmlSQL := "INSERT INTO test SELECT * FROM " + valueTable
	originQueryResp, err := ctl.sqlServiceClient.Query(ctx, &v1pb.QueryRequest{
		Name: instance.Name, ConnectionDatabase: databaseName, Statement: countSQL,
	})
	a.NoError(err)
	a.Equal(1, len(originQueryResp.Results))
	diff := cmp.Diff(wantQueryResult, originQueryResp.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Equal("", diff)

	createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, dmlSQL, false /* wait */)

	finalQueryResp, err := ctl.sqlServiceClient.Query(ctx, &v1pb.QueryRequest{
		Name: instance.Name, ConnectionDatabase: databaseName, Statement: countSQL,
	})
	a.NoError(err)
	a.Equal(1, len(finalQueryResp.Results))
	diff = cmp.Diff(wantQueryResult, finalQueryResp.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Equal("", diff)

	// disable the SQL review policy
	policy.Enforce = false
	_, err = ctl.orgPolicyServiceClient.UpdatePolicy(ctx, &v1pb.UpdatePolicyRequest{
		Policy: policy,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"enforce"},
		},
	})
	a.NoError(err)

	// delete the SQL review policy
	_, err = ctl.orgPolicyServiceClient.DeletePolicy(ctx, &v1pb.DeletePolicyRequest{
		Name: fmt.Sprintf("%s/policies/%s", prodEnvironment.Name, v1pb.PolicyType_SQL_REVIEW),
	})
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, databaseUID, projectUID, project.Name, statements[0], false)
	a.Equal(noSQLReviewPolicy, result)
}

func createIssueAndReturnSQLReviewResult(ctx context.Context, a *require.Assertions, ctl *controller, databaseID int, projectID int, projectName, statement string, wait bool) []api.TaskCheckResult {
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: projectName,
		Sheet: &v1pb.Sheet{
			Title:      "statement",
			Content:    []byte(statement),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)
	sheetUID, err := strconv.Atoi(strings.TrimPrefix(sheet.Name, fmt.Sprintf("%s/sheets/", projectName)))
	a.NoError(err)

	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    databaseID,
				SheetID:       sheetUID,
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
		status, err := ctl.waitIssuePipeline(ctx, issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)
	}

	return result
}
