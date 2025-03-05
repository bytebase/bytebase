package server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// generateOnboardingData generates onboarding data after the first signup.
func (s *Server) generateOnboardingData(ctx context.Context, user *store.UserMessage) error {
	userID := user.ID
	setting := &storepb.Project{
		AllowModifyStatement: true,
		AutoResolveIssue:     true,
	}
	project, err := s.store.CreateProjectV2(ctx, &store.ProjectMessage{
		ResourceID: "project-sample",
		Title:      "Sample Project",
		Setting:    setting,
	}, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to create onboarding project")
	}

	// Test Sample Instance
	testInstance, err := s.store.CreateInstanceV2(ctx, &store.InstanceMessage{
		ResourceID:   "test-sample-instance",
		Title:        "Test Sample Instance",
		Engine:       storepb.Engine_POSTGRES,
		ExternalLink: "",
		Metadata: &storepb.InstanceMetadata{
			DataSources: []*storepb.DataSource{
				{
					Id:                 "admin",
					Type:               storepb.DataSourceType_ADMIN,
					Username:           postgres.SampleUser,
					ObfuscatedPassword: common.Obfuscate("", s.secret),
					Host:               common.GetPostgresSocketDir(),
					Port:               strconv.Itoa(s.profile.SampleDatabasePort),
					Database:           postgres.SampleDatabaseTest,
				},
			},
		},
		EnvironmentID: api.DefaultTestEnvironmentID,
		Activation:    false,
	}, -1)
	if err != nil {
		return errors.Wrapf(err, "failed to create test onboarding instance")
	}

	// Sync the instance schema so we can transfer the sample database later.
	if _, _, _, err := s.schemaSyncer.SyncInstance(ctx, testInstance); err != nil {
		return errors.Wrapf(err, "failed to sync test onboarding instance")
	}

	// Transfer sample database to the just created project.
	transferDatabaseMessage := &store.UpdateDatabaseMessage{
		InstanceID:   testInstance.ResourceID,
		DatabaseName: postgres.SampleDatabaseTest,
		ProjectID:    &project.ResourceID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer test sample database")
	}

	dbName := postgres.SampleDatabaseTest
	testDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &testInstance.ResourceID,
		DatabaseName:    &dbName,
		IsCaseSensitive: store.IsObjectCaseSensitive(testInstance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find test onboarding instance")
	}
	if testDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, testDatabase); err != nil {
		return errors.Wrapf(err, "failed to sync test sample database schema")
	}

	// Prod Sample Instance
	prodInstance, err := s.store.CreateInstanceV2(ctx, &store.InstanceMessage{
		ResourceID:   "prod-sample-instance",
		Title:        "Prod Sample Instance",
		Engine:       storepb.Engine_POSTGRES,
		ExternalLink: "",
		Metadata: &storepb.InstanceMetadata{
			DataSources: []*storepb.DataSource{
				{
					Id:                 "admin",
					Type:               storepb.DataSourceType_ADMIN,
					Username:           postgres.SampleUser,
					ObfuscatedPassword: common.Obfuscate("", s.secret),
					Host:               common.GetPostgresSocketDir(),
					Port:               strconv.Itoa(s.profile.SampleDatabasePort + 1),
					Database:           postgres.SampleDatabaseProd,
				},
			},
		},
		EnvironmentID: api.DefaultProdEnvironmentID,
		Activation:    false,
	}, -1)
	if err != nil {
		return errors.Wrapf(err, "failed to create prod onboarding instance")
	}

	// Sync the instance schema so we can transfer the sample database later.
	if _, _, _, err := s.schemaSyncer.SyncInstance(ctx, prodInstance); err != nil {
		return errors.Wrapf(err, "failed to sync prod onboarding instance")
	}

	// Transfer sample database to the just created project.
	transferDatabaseMessage = &store.UpdateDatabaseMessage{
		InstanceID:   prodInstance.ResourceID,
		DatabaseName: postgres.SampleDatabaseProd,
		ProjectID:    &project.ResourceID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer prod sample database")
	}

	dbName = postgres.SampleDatabaseProd
	prodDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &prodInstance.ResourceID,
		DatabaseName:    &dbName,
		IsCaseSensitive: store.IsObjectCaseSensitive(prodInstance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find prod onboarding instance")
	}
	if prodDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, prodDatabase); err != nil {
		return errors.Wrapf(err, "failed to sync prod sample database schema")
	}

	// Add a sample SQL Review policy to the prod environment. This pairs with the following schema
	// change issue to demonstrate the SQL Review feature.
	sqlReviewConfig := &store.ReviewConfigMessage{
		ID:      "sample",
		Name:    "SQL Review Sample Policy",
		Enforce: true,
		Payload: getSampleSQLReviewPayload(),
	}

	config, err := s.store.CreateReviewConfig(ctx, sqlReviewConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create onboarding SQL Review policy")
	}

	policyPayload, err := protojson.Marshal(&storepb.TagPolicy{
		Tags: map[string]string{
			string(api.ReservedTagReviewConfig): common.FormatReviewConfig(config.ID),
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal environment tag")
	}

	_, err = s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		Resource:          common.FormatEnvironment(api.DefaultProdEnvironmentID),
		ResourceType:      api.PolicyResourceTypeEnvironment,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeTag,
		InheritFromParent: true,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create onboarding environment tag policy")
	}

	// Create a standalone sample SQL sheet.
	// This is different from another sample SQL sheet created below, which is created as part of
	// creating a schema change issue.
	if _, err = s.store.CreateWorkSheet(ctx, &store.WorkSheetMessage{
		CreatorID:    userID,
		ProjectID:    project.ResourceID,
		InstanceID:   &prodDatabase.InstanceID,
		DatabaseName: &prodDatabase.DatabaseName,
		Title:        "Sample Sheet",
		Statement:    "SELECT * FROM salary;",
		Visibility:   store.ProjectReadWorkSheet,
	}); err != nil {
		return errors.Wrapf(err, "failed to create sample work sheet")
	}

	// Create a schema update issue and start with creating the sheet for the schema update.
	testSheet, err := s.sheetManager.CreateSheet(ctx, &store.SheetMessage{
		CreatorID: api.SystemBotID,
		ProjectID: project.ResourceID,
		Title:     "Alter table to test sample instance for sample issue",
		Statement: "ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '';",

		Payload: &storepb.SheetPayload{
			Engine: testInstance.Engine,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create test sheet for sample project")
	}

	prodSheet, err := s.sheetManager.CreateSheet(ctx, &store.SheetMessage{
		CreatorID: api.SystemBotID,
		ProjectID: project.ResourceID,
		Title:     "Alter table to prod sample instance for sample issue",
		Statement: "ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '';",
		Payload: &storepb.SheetPayload{
			Engine: prodInstance.Engine,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create prod sheet for sample project")
	}

	// Use new CI/CD API.
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, user.ID)
	childCtx = context.WithValue(childCtx, common.UserContextKey, user)
	plan, err := s.planService.CreatePlan(childCtx, &v1pb.CreatePlanRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Plan: &v1pb.Plan{
			Title: "Onboarding sample plan for adding email column to Employee table",
			Steps: []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Id: uuid.NewString(),
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
									Target: common.FormatDatabase(testDatabase.InstanceID, testDatabase.DatabaseName),
									Sheet:  fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, testSheet.UID),
								},
							},
						},
					},
				},
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Id: uuid.NewString(),
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
									Target: common.FormatDatabase(prodDatabase.InstanceID, prodDatabase.DatabaseName),
									// This will violate the NOT NULL SQL Review policy configured above and emit a
									// warning. Thus to demonstrate the SQL Review capability.
									Sheet: fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, prodSheet.UID),
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create plan for sample project")
	}
	rollout, err := s.rolloutService.CreateRollout(childCtx, &v1pb.CreateRolloutRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Rollout: &v1pb.Rollout{
			Plan: plan.Name,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create rollout for sample project")
	}
	if _, err := s.issueService.CreateIssue(childCtx, &v1pb.CreateIssueRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Issue: &v1pb.Issue{
			Title: "👉👉👉 [START HERE] Add email column to Employee table",
			Description: `A sample issue to showcase how to review database schema change.

				Click "Approve" button to apply the schema update.`,
			Type:    v1pb.Issue_DATABASE_CHANGE,
			Plan:    plan.Name,
			Rollout: rollout.Name,
		},
	}); err != nil {
		return errors.Wrapf(err, "failed to create issue for sample project")
	}

	// Add a sensitive data policy to pair it with the sample query below. So that user can
	// experience the sensitive data masking feature from SQL Editor.
	dbSchema, err := s.store.GetDBSchema(ctx, prodDatabase.InstanceID, prodDatabase.DatabaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to get db schema for database %s", prodDatabase.String())
	}
	dbModelConfig := model.NewDatabaseConfig(nil)
	if dbSchema != nil {
		dbModelConfig = dbSchema.GetInternalConfig()
	}
	schemaConfig := dbModelConfig.CreateOrGetSchemaConfig("public")
	tableConfig := schemaConfig.CreateOrGetTableConfig("salary")
	columnConfig := tableConfig.CreateOrGetColumnConfig("amount")
	columnConfig.SemanticType = "default"

	if err := s.store.UpdateDBSchema(ctx, prodDatabase.InstanceID, prodDatabase.DatabaseName, &store.UpdateDBSchemaMessage{Config: dbModelConfig.BuildDatabaseConfig()}); err != nil {
		return errors.Wrapf(err, "failed to update db config for database %v", prodDatabase.String())
	}

	return nil
}

// getSampleSQLReviewPayload returns a sample SQL review policy for preparing onboardign data.
func getSampleSQLReviewPayload() *storepb.ReviewConfigPayload {
	payload := &storepb.ReviewConfigPayload{}

	// Add DropEmptyDatabase rule.
	for _, e := range []storepb.Engine{
		storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_OCEANBASE,
		storepb.Engine_MARIADB,
	} {
		payload.SqlReviewRules = append(payload.SqlReviewRules, &storepb.SQLReviewRule{
			Type:    string(advisor.SchemaRuleDropEmptyDatabase),
			Level:   storepb.SQLReviewRuleLevel_ERROR,
			Engine:  e,
			Payload: "{}",
		})
	}

	// Add ColumnNotNull rule.
	for _, e := range []storepb.Engine{
		storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_POSTGRES,
		storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_OCEANBASE,
		storepb.Engine_SNOWFLAKE, storepb.Engine_MSSQL, storepb.Engine_MARIADB,
	} {
		payload.SqlReviewRules = append(payload.SqlReviewRules, &storepb.SQLReviewRule{
			Type:    string(advisor.SchemaRuleColumnNotNull),
			Level:   storepb.SQLReviewRuleLevel_WARNING,
			Engine:  e,
			Payload: "{}",
		})
	}

	// Add TableDropNamingConvention rule.
	for _, e := range []storepb.Engine{
		storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_POSTGRES,
		storepb.Engine_OCEANBASE, storepb.Engine_SNOWFLAKE, storepb.Engine_MSSQL,
		storepb.Engine_MARIADB,
	} {
		payload.SqlReviewRules = append(payload.SqlReviewRules, &storepb.SQLReviewRule{
			Type:    string(advisor.SchemaRuleTableDropNamingConvention),
			Level:   storepb.SQLReviewRuleLevel_ERROR,
			Engine:  e,
			Payload: "{\"format\":\"_del$\"}",
		})
	}

	return payload
}
