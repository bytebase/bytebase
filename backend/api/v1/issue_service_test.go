package v1

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/review"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestIssueCommentsForMetadataEvents pins the record a metadata patch leaves
// on the issue timeline: one audit comment per changed field carrying the
// transition, and an approval check only when the workflow says the patch
// reset the approval. Which patches reset it is pinned in component/review.
func TestIssueCommentsForMetadataEvents(t *testing.T) {
	t.Parallel()
	comments, approvalCheck, err := issueCommentsForMetadataEvents(7, []review.Event{
		review.IssueTitleUpdatedEvent{FromTitle: "issue-a", ToTitle: "renamed issue"},
		review.IssueDescriptionUpdatedEvent{FromDescription: "", ToDescription: "updated description"},
		review.IssueLabelsUpdatedEvent{FromLabels: []string{"environment:prod"}, ToLabels: []string{"environment:staging"}},
		review.ApprovalCheckEvent{},
	})
	require.NoError(t, err)
	require.True(t, approvalCheck, "a reset approval is checked again")
	require.Len(t, comments, 3)
	for _, comment := range comments {
		require.EqualValues(t, 7, comment.IssueUID)
	}
	require.Equal(t, "issue-a", comments[0].Payload.GetIssueUpdate().GetFromTitle())
	require.Equal(t, "renamed issue", comments[0].Payload.GetIssueUpdate().GetToTitle())
	require.NotNil(t, comments[1].Payload.GetIssueUpdate().FromDescription, "an empty previous description is recorded as empty, not omitted")
	require.Equal(t, "updated description", comments[1].Payload.GetIssueUpdate().GetToDescription())
	require.Equal(t, []string{"environment:prod"}, comments[2].Payload.GetIssueUpdate().GetFromLabels())
	require.Equal(t, []string{"environment:staging"}, comments[2].Payload.GetIssueUpdate().GetToLabels())

	comments, approvalCheck, err = issueCommentsForMetadataEvents(7, nil)
	require.NoError(t, err)
	require.Empty(t, comments, "a no-op patch leaves no audit comment")
	require.False(t, approvalCheck)

	_, _, err = issueCommentsForMetadataEvents(7, []review.Event{review.SubmittedEvent{}})
	require.Error(t, err, "an event this method does not know how to record is a bug, not a silent drop")
}

// TestBuildIssueMessageRejectsBeforeAnyLookup pins the request shapes creation
// refuses on the request alone — a nil store proves nothing was read: the
// retired DATABASE_EXPORT type that legacy clients still send as wire value 3,
// a draft of anything but a database change, and a database-change draft
// naming no plan.
func TestBuildIssueMessageRejectsBeforeAnyLookup(t *testing.T) {
	ctx := issueServiceTestContext()
	service := &IssueService{}
	project := &store.ProjectMessage{ResourceID: "project-a", Setting: &storepb.Project{}}
	for _, tc := range []struct {
		name  string
		issue *v1pb.Issue
		want  string
	}{
		{"retired export type", &v1pb.Issue{Title: "legacy export issue", Type: v1pb.Issue_Type(3)}, "unknown issue type"},
		{"non-database draft", &v1pb.Issue{Title: "role request", Type: v1pb.Issue_ROLE_GRANT, Draft: true}, "draft issues must be database change issues"},
		{"draft without a plan", &v1pb.Issue{Type: v1pb.Issue_DATABASE_CHANGE, Draft: true}, "plan is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.buildIssueMessage(ctx, project, "creator@example.com", &v1pb.CreateIssueRequest{Parent: "projects/project-a", Issue: tc.issue}, nil)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// TestLinkedIssueForCreate pins what creating an issue means when its plan
// already has one. Idempotent draft creation and the draft-versus-rollout
// ordering are pinned against the database in backend/store
// (TestCreateDraftIssueIsIdempotent, TestCreateDraftAndRolloutAreSerialized).
func TestLinkedIssueForCreate(t *testing.T) {
	draftBy := func(creator string) *store.IssueMessage {
		return &store.IssueMessage{CreatorEmail: creator, Payload: &storepb.Issue{Draft: true}}
	}
	submittedBy := func(creator string) *store.IssueMessage {
		return &store.IssueMessage{CreatorEmail: creator, Payload: &storepb.Issue{}}
	}
	for _, tc := range []struct {
		name     string
		existing *store.IssueMessage
		incoming *store.IssueMessage
		wantCode connect.Code
		reuse    bool
	}{
		{"an unlinked plan is free", nil, draftBy("creator@example.com"), 0, false},
		{"a submitted issue is final for the plan", submittedBy("creator@example.com"), draftBy("creator@example.com"), connect.CodeAlreadyExists, false},
		{"a submission goes through the existing draft", draftBy("creator@example.com"), submittedBy("creator@example.com"), connect.CodeFailedPrecondition, false},
		{"another creator's draft is not exposed", draftBy("creator@example.com"), draftBy("other@example.com"), connect.CodeAlreadyExists, false},
		{"the same creator gets the draft back", draftBy("creator@example.com"), draftBy("creator@example.com"), 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := linkedIssueForCreate(101, tc.existing, tc.incoming)
			if tc.wantCode != 0 {
				require.Equal(t, tc.wantCode, connect.CodeOf(err))
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			if tc.reuse {
				require.Same(t, tc.existing, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}

func TestApproveIssueFailsClosedWhenIAMLookupFails(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	approval := proto.CloneOf(issue.Payload.GetApproval())
	approval.Approvers = nil
	_, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: approval},
	})
	require.NoError(t, err)
	_, err = stores.GetDB().ExecContext(ctx, "ALTER TABLE policy RENAME TO unavailable_policy")
	require.NoError(t, err)

	reviewerCtx := context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "reviewer@example.com",
		Name:  "reviewer",
	})
	_, err = service.ApproveIssue(reviewerCtx, connect.NewRequest(&v1pb.ApproveIssueRequest{
		Name: common.FormatIssue(issue.ProjectID, issue.UID),
	}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestCreateRolloutAndPendingTasksAllowsUnapprovedIssueWhenApprovalNotRequired(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	_, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
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

func TestCreateRolloutAndPendingTasksClassifiesApprovalRaceAsStale(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		Workspace:  "default",
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: true},
	}))
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	staleIssue := *issue
	staleIssue.Payload = proto.CloneOf(issue.Payload)

	unapproved := proto.CloneOf(issue.Payload)
	unapproved.Approval.Approvers = nil
	_, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: unapproved.Approval},
	})
	require.NoError(t, err)

	err = CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, &staleIssue, &store.ProjectMessage{
		Workspace:  "default",
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: true},
	}, []*store.TaskMessage{})
	require.Error(t, err)
	require.True(t, IsStaleRolloutApprovalError(err))

	got, getErr := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, getErr)
	require.False(t, got.Config.GetHasRollout())
}

func TestCreateRolloutAndPendingTasksRejectsDraft(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	draft, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Draft: true},
	})
	require.NoError(t, err)
	require.True(t, draft.Payload.GetDraft())
	require.False(t, plan.Config.GetHasRollout())
	require.Equal(t, storepb.Issue_OPEN, draft.Status)

	err = CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, draft, &store.ProjectMessage{
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}, []*store.TaskMessage{})
	require.ErrorIs(t, err, errDraftIssueNotSubmitted)

	gotPlan, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, gotPlan.Config.GetHasRollout())
	gotIssue := getIssueForTest(ctx, t, stores, draft.UID)
	require.Equal(t, storepb.Issue_OPEN, gotIssue.Status)
	require.True(t, gotIssue.Payload.GetDraft())
}

func TestCreateRolloutAndPendingTasksRejectsPersistedDraftWithStaleIssueSnapshot(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	draft, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Draft: true},
	})
	require.NoError(t, err)

	err = CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, nil, &store.ProjectMessage{
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}, []*store.TaskMessage{})
	require.ErrorIs(t, err, errDraftIssueNotSubmitted)

	gotPlan, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, gotPlan.Config.GetHasRollout())
	gotIssue := getIssueForTest(ctx, t, stores, draft.UID)
	require.Equal(t, storepb.Issue_OPEN, gotIssue.Status)
	require.True(t, gotIssue.Payload.GetDraft())
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

func setupIssueServiceTestStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	db, stores, _ := testcontainer.NewMetadataDB(t)

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	// SearchIssues authorizes the caller itself (CUSTOM auth), so the test
	// principal needs a role carrying bb.issues.get.
	_, err = stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("creator@example.com"),
		Roles:     []string{"roles/workspaceAdmin"},
	})
	require.NoError(t, err)
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
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	return NewIssueService(stores, webhook.NewManager(stores, nil), b, nil, iamManager)
}

func createIssueServiceApprovalIssue(ctx context.Context, t *testing.T, stores *store.Store) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Targets: []string{"instances/prod/databases/app"},
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
