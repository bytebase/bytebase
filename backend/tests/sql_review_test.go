package tests

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	// Import pg driver.
	// init() in pgx will register it's pgx driver.
	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

var (
	noSQLReviewPolicy = []*v1pb.PlanCheckRun_Result{
		{
			Status: v1pb.Advice_SUCCESS,
			Title:  "OK",
		},
	}
)

type test struct {
	Statement string
	Result    []*v1pb.PlanCheckRun_Result
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
				);
				COMMENT ON TABLE "user" IS 'comment';`,
			`CREATE TABLE "userTable"(
				id INT,
				name VARCHAR(255),
				"roomId" INT,
				CONSTRAINT uk1 UNIQUE (id, name),
				CONSTRAINT fk1 FOREIGN KEY ("roomId") REFERENCES room(id)
				);
				COMMENT ON TABLE "userTable" IS 'comment';`,
		}
		databaseName = "testsqlreview"
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	tests, err := readTestData(filepath)
	a.NoError(err)
	ctx, err = ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	pgDB := pgContainer.db
	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)
	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)
	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	reviewConfig := prodTemplateReviewConfigForPostgreSQL()

	createdConfig, err := ctl.reviewConfigServiceClient.CreateReviewConfig(ctx, connect.NewRequest(&v1pb.CreateReviewConfigRequest{
		ReviewConfig: reviewConfig,
	}))
	a.NoError(err)
	a.NotNil(createdConfig.Msg.Name)

	policy, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: "environments/prod",
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_TAG,
			Policy: &v1pb.Policy_TagPolicy{
				TagPolicy: &v1pb.TagPolicy{
					Tags: map[string]string{
						common.ReservedTagReviewConfig: createdConfig.Msg.Name,
					},
				},
			},
		},
	}))
	a.NoError(err)
	a.NotNil(policy.Msg.Name)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	}))
	a.NoError(err)

	err = ctl.createDatabase(ctx, ctl.project, instance.Msg, nil /* environment */, databaseName, "bytebase")
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Msg.Name, databaseName),
	}))
	a.NoError(err)

	for i, t := range tests {
		result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, t.Statement, t.Run)
		if record {
			tests[i].Result = result
		} else {
			equalReviewResultProtos(a, t.Result, result, database.Msg.Name, t.Statement)
		}
	}

	if record {
		err := writeTestData(filepath, tests)
		a.NoError(err)
	}

	// disable the SQL review policy
	policy.Msg.Enforce = false
	_, err = ctl.orgPolicyServiceClient.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy: policy.Msg,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"enforce"},
		},
	}))
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, statements[0], false)
	equalReviewResultProtos(a, noSQLReviewPolicy, result, database.Msg.Name, "")

	// delete the SQL review policy
	_, err = ctl.orgPolicyServiceClient.DeletePolicy(ctx, connect.NewRequest(&v1pb.DeletePolicyRequest{
		Name: policy.Msg.Name,
	}))
	a.NoError(err)

	result = createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, statements[0], false)
	equalReviewResultProtos(a, noSQLReviewPolicy, result, database.Msg.Name, "")
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
			Masked:          []*v1pb.MaskingReason{nil},
			Rows: []*v1pb.QueryRow{
				{
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_Int64Value{Int64Value: 4}},
					},
				},
			},
			Statement: "SELECT count(*) FROM test WHERE 1=1;",
			RowsCount: 1,
		}
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	tests, err := readTestData(filepath)
	a.NoError(err)
	ctx, err = ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mysqlContainer, err := getMySQLContainer(ctx)
	defer func() {
		mysqlContainer.Close(ctx)
	}()
	a.NoError(err)

	mysqlDB := mysqlContainer.db
	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)
	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	reviewConfig := prodTemplateReviewConfigForMySQL()

	createdConfig, err := ctl.reviewConfigServiceClient.CreateReviewConfig(ctx, connect.NewRequest(&v1pb.CreateReviewConfigRequest{
		ReviewConfig: reviewConfig,
	}))
	a.NoError(err)
	a.NotNil(createdConfig.Msg.Name)

	policy, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: "environments/prod",
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_TAG,
			Policy: &v1pb.Policy_TagPolicy{
				TagPolicy: &v1pb.TagPolicy{
					Tags: map[string]string{
						common.ReservedTagReviewConfig: createdConfig.Msg.Name,
					},
				},
			},
		},
	}))
	a.NoError(err)
	a.NotNil(policy.Msg.Name)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: mysqlContainer.host, Port: mysqlContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	}))
	a.NoError(err)

	err = ctl.createDatabase(ctx, ctl.project, instance.Msg, nil /* environment */, databaseName, "")
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Msg.Name, databaseName),
	}))
	a.NoError(err)

	for i, t := range tests {
		result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, t.Statement, t.Run)
		if record {
			tests[i].Result = result
		} else {
			equalReviewResultProtos(a, t.Result, result, database.Msg.Name, tests[i].Statement)
		}
	}

	if record {
		err := writeTestData(filepath, tests)
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
		createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, stmt, true /* wait */)
	}
	countSQL := "SELECT count(*) FROM test WHERE 1=1;"
	dmlSQL := "INSERT INTO test SELECT * FROM " + valueTable
	originQueryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:         database.Msg.Name,
		Statement:    countSQL,
		DataSourceId: "admin",
	}))
	a.NoError(err)
	a.Equal(1, len(originQueryResp.Msg.Results))
	diff := cmp.Diff(wantQueryResult, originQueryResp.Msg.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Empty(diff)

	createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, dmlSQL, false /* wait */)

	finalQueryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:         database.Msg.Name,
		Statement:    countSQL,
		DataSourceId: "admin",
	}))
	a.NoError(err)
	a.Equal(1, len(finalQueryResp.Msg.Results))
	diff = cmp.Diff(wantQueryResult, finalQueryResp.Msg.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Empty(diff)

	// disable the SQL review policy
	policy.Msg.Enforce = false
	_, err = ctl.orgPolicyServiceClient.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy: policy.Msg,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"enforce"},
		},
	}))
	a.NoError(err)

	// delete the SQL review policy
	_, err = ctl.orgPolicyServiceClient.DeletePolicy(ctx, connect.NewRequest(&v1pb.DeletePolicyRequest{
		Name: policy.Msg.Name,
	}))
	a.NoError(err)

	result := createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, statements[0], false)
	equalReviewResultProtos(a, noSQLReviewPolicy, result, database.Msg.Name, "")
}

func readTestData(path string) ([]test, error) {
	yamlFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()
	byteValue, err := io.ReadAll(yamlFile)
	if err != nil {
		return nil, err
	}
	type yamlStruct struct {
		Statement string
		Result    []string
		Run       bool
	}
	var yamlTests []yamlStruct
	if err := yaml.Unmarshal(byteValue, &yamlTests); err != nil {
		return nil, err
	}

	var tests []test
	for _, yamlTest := range yamlTests {
		t := test{
			Statement: yamlTest.Statement,
			Run:       yamlTest.Run,
		}
		for _, r := range yamlTest.Result {
			result := &v1pb.PlanCheckRun_Result{}
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(r), result); err != nil {
				return nil, err
			}
			t.Result = append(t.Result, result)
		}
		tests = append(tests, t)
	}
	return tests, nil
}

func writeTestData(filepath string, tests []test) error {
	type yamlStruct struct {
		Statement string
		Result    []string
		Run       bool
	}

	var yamlTests []yamlStruct
	for _, t := range tests {
		yamlTest := yamlStruct{
			Statement: t.Statement,
			Run:       t.Run,
		}
		for _, r := range t.Result {
			yamlTest.Result = append(yamlTest.Result, protojson.Format(r))
		}
		yamlTests = append(yamlTests, yamlTest)
	}

	byteValue, err := yaml.Marshal(yamlTests)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath, byteValue, 0644)
	if err != nil {
		return err
	}
	return nil
}

func createIssueAndReturnSQLReviewResult(ctx context.Context, a *require.Assertions, ctl *controller, project *v1pb.Project, database *v1pb.Database, statement string, wait bool) []*v1pb.PlanCheckRun_Result {
	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(statement),
		},
	}))
	a.NoError(err)

	plan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets:     []string{database.Name},
							Sheet:       sheet.Msg.Name,
							EnableGhost: false,
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	result, err := ctl.GetSQLReviewResult(ctx, plan.Msg)
	a.NoError(err)

	var statementAdviseResults []*v1pb.PlanCheckRun_Result
	for _, r := range result.Results {
		if r.Type == v1pb.PlanCheckRun_Result_STATEMENT_ADVISE {
			statementAdviseResults = append(statementAdviseResults, r)
		}
	}

	if wait {
		a.NotNil(result)
		a.Len(statementAdviseResults, 1)
		a.Equal(v1pb.Advice_SUCCESS, statementAdviseResults[0].Status)
		issue, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: project.Name,
			Issue: &v1pb.Issue{
				Type:        v1pb.Issue_DATABASE_CHANGE,
				Title:       fmt.Sprintf("change database %s", database.Name),
				Description: fmt.Sprintf("change database %s", database.Name),
				Plan:        plan.Msg.Name,
			},
		}))
		a.NoError(err)
		rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: plan.Msg.Name}))
		a.NoError(err)
		err = ctl.waitRollout(ctx, issue.Msg.Name, rollout.Msg.Name)
		a.NoError(err)
		// Wait some time till written data becomes consistent.
		time.Sleep(5 * time.Second)
	}

	return statementAdviseResults
}

func equalReviewResultProtos(a *require.Assertions, want, got []*v1pb.PlanCheckRun_Result, expectedTarget, message string) {
	a.Equal(len(want), len(got), message)
	for i := 0; i < len(want); i++ {
		// Verify target matches expected database
		a.Equal(expectedTarget, got[i].Target, message)
		// Verify type is STATEMENT_ADVISE (we filter for this type)
		a.Equal(v1pb.PlanCheckRun_Result_STATEMENT_ADVISE, got[i].Type, message)
		// Compare other fields, ignoring target and type since we checked them above
		diff := cmp.Diff(want[i], got[i], protocmp.Transform(),
			protocmp.IgnoreFields(&v1pb.PlanCheckRun_Result{}, "target", "type"))
		a.Empty(diff, message)
	}
}

func prodTemplateReviewConfigForPostgreSQL() *v1pb.ReviewConfig {
	config := &v1pb.ReviewConfig{
		Name:    common.FormatReviewConfig(generateRandomString("review")),
		Title:   "Prod",
		Enabled: true,
		Rules: []*v1pb.SQLReviewRule{
			// Naming
			{
				Type:   v1pb.SQLReviewRule_NAMING_TABLE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^[a-z]+(_[a-z]+)*$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_COLUMN,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^[a-z]+(_[a-z]+)*$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_IDX,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^idx_{{table}}_{{column_list}}$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_PK,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^pk_{{table}}_{{column_list}}$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_UK,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^uk_{{table}}_{{column_list}}$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_FK,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
						MaxLength: 64,
					},
				},
			},
			// Statement
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// TABLE
			{
				Type:   v1pb.SQLReviewRule_TABLE_REQUIRE_PK,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format: "_delete$",
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_COMMENT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_CommentConventionPayload{
					CommentConventionPayload: &v1pb.SQLReviewRule_CommentConventionRulePayload{
						Required:  true,
						MaxLength: 10,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// COLUMN
			{
				Type:   v1pb.SQLReviewRule_COLUMN_REQUIRED,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{
							"id",
							"created_ts",
							"updated_ts",
							"creator_id",
							"updater_id",
						},
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_NO_NULL,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"JSON", "BINARY_FLOAT"},
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 20,
					},
				},
			},
			// SCHEMA
			{
				Type:   v1pb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// INDEX
			{
				Type:   v1pb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   v1pb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 5,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 5,
					},
				},
			},
			// SYSTEM
			{
				Type:   v1pb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"utf8mb4", "UTF8"},
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_POSTGRES,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"utf8mb4_0900_ai_ci"},
					},
				},
			},
		},
	}

	return config
}

func prodTemplateReviewConfigForMySQL() *v1pb.ReviewConfig {
	config := &v1pb.ReviewConfig{
		Name:    common.FormatReviewConfig(generateRandomString("review")),
		Title:   "Prod",
		Enabled: true,
		Rules: []*v1pb.SQLReviewRule{
			// Engine
			{
				Type:   v1pb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			// Naming
			{
				Type:   v1pb.SQLReviewRule_NAMING_TABLE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^[a-z]+(_[a-z]+)*$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_COLUMN,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^[a-z]+(_[a-z]+)*$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_IDX,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^idx_{{table}}_{{column_list}}$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_UK,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^uk_{{table}}_{{column_list}}$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_INDEX_FK,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
						MaxLength: 64,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format:    "^id$",
						MaxLength: 64,
					},
				},
			},
			// Statement
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 5,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 5,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			// TABLE
			{
				Type:   v1pb.SQLReviewRule_TABLE_REQUIRE_PK,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NamingPayload{
					NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
						Format: "_delete$",
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_COMMENT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_CommentConventionPayload{
					CommentConventionPayload: &v1pb.SQLReviewRule_CommentConventionRulePayload{
						Required:  true,
						MaxLength: 10,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// COLUMN
			{
				Type:   v1pb.SQLReviewRule_COLUMN_REQUIRED,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{
							"id",
							"created_ts",
							"updated_ts",
							"creator_id",
							"updater_id",
						},
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_NO_NULL,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_DISALLOW_CHANGE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_COMMENT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_CommentConventionPayload{
					CommentConventionPayload: &v1pb.SQLReviewRule_CommentConventionRulePayload{
						Required:  true,
						MaxLength: 10,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"JSON", "BINARY_FLOAT"},
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 20,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 20,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// SCHEMA
			{
				Type:   v1pb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// DATABASE
			{
				Type:   v1pb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE,
				Level:  v1pb.SQLReviewRule_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			// INDEX
			{
				Type:   v1pb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 5,
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_INDEX_PK_TYPE_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_INDEX_TYPE_NO_BLOB,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   v1pb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_NumberPayload{
					NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
						Number: 5,
					},
				},
			},
			// SYSTEM
			{
				Type:   v1pb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"utf8mb4", "UTF8"},
					},
				},
			},
			{
				Type:   v1pb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
				Level:  v1pb.SQLReviewRule_WARNING,
				Engine: v1pb.Engine_MYSQL,
				Payload: &v1pb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"utf8mb4_0900_ai_ci"},
					},
				},
			},
		},
	}

	return config
}
