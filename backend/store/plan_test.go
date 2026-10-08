package store_test

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestListPlansExcludesMalformedUIPlans pins the list filter the plan UI reads
// through: an active plan with no issue is hidden unless it is a homogeneous
// release-backed change plan — those are GitOps plans, which never get an
// issue — while a deleted plan and any plan an issue links to stay visible.
// The filter is a list-only view: a hidden plan is still there to GetPlan.
func TestListPlansExcludesMalformedUIPlans(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	createPlan := func(name string, config *storepb.PlanConfig) *store.PlanMessage {
		t.Helper()
		plan, err := s.CreatePlan(ctx, &store.PlanMessage{ProjectID: "project-a", Name: name, Config: config}, "creator@example.com")
		require.NoError(t, err)
		return plan
	}
	changeConfig := func(id, release string) *storepb.PlanConfig {
		return &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id:     id,
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{Release: release}},
		}}}
	}
	createConfig := func(id string) *storepb.PlanConfig {
		return &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id:     id,
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{}},
		}}}
	}

	createPlan("malformed change", changeConfig("malformed-change", ""))
	createPlan("malformed create", createConfig("malformed-create"))
	createPlan("malformed mixed", &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{
		createConfig("mixed-create").Specs[0],
		changeConfig("mixed-change", "").Specs[0],
	}})
	oldMalformed := createPlan("old malformed", changeConfig("old", ""))
	_, err = db.ExecContext(ctx, `UPDATE plan SET created_at = CURRENT_TIMESTAMP - INTERVAL '31 days' WHERE project = $1 AND id = $2`, oldMalformed.ProjectID, oldMalformed.UID)
	require.NoError(t, err)
	gitOps := createPlan("GitOps", changeConfig("gitops", "projects/project-a/releases/release-a"))
	deleted := createPlan("deleted", changeConfig("deleted", ""))
	_, err = db.ExecContext(ctx, `UPDATE plan SET deleted = TRUE WHERE project = $1 AND id = $2`, deleted.ProjectID, deleted.UID)
	require.NoError(t, err)
	linked := createPlan("linked", changeConfig("linked", ""))
	_, err = s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID: linked.ProjectID, CreatorEmail: "creator@example.com", PlanUID: &linked.UID,
		Title: "linked issue", Type: storepb.Issue_DATABASE_CHANGE, Payload: &storepb.Issue{},
	})
	require.NoError(t, err)

	plans, err := s.ListPlans(ctx, &store.FindPlanMessage{Workspace: "default", ProjectID: "project-a", ExcludeMalformedUIPlans: true})
	require.NoError(t, err)
	var got []string
	for _, plan := range plans {
		got = append(got, plan.Name)
	}
	require.ElementsMatch(t, []string{gitOps.Name, deleted.Name, linked.Name}, got)

	hidden, err := s.GetPlan(ctx, &store.FindPlanMessage{Workspace: "default", ProjectID: "project-a", UID: &oldMalformed.UID})
	require.NoError(t, err)
	require.Equal(t, oldMalformed.Name, hidden.Name, "hidden from the list, still there by name")
}
