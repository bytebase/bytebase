package reviewrun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestRuleExecutorSkipsAnEmptyPolicy pins that review switched off is no
// review: the executor returns no findings without resolving a target, where
// the same issue fails once a rule is on because its target does not exist.
func TestRuleExecutorSkipsAnEmptyPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p', 'ws', 'P');
	`)
	require.NoError(t, err)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "p",
		Name:      "plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "spec-1",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/missing/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "p",
		CreatorEmail: "creator@example.com",
		Title:        "issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	setRules := func(rules ...storepb.ReviewRuleType) {
		t.Helper()
		payload, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: rules})
		require.NoError(t, err)
		_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:    "ws",
			ResourceType: storepb.Policy_PROJECT,
			Resource:     common.FormatProject("p"),
			Type:         storepb.Policy_REVIEW_RULE,
			Payload:      string(payload),
			Enforce:      true,
		})
		require.NoError(t, err)
	}
	executor := NewRuleExecutor(stores)

	setRules()
	comments, err := executor.RunOnce(ctx, "p", issue.UID)
	require.NoError(t, err)
	require.Empty(t, comments)

	setRules(storepb.ReviewRuleType_SYNTAX)
	_, err = executor.RunOnce(ctx, "p", issue.UID)
	require.Error(t, err)
}

// TestRuleExecutorSkipsAnEngineWithoutAReviewer pins that the rule review
// leaves an engine it has no reviewer for to V1's statement advice: the run
// skips its target and completes with no findings.
func TestRuleExecutorSkipsAnEngineWithoutAReviewer(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p', 'ws', 'P');
	`)
	require.NoError(t, err)
	payload, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX}})
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "ws",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("p"),
		Type:         storepb.Policy_REVIEW_RULE,
		Payload:      string(payload),
		Enforce:      true,
	})
	require.NoError(t, err)
	sheets, err := stores.CreateSheets(ctx, "p", &store.SheetMessage{Statement: "DELETE FROM t;"})
	require.NoError(t, err)

	runOn := func(instanceID string, engine storepb.Engine) error {
		t.Helper()
		require.False(t, ReviewsEngine(engine))
		_, err := stores.CreateInstance(ctx, &store.InstanceMessage{
			ResourceID: instanceID,
			Workspace:  "ws",
			Metadata: &storepb.Instance{
				Engine:      engine,
				DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
			},
		})
		require.NoError(t, err)
		_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
			ProjectID:    "p",
			InstanceID:   instanceID,
			DatabaseName: "app",
			Metadata:     &storepb.DatabaseMetadata{},
		})
		require.NoError(t, err)
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "p",
			Name:      "plan",
			Config: &storepb.PlanConfig{
				Specs: []*storepb.PlanConfig_Spec{{
					Id: "spec-1",
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Targets:     []string{"instances/" + instanceID + "/databases/app"},
							SheetSha256: sheets[0].Sha256,
						},
					},
				}},
			},
		}, "creator@example.com")
		require.NoError(t, err)
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID:    "p",
			CreatorEmail: "creator@example.com",
			Title:        "issue",
			Type:         storepb.Issue_DATABASE_CHANGE,
			Payload:      &storepb.Issue{},
			PlanUID:      &plan.UID,
		})
		require.NoError(t, err)
		comments, err := NewRuleExecutor(stores).RunOnce(ctx, "p", issue.UID)
		require.Empty(t, comments)
		return err
	}

	require.NoError(t, runOn("dynamo", storepb.Engine_DYNAMODB))
	require.NoError(t, runOn("clickhouse", storepb.Engine_CLICKHOUSE))
}
