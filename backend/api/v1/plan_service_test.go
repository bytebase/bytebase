package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/iam"
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

func TestResetIssueApprovalFindingSkipsDraftIssue(t *testing.T) {
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
		Title:        "draft issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Draft: true,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 1,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, updated, err := resetIssueApprovalFindingIfPlanApprovalInputVersion(ctx, s, issue, 2)
	require.NoError(t, err)
	require.False(t, updated)
	require.Nil(t, updatedIssue)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.True(t, got.Payload.GetDraft())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 1, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestPlanServiceListPlansHidesMalformedUIPlans(t *testing.T) {
	ctx := context.Background()
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	createPlan := func(name string, config *storepb.PlanConfig) *store.PlanMessage {
		t.Helper()
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "project-a",
			Name:      name,
			Config:    config,
		}, "creator@example.com")
		require.NoError(t, err)
		return plan
	}
	changeConfig := func(id, release string) *storepb.PlanConfig {
		return &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: id,
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{Release: release},
			},
		}}}
	}
	createConfig := func(id string) *storepb.PlanConfig {
		return &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: id,
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
			},
		}}}
	}

	malformedChange := createPlan("malformed change", changeConfig("malformed-change", ""))
	createPlan("malformed create", createConfig("malformed-create"))
	createPlan("malformed mixed", &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{
		createConfig("mixed-create").Specs[0],
		changeConfig("mixed-change", "").Specs[0],
	}})
	gitOps := createPlan("gitops", changeConfig("gitops", "projects/project-a/releases/release-a"))
	linked := createPlan("linked", changeConfig("linked", ""))
	export := createPlan("export", &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
		Id: "export",
		Config: &storepb.PlanConfig_Spec_ExportDataConfig{
			ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{},
		},
	}}})
	deleted := createPlan("deleted", changeConfig("deleted", ""))
	_, err := stores.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       deleted.UID,
		ProjectID: deleted.ProjectID,
		Deleted:   new(true),
	})
	require.NoError(t, err)
	_, err = stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    linked.ProjectID,
		CreatorEmail: "creator@example.com",
		Title:        "linked issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &linked.UID,
	})
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
	require.ElementsMatch(t, []string{gitOps.Name, linked.Name, export.Name, deleted.Name}, got)

	gotMalformed, err := service.GetPlan(ctx, connect.NewRequest(&v1pb.GetPlanRequest{
		Name: fmt.Sprintf("projects/project-a/plans/%d", malformedChange.UID),
	}))
	require.NoError(t, err)
	require.Equal(t, malformedChange.Name, gotMalformed.Msg.Title)
}

func TestPlanServiceUpdatePlanSynchronizesDraftIssueMetadata(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "editor@example.com",
		Name:  "editor",
	})
	stores := setupPlanServiceTestStore(ctx, t)
	_, err := stores.CreateRole(ctx, &store.RoleMessage{
		ResourceID: "planEditor",
		Workspace:  "default",
		Name:       "Plan editor",
		Permissions: map[permission.Permission]bool{
			permission.PlansUpdate: true,
		},
	})
	require.NoError(t, err)
	_, err = stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("editor@example.com"),
		Roles:     []string{"roles/planEditor"},
	})
	require.NoError(t, err)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := NewPlanService(stores, nil, iamManager, nil, nil)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "old plan title",
		Description: "old plan description",
		Config:      &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "old issue title",
		Description:  "old issue description",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	response, err := service.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name:        common.FormatPlan("project-a", plan.UID),
			Title:       "new plan title",
			Description: "new plan description",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "description"}},
	}))
	require.NoError(t, err)
	require.Equal(t, "new plan title", response.Msg.Title)
	require.Equal(t, "new plan description", response.Msg.Description)

	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		UID:        &issue.UID,
	})
	require.NoError(t, err)
	require.Equal(t, "new plan title", got.Title)
	require.Equal(t, "new plan description", got.Description)
	require.True(t, got.Payload.GetDraft())
	require.Equal(t, storepb.Issue_OPEN, got.Status)
}

func TestPlanServiceUpdatePlanClosesAndReopensLinkedDraftIssue(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "creator@example.com",
		Name:  "creator",
	})
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft plan",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	updateState := func(state v1pb.State) *v1pb.Plan {
		t.Helper()
		response, err := service.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
			Plan: &v1pb.Plan{
				Name:  common.FormatPlan("project-a", plan.UID),
				State: state,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
		}))
		require.NoError(t, err)
		return response.Msg
	}
	getIssue := func() *store.IssueMessage {
		t.Helper()
		got, err := stores.GetIssue(ctx, &store.FindIssueMessage{
			ProjectIDs: []string{"project-a"},
			UID:        &issue.UID,
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		return got
	}

	require.Equal(t, v1pb.State_DELETED, updateState(v1pb.State_DELETED).State)
	closed := getIssue()
	require.True(t, closed.Payload.GetDraft())
	require.Equal(t, storepb.Issue_CANCELED, closed.Status)

	require.Equal(t, v1pb.State_ACTIVE, updateState(v1pb.State_ACTIVE).State)
	reopened := getIssue()
	require.True(t, reopened.Payload.GetDraft())
	require.Equal(t, storepb.Issue_OPEN, reopened.Status)

	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: "project-a",
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestPlanServiceUpdatePlanPreservesSubmittedIssueReviewState(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "creator@example.com",
		Name:  "creator",
	})
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "submitted plan",
		Description: "submitted plan description",
		Config:      &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	approval := &storepb.IssuePayloadApproval{
		ApprovalFindingDone:  true,
		ApprovalInputVersion: 7,
	}
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "submitted issue",
		Description:  "submitted issue description",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Approval: approval},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	_, err = service.UpdatePlan(ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name:        common.FormatPlan("project-a", plan.UID),
			Title:       "updated plan",
			Description: "updated plan description",
			State:       v1pb.State_DELETED,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "description", "state"}},
	}))
	require.NoError(t, err)

	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		UID:        &issue.UID,
	})
	require.NoError(t, err)
	require.Equal(t, "submitted issue", got.Title)
	require.Equal(t, "submitted issue description", got.Description)
	require.Equal(t, storepb.Issue_OPEN, got.Status)
	require.False(t, got.Payload.GetDraft())
	require.Equal(t, approval, got.Payload.GetApproval())

	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: "project-a",
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestPlanServiceListPlansProjectsLinkedDraftApprovalStatusAsUnspecified(t *testing.T) {
	ctx := context.Background()
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "linked draft",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    plan.ProjectID,
		CreatorEmail: "creator@example.com",
		Title:        plan.Name,
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{
			Draft: true,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	response, err := service.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent:   "projects/project-a",
		PageSize: 100,
	}))
	require.NoError(t, err)
	require.Len(t, response.Msg.Plans, 1)
	require.Equal(t, common.FormatIssue(issue.ProjectID, issue.UID), response.Msg.Plans[0].Issue)
	require.Equal(t, v1pb.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED, response.Msg.Plans[0].ApprovalStatus)
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
