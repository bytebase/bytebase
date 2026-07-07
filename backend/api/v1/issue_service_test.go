package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestUpdateIssueLabelsResetsApprovalBeforeRollout(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	updateIssueLabels(ctx, t, service, issue, []string{"environment:staging"})

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, []string{"environment:staging"}, got.Payload.GetLabels())
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsClearedBeforeRolloutResetsApproval(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	updateIssueLabels(ctx, t, service, issue, nil)

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Empty(t, got.Payload.GetLabels())
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
	require.Len(t, service.bus.ApprovalCheckChan, 1)
}

func TestUpdateIssueLabelsDoesNotResetApprovalAfterRollout(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	approvalInputVersion := int64(2)
	marked, _, err := stores.CreateRolloutTasks(ctx, "project-a", plan.UID, &approvalInputVersion, nil, nil)
	require.NoError(t, err)
	require.True(t, marked)

	updateIssueLabels(ctx, t, service, issue, []string{"environment:staging"})

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, []string{"environment:staging"}, got.Payload.GetLabels())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
	require.Len(t, service.bus.ApprovalCheckChan, 0)
}

func TestCreateRolloutAndPendingTasksAllowsUnapprovedIssueWhenApprovalNotRequired(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	approvalInputVersion := int64(2)
	_, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
	})
	require.NoError(t, err)

	stalePlan := *plan
	stalePlan.Config = &storepb.PlanConfig{ApprovalInputVersion: 1}
	err = CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", &stalePlan, issue, &store.ProjectMessage{
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}, []*store.TaskMessage{})
	require.Error(t, err)
	require.True(t, IsStaleRolloutApprovalError(err))

	err = CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, issue, &store.ProjectMessage{
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}, []*store.TaskMessage{})
	require.NoError(t, err)

	gotPlan, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, gotPlan.Config.GetHasRollout())

	gotIssue := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, storepb.Issue_DONE, gotIssue.Status)
}

func TestUpdateIssueLabelsNoopDoesNotResetApproval(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	updateIssueLabels(ctx, t, service, issue, []string{" environment:prod ", "environment:prod"})

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, []string{"environment:prod"}, got.Payload.GetLabels())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
	require.Len(t, service.bus.ApprovalCheckChan, 0)
}

func TestUpdateIssueLabelsDoesNotResetCreateDatabaseApproval(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceCreateDatabaseApprovalIssue(ctx, t, stores)

	updateIssueLabels(ctx, t, service, issue, []string{"environment:staging"})

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, []string{"environment:staging"}, got.Payload.GetLabels())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
	require.Len(t, service.bus.ApprovalCheckChan, 0)
}

func setupIssueServiceTestStore(ctx context.Context, t *testing.T) *store.Store {
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
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
	return stores
}

func issueServiceTestContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "creator@example.com",
		Name:  "creator",
	})
	return ctx
}

func newIssueServiceForTest(t *testing.T, stores *store.Store) *IssueService {
	t.Helper()

	b, err := bus.New()
	require.NoError(t, err)
	return NewIssueService(stores, nil, b, nil, nil)
}

func createIssueServiceApprovalIssue(ctx context.Context, t *testing.T, stores *store.Store) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"environment:prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow:  &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
					Title: "manual approval",
				},
				Approvers: []*storepb.IssuePayloadApproval_Approver{
					{
						Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
						Principal: common.FormatUserEmail("creator@example.com"),
					},
				},
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)
	return plan, issue
}

func createIssueServiceCreateDatabaseApprovalIssue(ctx context.Context, t *testing.T, stores *store.Store) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
							Target: "instances/prod/databases/app",
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"environment:prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow:  &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
					Title: "manual approval",
				},
				Approvers: []*storepb.IssuePayloadApproval_Approver{
					{
						Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
						Principal: common.FormatUserEmail("creator@example.com"),
					},
				},
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)
	return plan, issue
}

func updateIssueLabels(ctx context.Context, t *testing.T, service *IssueService, issue *store.IssueMessage, labels []string) {
	t.Helper()

	_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:   common.FormatIssue(issue.ProjectID, issue.UID),
			Labels: labels,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
	}))
	require.NoError(t, err)
}

func getIssueForTest(ctx context.Context, t *testing.T, stores *store.Store, issueUID int64) *store.IssueMessage {
	t.Helper()

	got, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		UID:        &issueUID,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}
