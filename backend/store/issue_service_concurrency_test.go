package store_test

// Concurrency, transaction and pagination tests for the issue service. They
// assert SQL behavior — serialization, rollback, claim ordering, page
// boundaries — so they live here rather than in backend/api/v1, which may not
// hold a real metadata Postgres.

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestDraftLabelUpdateConflictsWithConcurrentSubmission(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service, issueBus := newIssueServiceForTest(t, stores)
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
	require.Empty(t, issueBus.ApprovalCheckChan)
}

func TestConcurrentSubmissionWithLabelsWritesOneAuditComment(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service, issueBus := newIssueServiceForTest(t, stores)
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
	require.Len(t, issueBus.ApprovalCheckChan, 1)
}

func TestConcurrentIdenticalLabelUpdatesCreateOneAuditComment(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service, _ := newIssueServiceForTest(t, stores)
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
	service, _ := newIssueServiceForTest(t, stores)
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
	service, _ := newIssueServiceForTest(t, stores)
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

func TestCreateDraftAndRolloutAreSerialized(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service, _ := newIssueServiceForTest(t, stores)

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
				rolloutResult <- apiv1.CreateRolloutAndPendingTasks(ctx, stores, "creator@example.com", plan, nil, &store.ProjectMessage{
					ResourceID: "project-a",
					Setting:    &storepb.Project{RequireIssueApproval: false},
				}, []*store.TaskMessage{})
			}()
			close(start)
			draftErr := <-draftResult
			rolloutErr := <-rolloutResult

			require.NotEqual(t, draftErr == nil, rolloutErr == nil)
			if draftErr == nil {
				require.True(t, apiv1.IsDraftIssueNotSubmittedError(rolloutErr))
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

func TestCreateDraftIssueIsIdempotent(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	service := apiv1.NewIssueService(stores, nil, b, nil, nil)

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

func TestIssueApprovalFiltersRunBeforePaging(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service, _ := newIssueServiceForTest(t, stores)

	// Created first, so the default id DESC ordering puts it behind both
	// non-matching issues. With page size 1 the pre-fix code read the two
	// non-matching rows, minted a token, and then dropped every row it had.
	waiting := createIssueWithApproval(ctx, t, stores, "waiting on the caller", &storepb.IssuePayloadApproval{
		ApprovalTemplate:    approvalTemplateWithRoles("roles/workspaceAdmin"),
		ApprovalFindingDone: true,
	})
	createIssueWithApproval(ctx, t, stores, "already approved", &storepb.IssuePayloadApproval{
		ApprovalTemplate:    approvalTemplateWithRoles("roles/workspaceAdmin"),
		Approvers:           []*storepb.IssuePayloadApproval_Approver{issueApprover(storepb.IssuePayloadApproval_Approver_APPROVED)},
		ApprovalFindingDone: true,
	})
	// The caller holds roles/workspaceAdmin, not roles/projectOwner, so this one
	// is pending but waiting on somebody else.
	createIssueWithApproval(ctx, t, stores, "waiting on another role", &storepb.IssuePayloadApproval{
		ApprovalTemplate:    approvalTemplateWithRoles("roles/projectOwner"),
		ApprovalFindingDone: true,
	})

	const filter = `approval_status == "PENDING" && current_approver == "users/creator@example.com"`
	want := []string{common.FormatIssue("project-a", waiting.UID)}

	list, err := service.ListIssues(ctx, connect.NewRequest(&v1pb.ListIssuesRequest{
		Parent:   "projects/project-a",
		Filter:   filter,
		PageSize: 1,
	}))
	require.NoError(t, err)
	require.Equal(t, want, issueNames(list.Msg.Issues))
	require.Empty(t, list.Msg.GetNextPageToken())

	search, err := service.SearchIssues(ctx, connect.NewRequest(&v1pb.SearchIssuesRequest{
		Parent:   "projects/project-a",
		Filter:   filter,
		PageSize: 1,
	}))
	require.NoError(t, err)
	require.Equal(t, want, issueNames(search.Msg.Issues))
	require.Empty(t, search.Msg.GetNextPageToken())
}

// The fixture helpers below are duplicated from
// backend/api/v1/issue_service_test.go, which keeps its own copies for the
// issue tests that stay there. Change both together: nothing catches a
// divergence, and these tests would keep passing against a stale fixture.
// The duplication ends when those tests move to a fake.
// TestIssueApprovalStatusFilterMatchesConverter pins the SQL derivation of
// approval_status to computeApprovalStatus. They are two implementations of one
// rule, and the SQL side has to read an absent JSON key as a zero value because
// payloads are written with a bare protojson.Marshal.
func issueNames(issues []*v1pb.Issue) []string {
	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		names = append(names, issue.GetName())
	}
	return names
}

func setupIssueServiceTestStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	db, stores, _ := newTestDB(t)
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

// newIssueServiceForTest also returns the bus it built: the service keeps it
// unexported, and two tests assert on what the handler published.
func newIssueServiceForTest(t *testing.T, stores *store.Store) (*apiv1.IssueService, *bus.Bus) {
	t.Helper()

	b, err := bus.New()
	require.NoError(t, err)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	return apiv1.NewIssueService(stores, webhook.NewManager(stores, nil), b, nil, iamManager), b
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
	service *apiv1.IssueService,
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

func createIssueWithApproval(ctx context.Context, t *testing.T, stores *store.Store, title string, approval *storepb.IssuePayloadApproval) *store.IssueMessage {
	t.Helper()

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      title + " plan",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        title,
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Approval: approval},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)
	return issue
}

func approvalTemplateWithRoles(roles ...string) *storepb.ApprovalTemplate {
	return &storepb.ApprovalTemplate{Flow: &storepb.ApprovalFlow{Roles: roles}}
}

func issueApprover(status storepb.IssuePayloadApproval_Approver_Status) *storepb.IssuePayloadApproval_Approver {
	return &storepb.IssuePayloadApproval_Approver{
		Status:    status,
		Principal: common.FormatUserEmail("creator@example.com"),
	}
}

// TestIssueApprovalFiltersRunBeforePaging is the regression lock for audit
// finding T13: approval_status and current_approver used to be applied to the
// page after the store had already cut it and minted the page token, so
// filtering the default My Issues view answered with an empty page and a live
// next page token underneath it.
