package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func cdcSpec(id, sheet string, targets []string, priorBackup bool) *storepb.PlanConfig_Spec {
	return &storepb.PlanConfig_Spec{
		Id: id,
		Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
				SheetSha256:       sheet,
				Targets:           targets,
				EnablePriorBackup: priorBackup,
			},
		},
	}
}

func TestPlanSpecsEqualSet(t *testing.T) {
	cases := []struct {
		name string
		a, b []*storepb.PlanConfig_Spec
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "identical single spec",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			want: true,
		},
		{
			name: "same set reordered",
			a: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			b: []*storepb.PlanConfig_Spec{
				cdcSpec("s2", "sha2", []string{"db2"}, false),
				cdcSpec("s1", "sha1", []string{"db1"}, false),
			},
			want: true,
		},
		{
			name: "added spec",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			want: false,
		},
		{
			name: "removed spec",
			a: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			want: false,
		},
		{
			name: "same id sheet differs",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha2", []string{"db1"}, false)},
			want: false,
		},
		{
			name: "same id targets differ",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1", "db2"}, false)},
			want: false,
		},
		{
			name: "same id prior_backup differs",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, true)},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, planSpecsEqualSet(tc.a, tc.b))
		})
	}
}

func TestUpdateIssueApprovalResetSkipsStalePlanApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanServiceTestStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 1,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, updated, err := resetIssueApprovalFindingIfPlanApprovalInputVersion(ctx, s, issue, 1)
	require.NoError(t, err)
	require.False(t, updated)
	require.Nil(t, updatedIssue)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 1, got.Payload.GetApproval().GetApprovalInputVersion())

	updatedIssue, updated, err = resetIssueApprovalFindingIfPlanApprovalInputVersion(ctx, s, issue, 2)
	require.NoError(t, err)
	require.True(t, updated)
	require.NotNil(t, updatedIssue)
	require.False(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestPlanServiceListPlansKeepsIssueLessDraftsVisible(t *testing.T) {
	ctx := context.Background()
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	changeDraft, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change draft",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: "change",
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)
	createDraft, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "create draft",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: "create",
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)

	response, err := service.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent:   "projects/project-a",
		PageSize: 100,
	}))
	require.NoError(t, err)
	var got []string
	for _, plan := range response.Msg.Plans {
		got = append(got, plan.Title)
	}
	require.ElementsMatch(t, []string{changeDraft.Name, createDraft.Name}, got)
}

func TestPlanServiceCreatePlanRejectsMixedDatabaseSpecs(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "creator@example.com",
		Name:  "creator",
	})
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	_, err := service.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: "projects/project-a",
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: "create",
					Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{},
					},
				},
				{
					Id: "change",
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{},
					},
				},
			},
		},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.ErrorContains(t, err, "each plan must contain only one type")
}

func setupPlanServiceTestStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
