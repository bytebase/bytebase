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
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

var (
	// builtinOnlyPolicyWithConflict is what review returns with no user policy
	// configured but the builtin walk-through still running, against a statement
	// that conflicts with the schema already in the database.
	builtinOnlyPolicyWithConflict = []*v1pb.PlanCheckRun_Result{
		{
			Status: v1pb.Advice_WARNING,
			// Postgres reaches this through DDL simulation where MySQL used a
			// walk-through, but either way it is the builtin rule speaking, which is
			// the point: no user policy is configured at this stage.
			Title:   "DDL simulation failed",
			Content: `ERROR: relation "tech_book" already exists (SQLSTATE 42P07)`,
			Code:    607,
			Report: &v1pb.PlanCheckRun_Result_SqlReviewReport_{
				SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
					StartPosition: &v1pb.Position{Line: 1},
				},
			},
		},
	}

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
			Environment: new("environments/prod"),
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

	// With no user-configured policy the builtin rules still run, and this is the
	// only place that proves it: statements[0] creates a table that does not
	// exist, so an OK there cannot tell "builtin ran and found nothing" apart
	// from "review was skipped". tech_book was created for real by the run:true
	// case in the fixture, so re-creating it must draw the builtin walk-through.
	result = createIssueAndReturnSQLReviewResult(ctx, a, ctl, ctl.project, database.Msg, "CREATE TABLE tech_book(id INT);", false)
	equalReviewResultProtos(a, builtinOnlyPolicyWithConflict, result, database.Msg.Name, "")
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

	return yamltest.WriteFile(filepath, yamlTests)
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
							Targets: []string{database.Name},
							Sheet:   sheet.Msg.Name,
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
