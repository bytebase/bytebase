package v1

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/review"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestDraftLabelUpdateConflictsWithConcurrentSubmission(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	draft := createReadyDraftForUpdateTest(ctx, t, stores, service, "label submission race")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var lockedUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT id FROM issue
		WHERE project = $1 AND id = $2
		FOR UPDATE`, draft.ProjectID, draft.UID).Scan(&lockedUID))

	updateResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue: &v1pb.Issue{
				Name: common.FormatIssue(draft.ProjectID, draft.UID), Labels: []string{"environment:staging"},
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
		}))
		updateResult <- err
	}()
	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)
	_, err = tx.ExecContext(ctx, `
		UPDATE issue
		SET payload = payload || jsonb_build_object('draft', false)
		WHERE project = $1 AND id = $2`, draft.ProjectID, draft.UID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	select {
	case err := <-updateResult:
		require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	case <-time.After(5 * time.Second):
		t.Fatal("draft label update did not return")
	}
	stored := getIssueForTest(ctx, t, stores, draft.UID)
	require.False(t, stored.Payload.GetDraft())
	require.Equal(t, []string{"team:database"}, stored.Payload.GetLabels())
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID, IssueUID: &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
	require.Empty(t, service.bus.ApprovalCheckChan)
}

func TestConcurrentSubmissionWithLabelsWritesOneAuditComment(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	draft := createReadyDraftForUpdateTest(ctx, t, stores, service, "concurrent labels")

	type outcome struct {
		response *connect.Response[v1pb.Issue]
		err      error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			<-start
			response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
				Issue: &v1pb.Issue{
					Name: common.FormatIssue(draft.ProjectID, draft.UID), Labels: []string{"environment:prod"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels", "draft"}},
			}))
			results <- outcome{response: response, err: err}
		})
	}
	close(start)
	wg.Wait()
	close(results)
	for result := range results {
		require.NoError(t, result.err)
		require.False(t, result.response.Msg.GetDraft())
	}
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID, IssueUID: &draft.UID,
	})
	require.NoError(t, err)
	var labelUpdates, submissions int
	for _, comment := range comments {
		switch comment.Payload.Event.(type) {
		case *storepb.IssueCommentPayload_IssueUpdate_:
			labelUpdates++
		case *storepb.IssueCommentPayload_ReviewSubmission_:
			submissions++
		default:
		}
	}
	require.Equal(t, 1, labelUpdates)
	require.Equal(t, 1, submissions)
	require.Len(t, service.bus.ApprovalCheckChan, 1)
}

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

	_, err := review.NewWorkflow(stores).CreateRollout(ctx, review.CreateRolloutInput{
		Workspace: "default",
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		BuildTasks: func(context.Context, *store.PlanMessage, *store.ProjectMessage) ([]*store.TaskMessage, error) {
			return nil, nil
		},
	})
	require.NoError(t, err)

	updateIssueLabels(ctx, t, service, issue, []string{"environment:staging"})

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, []string{"environment:staging"}, got.Payload.GetLabels())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
	require.Len(t, service.bus.ApprovalCheckChan, 0)
}

func TestConcurrentIdenticalLabelUpdatesCreateOneAuditComment(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	for i := range 10 {
		labels := []string{fmt.Sprintf("iteration:%d", i)}
		start := make(chan struct{})
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				<-start
				_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
					Issue: &v1pb.Issue{
						Name:   common.FormatIssue(issue.ProjectID, issue.UID),
						Labels: labels,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				}))
				errs <- err
			})
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
	}

	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: "project-a",
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Len(t, comments, 10)
}

func TestConcurrentIdenticalTitleUpdatesCreateOneAuditComment(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			<-start
			_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
				Issue: &v1pb.Issue{
					Name:  common.FormatIssue(issue.ProjectID, issue.UID),
					Title: "renamed issue",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
			}))
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	update := comments[0].Payload.GetIssueUpdate()
	require.Equal(t, "issue-a", update.GetFromTitle())
	require.Equal(t, "renamed issue", update.GetToTitle())
}

func TestMixedIssuePatchRollsBackWhenLabelsFail(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)
	_, err := stores.GetDB().ExecContext(ctx, `
		ALTER TABLE issue ADD CONSTRAINT reject_atomic_test_label
		CHECK (NOT COALESCE(payload->'labels' ? 'reject', false))`)
	require.NoError(t, err)

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:        common.FormatIssue(issue.ProjectID, issue.UID),
			Title:       "renamed issue",
			Description: "updated description",
			Labels:      []string{"reject"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "description", "labels"}},
	}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, "issue-a", got.Title)
	require.Empty(t, got.Description)
	require.Equal(t, []string{"environment:prod"}, got.Payload.GetLabels())
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestMixedIssuePatchCommitsWithAuditComments(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:        common.FormatIssue(issue.ProjectID, issue.UID),
			Title:       "renamed issue",
			Description: "updated description",
			Labels:      []string{"environment:staging"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "description", "labels"}},
	}))
	require.NoError(t, err)
	require.Equal(t, "renamed issue", response.Msg.Title)
	require.Equal(t, "updated description", response.Msg.Description)
	require.Equal(t, []string{"environment:staging"}, response.Msg.Labels)

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.Len(t, service.bus.ApprovalCheckChan, 1)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Len(t, comments, 3)
	var titleAudit, descriptionAudit, labelsAudit bool
	for _, comment := range comments {
		update := comment.Payload.GetIssueUpdate()
		require.NotNil(t, update)
		switch {
		case update.FromTitle != nil:
			titleAudit = true
			require.Equal(t, "issue-a", update.GetFromTitle())
			require.Equal(t, "renamed issue", update.GetToTitle())
		case update.FromDescription != nil:
			descriptionAudit = true
			require.Empty(t, update.GetFromDescription())
			require.Equal(t, "updated description", update.GetToDescription())
		default:
			labelsAudit = true
			require.Equal(t, []string{"environment:prod"}, update.GetFromLabels())
			require.Equal(t, []string{"environment:staging"}, update.GetToLabels())
		}
	}
	require.True(t, titleAudit)
	require.True(t, descriptionAudit)
	require.True(t, labelsAudit)
}

func TestIssueMetadataNoopDoesNotCreateAuditComments(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:        common.FormatIssue(issue.ProjectID, issue.UID),
			Title:       " issue-a ",
			Description: "",
			Labels:      []string{" environment:prod ", "environment:prod"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "description", "labels"}},
	}))
	require.NoError(t, err)

	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, issue.UpdatedAt, got.UpdatedAt)
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.Empty(t, service.bus.ApprovalCheckChan)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestApproveIssueFailsClosedWhenIAMLookupFails(t *testing.T) {
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

	gotPlan, getPlanErr := stores.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: "project-a",
		UID:       &plan.UID,
	})
	require.NoError(t, getPlanErr)
	require.NotNil(t, gotPlan)
	gotIssue := getIssueForTest(ctx, t, stores, draft.UID)
	assert.Error(t, err)
	assert.False(t, gotPlan.Config.GetHasRollout())
	assert.Equal(t, storepb.Issue_OPEN, gotIssue.Status)
	assert.True(t, gotIssue.Payload.GetDraft())
}

func TestCreateRolloutAndPendingTasksRejectsPersistedDraftWithStaleIssueSnapshot(t *testing.T) {
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
	gotPlan, getPlanErr := stores.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: "project-a",
		UID:       &plan.UID,
	})
	require.NoError(t, getPlanErr)
	require.False(t, gotPlan.Config.GetHasRollout())
	gotIssue := getIssueForTest(ctx, t, stores, draft.UID)
	require.Equal(t, storepb.Issue_OPEN, gotIssue.Status)
	require.True(t, gotIssue.Payload.GetDraft())
}

func TestCreateDraftAndRolloutAreSerialized(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	for i := range 10 {
		t.Run(fmt.Sprintf("attempt-%d", i), func(t *testing.T) {
			plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
				ProjectID: "project-a",
				Name:      "draft rollout race",
				Config: &storepb.PlanConfig{
					Specs: []*storepb.PlanConfig_Spec{{
						Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
							ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
								SheetSha256: "sheet",
							},
						},
					}},
				},
			}, "creator@example.com")
			require.NoError(t, err)

			start := make(chan struct{})
			draftResult := make(chan error, 1)
			rolloutResult := make(chan error, 1)
			go func() {
				<-start
				_, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
					Parent: "projects/project-a",
					Issue: &v1pb.Issue{
						Type:  v1pb.Issue_DATABASE_CHANGE,
						Plan:  common.FormatPlan("project-a", plan.UID),
						Draft: true,
					},
				}))
				draftResult <- err
			}()
			go func() {
				<-start
				rolloutResult <- CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, nil, &store.ProjectMessage{
					ResourceID: "project-a",
					Setting:    &storepb.Project{RequireIssueApproval: false},
				}, []*store.TaskMessage{})
			}()
			close(start)
			draftErr := <-draftResult
			rolloutErr := <-rolloutResult

			require.NotEqual(t, draftErr == nil, rolloutErr == nil)
			if draftErr == nil {
				require.ErrorIs(t, rolloutErr, errDraftIssueNotSubmitted)
			} else {
				require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(draftErr))
				require.NoError(t, rolloutErr)
			}

			gotPlan, err := stores.GetPlan(ctx, &store.FindPlanMessage{
				ProjectID: "project-a",
				UID:       &plan.UID,
			})
			require.NoError(t, err)
			linkedIssue, err := stores.GetIssue(ctx, &store.FindIssueMessage{
				ProjectIDs: []string{"project-a"},
				PlanUID:    &plan.UID,
			})
			require.NoError(t, err)
			require.NotEqual(t, gotPlan.Config.GetHasRollout(), linkedIssue != nil)
			if linkedIssue != nil {
				require.True(t, linkedIssue.Payload.GetDraft())
				require.Equal(t, storepb.Issue_OPEN, linkedIssue.Status)
			}
		})
	}
}

func TestCreateDraftIssueRejectedAfterRollout(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "rolled out plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						SheetSha256: "sheet",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	require.NoError(t, CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, nil, &store.ProjectMessage{
		ResourceID: "project-a",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}, []*store.TaskMessage{}))

	_, err = service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  common.FormatPlan("project-a", plan.UID),
			Draft: true,
		},
	}))

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	linkedIssue, getErr := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, getErr)
	require.Nil(t, linkedIssue)
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

func TestUpdateIssueLabelsOnDraftPreservesApproval(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	draft, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Draft: true},
	})
	require.NoError(t, err)
	require.True(t, draft.Payload.GetDraft())
	expectedApproval := draft.Payload.GetApproval()

	updateIssueLabels(ctx, t, service, draft, []string{"environment:staging"})

	got := getIssueForTest(ctx, t, stores, draft.UID)
	assert.True(t, got.Payload.GetDraft())
	assert.Equal(t, []string{"environment:staging"}, got.Payload.GetLabels())
	assert.Equal(t, expectedApproval, got.Payload.GetApproval())
	assert.Empty(t, service.bus.ApprovalCheckChan)
}

func TestRetryIssueApprovalRejectsDraft(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	draft, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Draft: true},
	})
	require.NoError(t, err)
	require.True(t, draft.Payload.GetDraft())

	_, err = service.RetryIssueApproval(ctx, connect.NewRequest(&v1pb.RetryIssueApprovalRequest{
		Name: common.FormatIssue(draft.ProjectID, draft.UID),
	}))

	got := getIssueForTest(ctx, t, stores, draft.UID)
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.True(t, got.Payload.GetDraft())
	assert.Equal(t, draft.Payload.GetApproval(), got.Payload.GetApproval())
	assert.Empty(t, service.bus.ApprovalCheckChan)
}

func TestDraftIssueApprovalActionsRejectedBeforeApprovalValidation(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)

	tests := []struct {
		name string
		call func(service *IssueService, issueName string) error
	}{
		{
			name: "approve",
			call: func(service *IssueService, issueName string) error {
				_, err := service.ApproveIssue(ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{Name: issueName}))
				return err
			},
		},
		{
			name: "reject",
			call: func(service *IssueService, issueName string) error {
				_, err := service.RejectIssue(ctx, connect.NewRequest(&v1pb.RejectIssueRequest{Name: issueName}))
				return err
			},
		},
		{
			name: "request",
			call: func(service *IssueService, issueName string) error {
				_, err := service.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{Name: issueName}))
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newIssueServiceForTest(t, stores)
			plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
				ProjectID: "project-a",
				Name:      fmt.Sprintf("%s draft approval plan", test.name),
				Config:    &storepb.PlanConfig{},
			}, "creator@example.com")
			require.NoError(t, err)
			draft, err := stores.CreateIssue(ctx, &store.IssueMessage{
				ProjectID:    "project-a",
				CreatorEmail: "creator@example.com",
				Title:        fmt.Sprintf("%s draft", test.name),
				Type:         storepb.Issue_DATABASE_CHANGE,
				Payload:      &storepb.Issue{Draft: true},
				PlanUID:      &plan.UID,
			})
			require.NoError(t, err)
			require.True(t, draft.Payload.GetDraft())
			require.Nil(t, draft.Payload.GetApproval())

			err = test.call(service, common.FormatIssue(draft.ProjectID, draft.UID))

			got := getIssueForTest(ctx, t, stores, draft.UID)
			assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			assert.Equal(t, draft, got)
			assert.Empty(t, service.bus.ApprovalCheckChan)
			assert.Empty(t, service.bus.RolloutCreationChan)
		})
	}
}

func TestStaleReviewRequestDispatchesNoPostCommitEffects(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	staleApproval := issue.Payload.GetApproval()
	staleApproval.Approvers[0].Status = storepb.IssuePayloadApproval_Approver_REJECTED
	_, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: staleApproval},
	})
	require.NoError(t, err)
	_, err = stores.GetDB().ExecContext(ctx, `
		UPDATE plan
		SET config = jsonb_set(config, '{approvalInputVersion}', '3')
		WHERE project = $1 AND id = $2`, plan.ProjectID, plan.UID)
	require.NoError(t, err)

	for range 2 {
		_, err = service.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{
			Name:    common.FormatIssue(issue.ProjectID, issue.UID),
			Comment: "retry",
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	}
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
	require.Empty(t, service.bus.ApprovalCheckChan)
	require.Empty(t, service.bus.RolloutCreationChan)
	got := getIssueForTest(ctx, t, stores, issue.UID)
	require.True(t, got.Payload.GetApproval().Equal(staleApproval))
}

func TestCreateDraftIssue(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "draft plan",
		Description: "draft description",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							SheetSha256: "sheet",
							Targets:     []string{"instances/prod/databases/app"},
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	response, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  common.FormatPlan("project-a", plan.UID),
			Draft: true,
		},
	}))
	require.NoError(t, err)
	require.Equal(t, common.FormatPlan("project-a", plan.UID), response.Msg.Plan)
	require.Equal(t, "draft plan", response.Msg.Title)
	require.Equal(t, "draft description", response.Msg.Description)
	require.True(t, response.Msg.Draft)

	stored, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, err)
	require.True(t, stored.Payload.GetDraft())

	var rawDraft string
	err = stores.GetDB().QueryRowContext(
		ctx,
		"SELECT payload->>'draft' FROM issue WHERE project = $1 AND id = $2",
		"project-a",
		stored.UID,
	).Scan(&rawDraft)
	require.NoError(t, err)
	require.Equal(t, "true", rawDraft)

	_, err = stores.GetDB().ExecContext(
		ctx,
		"UPDATE project SET setting = jsonb_set(setting, '{forceIssueLabels}', 'true') WHERE resource_id = $1",
		"project-a",
	)
	require.NoError(t, err)

	createDatabasePlan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "create database draft",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
							Target:   "instances/prod",
							Database: "app",
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	createDatabaseDraft, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  common.FormatPlan("project-a", createDatabasePlan.UID),
			Draft: true,
		},
	}))
	require.NoError(t, err)
	require.True(t, createDatabaseDraft.Msg.GetDraft())
}

func TestCreateDraftIssueIsIdempotent(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	service := NewIssueService(stores, nil, b, nil, nil)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "idempotent draft plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							SheetSha256: "sheet",
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	first, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Title:       "original title",
			Description: "original description",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        common.FormatPlan("project-a", plan.UID),
			Labels:      []string{"original"},
			Draft:       true,
		},
	}))
	require.NoError(t, err)

	second, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Title:       "replacement title",
			Description: "replacement description",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        common.FormatPlan("project-a", plan.UID),
			Labels:      []string{"replacement"},
			Draft:       true,
		},
	}))
	require.NoError(t, err)
	require.Equal(t, first.Msg, second.Msg)
	require.Empty(t, b.ApprovalCheckChan)

	issues, err := stores.ListIssues(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, plan.Name, issues[0].Title)
	require.Equal(t, plan.Description, issues[0].Description)
	require.Equal(t, []string{"original"}, issues[0].Payload.GetLabels())

	concurrentPlan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "concurrent draft plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							SheetSha256: "sheet",
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	type createResult struct {
		response *connect.Response[v1pb.Issue]
		err      error
	}
	start := make(chan struct{})
	results := make(chan createResult, 2)
	var waitGroup sync.WaitGroup
	for _, title := range []string{"first concurrent title", "second concurrent title"} {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			response, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/project-a",
				Issue: &v1pb.Issue{
					Title: title,
					Type:  v1pb.Issue_DATABASE_CHANGE,
					Plan:  common.FormatPlan("project-a", concurrentPlan.UID),
					Draft: true,
				},
			}))
			results <- createResult{response: response, err: err}
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var concurrentIssueName string
	for result := range results {
		require.NoError(t, result.err)
		require.NotNil(t, result.response)
		if concurrentIssueName == "" {
			concurrentIssueName = result.response.Msg.GetName()
		} else {
			require.Equal(t, concurrentIssueName, result.response.Msg.GetName())
		}
	}
	concurrentIssues, err := stores.ListIssues(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &concurrentPlan.UID,
	})
	require.NoError(t, err)
	require.Len(t, concurrentIssues, 1)
}

func TestCreateDraftIssueDoesNotExposeAnotherCreatorsDraft(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "private draft plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						SheetSha256: "sheet",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	existing, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "private draft",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	otherCtx := context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "other@example.com",
		Name:  "other",
	})
	response, err := service.CreateIssue(otherCtx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  common.FormatPlan("project-a", plan.UID),
			Draft: true,
		},
	}))

	require.Nil(t, response)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	stored, getErr := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, getErr)
	require.Equal(t, existing, stored)
}

func TestCreateSubmittedIssueBlockedByExistingDraft(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft conflict plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							SheetSha256: "sheet",
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	draft, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "original draft",
		Description:  "original description",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{
			Labels: []string{"original"},
			Draft:  true,
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)
	require.True(t, draft.Payload.GetDraft())

	_, err = service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Title:       "submitted replacement",
			Description: "replacement description",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        common.FormatPlan("project-a", plan.UID),
			Labels:      []string{"replacement"},
			Draft:       false,
		},
	}))

	issues, listErr := stores.ListIssues(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, listErr)
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.Len(t, issues, 1)
	if assert.NotEmpty(t, issues) {
		assert.Equal(t, draft, issues[0])
	}
	assert.Empty(t, service.bus.ApprovalCheckChan)
}

func TestCreateDraftIssueBlockedByExistingNonDraftIssue(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "submitted plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{
				{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							SheetSha256: "sheet",
						},
					},
				},
			},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	existing, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "submitted issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Labels: []string{"original"}},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)
	require.False(t, existing.Payload.GetDraft())

	var hasDraft bool
	err = stores.GetDB().QueryRowContext(
		ctx,
		"SELECT payload ? 'draft' FROM issue WHERE project = $1 AND id = $2",
		"project-a",
		existing.UID,
	).Scan(&hasDraft)
	require.NoError(t, err)
	require.False(t, hasDraft)

	_, err = service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Title: "draft replacement",
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  common.FormatPlan("project-a", plan.UID),
			Draft: true,
		},
	}))
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))

	stored, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, err)
	require.Equal(t, existing.UID, stored.UID)
	require.Equal(t, "submitted issue", stored.Title)
	require.Equal(t, []string{"original"}, stored.Payload.GetLabels())
	require.False(t, stored.Payload.GetDraft())

	direct, err := service.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
		Name: common.FormatIssue("project-a", existing.UID),
	}))
	require.NoError(t, err)
	require.False(t, direct.Msg.GetDraft())
}

func TestCreateDraftIssueRejectsUnsupportedWorkflows(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	createPlan := func(name string, specs ...*storepb.PlanConfig_Spec) *store.PlanMessage {
		t.Helper()
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "project-a",
			Name:      name,
			Config:    &storepb.PlanConfig{Specs: specs},
		}, "creator@example.com")
		require.NoError(t, err)
		return plan
	}
	changeSpec := func(release string) *storepb.PlanConfig_Spec {
		return &storepb.PlanConfig_Spec{
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
					SheetSha256: "sheet",
					Release:     release,
				},
			},
		}
	}
	createSpec := func() *storepb.PlanConfig_Spec {
		return &storepb.PlanConfig_Spec{
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
					Target:   "instances/prod",
					Database: "app",
				},
			},
		}
	}
	exportSpec := func() *storepb.PlanConfig_Spec {
		return &storepb.PlanConfig_Spec{
			Config: &storepb.PlanConfig_Spec_ExportDataConfig{
				ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{
					SheetSha256: "sheet",
				},
			},
		}
	}

	validPlan := createPlan("valid", changeSpec(""))
	gitOpsPlan := createPlan("gitops", changeSpec("projects/project-a/releases/release"))
	mixedPlan := createPlan("mixed", createSpec(), changeSpec(""))
	exportPlan := createPlan("export", exportSpec())

	tests := []struct {
		name  string
		issue *v1pb.Issue
	}{
		{
			name: "no plan",
			issue: &v1pb.Issue{
				Type:  v1pb.Issue_DATABASE_CHANGE,
				Draft: true,
			},
		},
		{
			name: "GitOps plan",
			issue: &v1pb.Issue{
				Type:  v1pb.Issue_DATABASE_CHANGE,
				Plan:  common.FormatPlan("project-a", gitOpsPlan.UID),
				Draft: true,
			},
		},
		{
			name: "mixed plan",
			issue: &v1pb.Issue{
				Type:  v1pb.Issue_DATABASE_CHANGE,
				Plan:  common.FormatPlan("project-a", mixedPlan.UID),
				Draft: true,
			},
		},
		{
			name: "export plan",
			issue: &v1pb.Issue{
				Type:  v1pb.Issue_DATABASE_CHANGE,
				Plan:  common.FormatPlan("project-a", exportPlan.UID),
				Draft: true,
			},
		},
		{
			name: "non-database issue",
			issue: &v1pb.Issue{
				Title: "role request",
				Type:  v1pb.Issue_ROLE_GRANT,
				Plan:  common.FormatPlan("project-a", validPlan.UID),
				Draft: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/project-a",
				Issue:  test.issue,
			}))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}

func TestIssueListsHideDraftIssues(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	submittedPlan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "submitted plan",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	submitted, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "submitted",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &submittedPlan.UID,
	})
	require.NoError(t, err)

	draftPlan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft plan",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
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

	list, err := service.ListIssues(ctx, connect.NewRequest(&v1pb.ListIssuesRequest{
		Parent:   "projects/project-a",
		PageSize: 1,
	}))
	require.NoError(t, err)
	require.Equal(t, []string{common.FormatIssue("project-a", submitted.UID)}, issueNames(list.Msg.Issues))
	require.Empty(t, list.Msg.GetNextPageToken())

	search, err := service.SearchIssues(ctx, connect.NewRequest(&v1pb.SearchIssuesRequest{
		Parent:   "projects/project-a",
		PageSize: 1,
	}))
	require.NoError(t, err)
	require.Equal(t, []string{common.FormatIssue("project-a", submitted.UID)}, issueNames(search.Msg.Issues))
	require.Empty(t, search.Msg.GetNextPageToken())

	direct, err := service.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
		Name: common.FormatIssue("project-a", draft.UID),
	}))
	require.NoError(t, err)
	require.True(t, direct.Msg.GetDraft())
}

func issueNames(issues []*v1pb.Issue) []string {
	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		names = append(names, issue.GetName())
	}
	return names
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
	return NewIssueService(stores, webhook.NewManager(stores, nil), b, nil, nil)
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

func createReadyDraftForUpdateTest(
	ctx context.Context,
	t *testing.T,
	stores *store.Store,
	service *IssueService,
	title string,
) *store.IssueMessage {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a", Name: title,
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
					Targets: []string{"instances/prod/databases/app"}, SheetSha256: "sheet",
				},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)
	response, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type: v1pb.Issue_DATABASE_CHANGE, Plan: common.FormatPlan("project-a", plan.UID),
			Labels: []string{"team:database"}, Draft: true,
		},
	}))
	require.NoError(t, err)
	_, issueUID, err := common.GetProjectIDIssueUID(response.Msg.GetName())
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: plan.ProjectID, PlanUID: plan.UID,
		Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()},
	})
	require.NoError(t, err)
	require.True(t, created)
	run, err := stores.GetPlanCheckRun(ctx, plan.ProjectID, plan.UID)
	require.NoError(t, err)
	require.NoError(t, stores.UpdatePlanCheckRun(ctx, plan.ProjectID, store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: plan.Config.GetApprovalInputVersion(),
		Results: []*storepb.PlanCheckRunResult_Result{{
			Type: storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE, Status: storepb.Advice_SUCCESS,
		}},
	}, run.UID))
	return getIssueForTest(ctx, t, stores, issueUID)
}

func waitForTransactionBlock(ctx context.Context, t *testing.T, db *sql.DB, tx *sql.Tx) {
	t.Helper()
	var blockerPID int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockerPID))
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT count(*)
			FROM pg_stat_activity AS activity
			WHERE activity.pid <> pg_backend_pid()
			  AND $1 = ANY(pg_blocking_pids(activity.pid))`, blockerPID).Scan(&waiting))
		if waiting >= 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for a session blocked by transaction PID %d", blockerPID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
