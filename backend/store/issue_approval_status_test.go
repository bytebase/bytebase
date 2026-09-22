package store_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestIssueApprovalStatusFilterMatchesComputation pins the SQL derivation of
// approval_status to ComputeApprovalStatus. They are two implementations of
// one rule, and the SQL side has to read an absent JSON key as a zero value
// because payloads are written with a bare protojson.Marshal.
func TestIssueApprovalStatusFilterMatchesComputation(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)

	cases := []struct {
		title    string
		approval *storepb.IssuePayloadApproval
	}{
		{"no approval payload at all", nil},
		{"approval finding not done", &storepb.IssuePayloadApproval{ApprovalTemplate: approvalTemplateWithRoles("roles/workspaceAdmin")}},
		{"no template", &storepb.IssuePayloadApproval{ApprovalFindingDone: true}},
		{"template with an empty flow", &storepb.IssuePayloadApproval{ApprovalFindingDone: true, ApprovalTemplate: approvalTemplateWithRoles()}},
		{"no approvers yet", &storepb.IssuePayloadApproval{ApprovalFindingDone: true, ApprovalTemplate: approvalTemplateWithRoles("roles/workspaceAdmin")}},
		{"one of two steps approved", &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
			ApprovalTemplate:    approvalTemplateWithRoles("roles/workspaceAdmin", "roles/projectOwner"),
			Approvers:           []*storepb.IssuePayloadApproval_Approver{issueApprover(storepb.IssuePayloadApproval_Approver_APPROVED)},
		}},
		{"rejected at the second step", &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
			ApprovalTemplate:    approvalTemplateWithRoles("roles/workspaceAdmin", "roles/projectOwner"),
			Approvers: []*storepb.IssuePayloadApproval_Approver{
				issueApprover(storepb.IssuePayloadApproval_Approver_APPROVED),
				issueApprover(storepb.IssuePayloadApproval_Approver_REJECTED),
			},
		}},
		{"every step approved", &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
			ApprovalTemplate:    approvalTemplateWithRoles("roles/workspaceAdmin"),
			Approvers:           []*storepb.IssuePayloadApproval_Approver{issueApprover(storepb.IssuePayloadApproval_Approver_APPROVED)},
		}},
	}

	want := map[v1pb.ApprovalStatus][]int64{}
	for _, tc := range cases {
		issue := createIssueWithApproval(ctx, t, stores, tc.title, tc.approval)
		status := store.ComputeApprovalStatus(tc.approval)
		want[status] = append(want[status], issue.UID)
	}
	// Every branch of ComputeApprovalStatus should be represented, otherwise
	// the comparison below passes on empty sets.
	require.Len(t, want, 5)

	for status, wantUIDs := range want {
		statusName := status.String()
		issues, err := stores.ListIssues(ctx, &store.FindIssueMessage{
			Workspace:      "default",
			ProjectIDs:     []string{"project-a"},
			ApprovalStatus: &statusName,
		})
		require.NoError(t, err, "approval_status == %q", statusName)
		var gotUIDs []int64
		for _, issue := range issues {
			gotUIDs = append(gotUIDs, issue.UID)
		}
		slices.Sort(gotUIDs)
		slices.Sort(wantUIDs)
		require.Equal(t, wantUIDs, gotUIDs, "approval_status == %q", statusName)
	}
}

// TestListIssuesExcludeDraft pins the list-side half of drafts: a draft is
// reachable by name but never listed, so it takes no page slot and shows up
// nowhere a submitted issue would.
func TestListIssuesExcludeDraft(t *testing.T) {
	t.Parallel()
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)

	submitted := createIssueWithApproval(ctx, t, stores, "submitted", nil)
	draftPlan, err := stores.CreatePlan(ctx, &store.PlanMessage{ProjectID: "project-a", Name: "draft plan", Config: &storepb.PlanConfig{}}, "creator@example.com")
	require.NoError(t, err)
	draft, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &draftPlan.UID,
	})
	require.NoError(t, err)

	listed, err := stores.ListIssues(ctx, &store.FindIssueMessage{Workspace: "default", ProjectIDs: []string{"project-a"}, ExcludeDraft: true})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, submitted.UID, listed[0].UID)

	byName, err := stores.GetIssue(ctx, &store.FindIssueMessage{Workspace: "default", ProjectIDs: []string{"project-a"}, UID: &draft.UID})
	require.NoError(t, err)
	require.True(t, byName.Payload.GetDraft(), "the draft is still there by name")
}
