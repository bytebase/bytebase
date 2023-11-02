package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// generateOnboardingData generates onboarding data after the first signup.
func (s *Server) generateOnboardingData(ctx context.Context, user *store.UserMessage) error {
	userID := user.ID
	project, err := s.store.CreateProjectV2(ctx, &store.ProjectMessage{
		ResourceID: "project-sample",
		Title:      "Sample Project",
		Key:        "SAM",
		Workflow:   api.UIWorkflow,
		Visibility: api.Public,
		TenantMode: api.TenantModeDisabled,
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
		DataSources: []*store.DataSourceMessage{
			{
				ID:                 "admin",
				Type:               api.Admin,
				Username:           postgres.SampleUser,
				ObfuscatedPassword: common.Obfuscate("", s.secret),
				Host:               common.GetPostgresSocketDir(),
				Port:               strconv.Itoa(s.profile.SampleDatabasePort),
				Database:           postgres.SampleDatabaseTest,
			},
		},
		EnvironmentID: api.DefaultTestEnvironmentID,
		Activation:    false,
	}, userID, -1)
	if err != nil {
		return errors.Wrapf(err, "failed to create test onboarding instance")
	}

	// Sync the instance schema so we can transfer the sample database later.
	if err := s.schemaSyncer.SyncInstance(ctx, testInstance); err != nil {
		return errors.Wrapf(err, "failed to sync test onboarding instance")
	}

	// Transfer sample database to the just created project.
	transferDatabaseMessage := &store.UpdateDatabaseMessage{
		InstanceID:   testInstance.ResourceID,
		DatabaseName: postgres.SampleDatabaseTest,
		ProjectID:    &project.ResourceID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer test sample database")
	}

	dbName := postgres.SampleDatabaseTest
	testDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &testInstance.ResourceID,
		DatabaseName:        &dbName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(testInstance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find test onboarding instance")
	}
	if testDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, testDatabase, true /* force */); err != nil {
		return errors.Wrapf(err, "failed to sync test sample database schema")
	}

	// Prod Sample Instance
	prodInstance, err := s.store.CreateInstanceV2(ctx, &store.InstanceMessage{
		ResourceID:   "prod-sample-instance",
		Title:        "Prod Sample Instance",
		Engine:       storepb.Engine_POSTGRES,
		ExternalLink: "",
		DataSources: []*store.DataSourceMessage{
			{
				ID:                 "admin",
				Type:               api.Admin,
				Username:           postgres.SampleUser,
				ObfuscatedPassword: common.Obfuscate("", s.secret),
				Host:               common.GetPostgresSocketDir(),
				Port:               strconv.Itoa(s.profile.SampleDatabasePort + 1),
				Database:           postgres.SampleDatabaseProd,
			},
		},
		EnvironmentID: api.DefaultProdEnvironmentID,
		Activation:    false,
	}, userID, -1)
	if err != nil {
		return errors.Wrapf(err, "failed to create prod onboarding instance")
	}

	// Sync the instance schema so we can transfer the sample database later.
	if err := s.schemaSyncer.SyncInstance(ctx, prodInstance); err != nil {
		return errors.Wrapf(err, "failed to sync prod onboarding instance")
	}

	// Transfer sample database to the just created project.
	transferDatabaseMessage = &store.UpdateDatabaseMessage{
		InstanceID:   prodInstance.ResourceID,
		DatabaseName: postgres.SampleDatabaseProd,
		ProjectID:    &project.ResourceID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer prod sample database")
	}

	dbName = postgres.SampleDatabaseProd
	prodDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &prodInstance.ResourceID,
		DatabaseName:        &dbName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(prodInstance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find prod onboarding instance")
	}
	if prodDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, prodDatabase, true /* force */); err != nil {
		return errors.Wrapf(err, "failed to sync prod sample database schema")
	}

	// Add a sample SQL Review policy to the prod environment. This pairs with the following schema
	// change issue to demonstrate the SQL Review feature.
	policyPayload, err := protojson.Marshal(getSampleSQLReviewPolicy())
	if err != nil {
		return errors.Wrapf(err, "failed to marshal onboarding SQL Review policy")
	}
	_, err = s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       api.DefaultProdEnvironmentUID,
		ResourceType:      api.PolicyResourceTypeEnvironment,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeSQLReview,
		InheritFromParent: true,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to create onboarding SQL Review policy")
	}

	// Create a standalone sample SQL sheet.
	// This is different from another sample SQL sheet created below, which is created as part of
	// creating a schema change issue.
	sheetCreate := &store.SheetMessage{
		CreatorID:   userID,
		ProjectUID:  project.UID,
		DatabaseUID: &prodDatabase.UID,
		Name:        "Sample Sheet",
		Statement:   "SELECT * FROM salary;",
		Visibility:  store.ProjectSheet,
		Source:      store.SheetFromBytebase,
		Type:        store.SheetForSQL,
	}
	_, err = s.store.CreateSheet(ctx, sheetCreate)
	if err != nil {
		return errors.Wrapf(err, "failed to create sample sheet")
	}

	// Create a schema update issue and start with creating the sheet for the schema update.
	testSheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
		CreatorID: api.SystemBotID,

		ProjectUID:  project.UID,
		DatabaseUID: &testDatabase.UID,

		Name:       "Alter table to test sample instance for sample issue",
		Statement:  "ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '';",
		Visibility: store.ProjectSheet,
		Source:     store.SheetFromBytebaseArtifact,
		Type:       store.SheetForSQL,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create test sheet for sample project")
	}

	prodSheet, err := s.store.CreateSheet(ctx, &store.SheetMessage{
		CreatorID: api.SystemBotID,

		ProjectUID:  project.UID,
		DatabaseUID: &prodDatabase.UID,

		Name:       "Alter table to prod sample instance for sample issue",
		Statement:  "ALTER TABLE employee ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '';",
		Visibility: store.ProjectSheet,
		Source:     store.SheetFromBytebaseArtifact,
		Type:       store.SheetForSQL,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create prod sheet for sample project")
	}

	var issueLink string
	// Use new CI/CD API.
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, user.ID)
	plan, err := s.rolloutService.CreatePlan(childCtx, &v1pb.CreatePlanRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Plan: &v1pb.Plan{
			Title: "Onboarding sample plan for adding email column to Employee table",
			Steps: []*v1pb.Plan_Step{
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
									Target: fmt.Sprintf("instances/%s/databases/%s", testDatabase.InstanceID, testDatabase.DatabaseName),
									Sheet:  fmt.Sprintf("projects/%s/sheets/%d", project.ResourceID, testSheet.UID),
								},
							},
						},
					},
				},
				{
					Specs: []*v1pb.Plan_Spec{
						{
							Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
								ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
									Type:   v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
									Target: fmt.Sprintf("instances/%s/databases/%s", prodDatabase.InstanceID, prodDatabase.DatabaseName),
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
		Plan:   plan.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create rollout for sample project")
	}
	issue, err := s.issueService.CreateIssue(childCtx, &v1pb.CreateIssueRequest{
		Parent: fmt.Sprintf("projects/%s", project.ResourceID),
		Issue: &v1pb.Issue{
			Title: "👉👉👉 [START HERE] Add email column to Employee table",
			Description: `A sample issue to showcase how to review database schema change.

				Click "Approve" button to apply the schema update.`,
			Type:     v1pb.Issue_DATABASE_CHANGE,
			Assignee: fmt.Sprintf("users/%s", user.Email),
			Plan:     plan.Name,
			Rollout:  rollout.Name,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create issue for sample project")
	}
	issueLink = fmt.Sprintf("/issue/%s-%s", slug.Make(issue.Title), issue.Uid)

	// Bookmark the issue.
	if _, err := s.store.CreateBookmarkV2(ctx, &store.BookmarkMessage{
		Name: "Sample Issue",
		Link: issueLink,
	}, userID); err != nil {
		return errors.Wrapf(err, "failed to bookmark sample issue")
	}

	// Add a sensitive data policy to pair it with the sample query below. So that user can
	// experience the sensitive data masking feature from SQL Editor.
	maskingPolicy := &storepb.MaskingPolicy{
		MaskData: []*storepb.MaskData{
			{
				Schema:       "public",
				Table:        "salary",
				Column:       "amount",
				MaskingLevel: storepb.MaskingLevel_FULL,
			},
		},
	}
	policyPayload, err = json.Marshal(maskingPolicy)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal onboarding sensitive data policy")
	}

	_, err = s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       prodDatabase.UID,
		ResourceType:      api.PolicyResourceTypeDatabase,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeMasking,
		InheritFromParent: true,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}, userID)
	if err != nil {
		return errors.Wrapf(err, "failed to create onboarding sensitive data policy")
	}

	return nil
}

// getSampleSQLReviewPolicy returns a sample SQL review policy for preparing onboardign data.
func getSampleSQLReviewPolicy() *storepb.SQLReviewPolicy {
	policy := &storepb.SQLReviewPolicy{
		Name: "SQL Review Sample Policy",
	}

	ruleList := []*storepb.SQLReviewRule{}

	// Add DropEmptyDatabase rule for MySQL, TiDB, MariaDB.
	for _, e := range []storepb.Engine{storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB} {
		ruleList = append(ruleList, &storepb.SQLReviewRule{
			Type:    string(advisor.SchemaRuleDropEmptyDatabase),
			Level:   storepb.SQLReviewRuleLevel_ERROR,
			Engine:  e,
			Payload: "{}",
		})
	}

	// Add ColumnNotNull rule for MySQL, TiDB, MariaDB, Postgres.
	for _, e := range []storepb.Engine{storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_POSTGRES} {
		ruleList = append(ruleList, &storepb.SQLReviewRule{
			Type:    string(advisor.SchemaRuleColumnNotNull),
			Level:   storepb.SQLReviewRuleLevel_WARNING,
			Engine:  e,
			Payload: "{}",
		})
	}

	// Add TableDropNamingConvention rule for MySQL, TiDB, MariaDB Postgres.
	for _, e := range []storepb.Engine{storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_POSTGRES} {
		ruleList = append(ruleList, &storepb.SQLReviewRule{
			Type:    string(advisor.SchemaRuleTableDropNamingConvention),
			Level:   storepb.SQLReviewRuleLevel_ERROR,
			Engine:  e,
			Payload: "{\"format\":\"_del$\"}",
		})
	}

	policy.RuleList = ruleList
	return policy
}
