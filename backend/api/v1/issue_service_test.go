package v1

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestIssueWritePermissionContract(t *testing.T) {
	service := v1pb.File_v1_issue_service_proto.Services().ByName("IssueService")
	require.NotNil(t, service)

	updateIssue := service.Methods().ByName("UpdateIssue")
	require.NotNil(t, updateIssue)
	require.Equal(t, "bb.issues.update", proto.GetExtension(updateIssue.Options(), v1pb.E_Permission))

	createIssue := service.Methods().ByName("CreateIssue")
	require.NotNil(t, createIssue)
	require.Equal(t, "bb.issues.create", proto.GetExtension(createIssue.Options(), v1pb.E_Permission))
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

	approvalInputVersion := int64(2)
	marked, _, err := stores.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
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

func TestConcurrentRetryIssueApprovalOnlyWinningFindingCreatesRollout(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_APPROVAL,
		Workspace: "default",
		Value:     &storepb.WorkspaceApprovalSetting{},
	})
	require.NoError(t, err)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "concurrent retry",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
						Target:      "instances/prod",
						Database:    "app",
						Environment: "prod",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "concurrent retry",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)
	b, err := bus.New()
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	service := NewIssueService(stores, webhook.NewManager(stores, nil), b, licenseService, nil)
	issueName := common.FormatIssue(issue.ProjectID, issue.UID)

	results := runIssueCallsBehindUpdateLock(ctx, t, stores, issue, []func() error{
		func() error {
			_, err := service.RetryIssueApproval(ctx, connect.NewRequest(&v1pb.RetryIssueApprovalRequest{Name: issueName}))
			return err
		},
		func() error {
			_, err := service.RetryIssueApproval(ctx, connect.NewRequest(&v1pb.RetryIssueApprovalRequest{Name: issueName}))
			return err
		},
	})

	require.NoError(t, results[0])
	require.NoError(t, results[1])
	require.Len(t, b.RolloutCreationChan, 1)
	stored := getIssueForTest(ctx, t, stores, issue.UID)
	require.True(t, stored.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, stored.Payload.GetApproval().GetApprovalInputVersion())
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

func TestRequestIssueDoesNotOverwriteConcurrentLabelApprovalReset(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, issue := createIssueServiceApprovalIssue(ctx, t, stores)

	rejectedApproval := proto.CloneOf(issue.Payload.GetApproval())
	rejectedApproval.Approvers = []*storepb.IssuePayloadApproval_Approver{{
		Status:    storepb.IssuePayloadApproval_Approver_REJECTED,
		Principal: common.FormatUserEmail("creator@example.com"),
	}}
	issue, err := stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: rejectedApproval},
	})
	require.NoError(t, err)

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var lockedUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT id
		FROM issue
		WHERE project = $1 AND id = $2
		FOR UPDATE
	`, issue.ProjectID, issue.UID).Scan(&lockedUID))
	require.Equal(t, issue.UID, lockedUID)

	requestResult := make(chan error, 1)
	go func() {
		_, err := service.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{
			Name: common.FormatIssue(issue.ProjectID, issue.UID),
		}))
		requestResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)

	_, err = tx.ExecContext(ctx, `
		UPDATE issue
		SET payload = payload
			|| jsonb_build_object('labels', '["environment:staging"]'::jsonb)
			|| jsonb_build_object(
				'approval',
				'{"approvalFindingDone":false,"approvalInputVersion":"2"}'::jsonb
			)
		WHERE project = $1 AND id = $2
	`, issue.ProjectID, issue.UID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	require.Contains(t,
		[]connect.Code{connect.CodeAborted, connect.CodeFailedPrecondition},
		connect.CodeOf(receiveTestResult(t, requestResult, "approval action did not return")),
	)
	stored := getIssueForTest(ctx, t, stores, issue.UID)
	require.Equal(t, []string{"environment:staging"}, stored.Payload.GetLabels())
	require.False(t, stored.Payload.GetApproval().GetApprovalFindingDone())
	require.Empty(t, stored.Payload.GetApproval().GetApprovers())
	require.Empty(t, service.bus.RolloutCreationChan)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestConcurrentApprovalActionsOnlyWinningTransitionHasSideEffects(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{AllowSelfApproval: true},
	}))
	_, err := stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: "default",
		Member:    common.FormatUserEmail("creator@example.com"),
		Roles:     []string{"roles/workspaceAdmin"},
	})
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_APP_IM,
		Workspace: "default",
		Value:     &storepb.AppIMSetting{},
	})
	require.NoError(t, err)

	requestReceived := make(chan struct{}, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestReceived <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	_, err = stores.CreateProjectWebhook(ctx, "project-a", &store.ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhook{
			Type: storepb.WebhookType_SLACK,
			Url:  server.URL,
			Activities: []storepb.Activity_Type{
				storepb.Activity_ISSUE_APPROVED,
				storepb.Activity_ISSUE_SENT_BACK,
			},
		},
	})
	require.NoError(t, err)
	b, err := bus.New()
	require.NoError(t, err)
	service := NewIssueService(stores, webhook.NewManager(stores, &config.Profile{
		ExternalURL: "http://bytebase.example",
	}), b, nil, nil)

	type approvalAction struct {
		name   string
		status storepb.IssuePayloadApproval_Approver_Status
		call   func(string) error
	}
	approve := approvalAction{
		name:   "approve",
		status: storepb.IssuePayloadApproval_Approver_APPROVED,
		call: func(issueName string) error {
			_, err := service.ApproveIssue(ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{Name: issueName}))
			return err
		},
	}
	reject := approvalAction{
		name:   "reject",
		status: storepb.IssuePayloadApproval_Approver_REJECTED,
		call: func(issueName string) error {
			_, err := service.RejectIssue(ctx, connect.NewRequest(&v1pb.RejectIssueRequest{Name: issueName}))
			return err
		},
	}

	for _, test := range []struct {
		name   string
		first  approvalAction
		second approvalAction
	}{
		{name: "approve before reject", first: approve, second: reject},
		{name: "reject before approve", first: reject, second: approve},
	} {
		t.Run(test.name, func(t *testing.T) {
			issue := createPendingNonDatabaseApprovalIssue(ctx, t, stores, test.name)
			issueName := common.FormatIssue(issue.ProjectID, issue.UID)
			results := runIssueCallsBehindUpdateLock(ctx, t, stores, issue, []func() error{
				func() error { return test.first.call(issueName) },
				func() error { return test.second.call(issueName) },
			})

			require.NoError(t, results[0])
			require.Contains(t,
				[]connect.Code{connect.CodeAborted, connect.CodeFailedPrecondition},
				connect.CodeOf(results[1]),
			)
			stored := getIssueForTest(ctx, t, stores, issue.UID)
			require.Len(t, stored.Payload.GetApproval().GetApprovers(), 1)
			require.Equal(t, test.first.status, stored.Payload.GetApproval().GetApprovers()[0].GetStatus())

			comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
				ProjectID: issue.ProjectID,
				IssueUID:  &issue.UID,
			})
			require.NoError(t, err)
			require.Len(t, comments, 1)
			require.Equal(t, test.first.status, comments[0].Payload.GetApproval().GetStatus())
			select {
			case <-requestReceived:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for the winning approval webhook")
			}
			select {
			case <-requestReceived:
				t.Fatal("stale approval action emitted a duplicate webhook")
			case <-time.After(200 * time.Millisecond):
			}
			require.Empty(t, b.RolloutCreationChan)
		})
	}
}

func TestConcurrentRequestIssueOnlyWinningTransitionHasSideEffects(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_APP_IM,
		Workspace: "default",
		Value:     &storepb.AppIMSetting{},
	})
	require.NoError(t, err)
	requestReceived := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestReceived <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	_, err = stores.CreateProjectWebhook(ctx, "project-a", &store.ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhook{
			Type:       storepb.WebhookType_SLACK,
			Url:        server.URL,
			Activities: []storepb.Activity_Type{storepb.Activity_ISSUE_APPROVAL_REQUESTED},
		},
	})
	require.NoError(t, err)
	b, err := bus.New()
	require.NoError(t, err)
	service := NewIssueService(stores, webhook.NewManager(stores, &config.Profile{
		ExternalURL: "http://bytebase.example",
	}), b, nil, nil)
	issue := createPendingNonDatabaseApprovalIssue(ctx, t, stores, "concurrent request")
	rejectedApproval := proto.CloneOf(issue.Payload.GetApproval())
	rejectedApproval.Approvers = []*storepb.IssuePayloadApproval_Approver{{
		Status:    storepb.IssuePayloadApproval_Approver_REJECTED,
		Principal: common.FormatUserEmail("reviewer@example.com"),
	}}
	issue, err = stores.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: rejectedApproval},
	})
	require.NoError(t, err)
	issueName := common.FormatIssue(issue.ProjectID, issue.UID)

	results := runIssueCallsBehindUpdateLock(ctx, t, stores, issue, []func() error{
		func() error {
			_, err := service.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{Name: issueName}))
			return err
		},
		func() error {
			_, err := service.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{Name: issueName}))
			return err
		},
	})

	require.NoError(t, results[0])
	require.Contains(t,
		[]connect.Code{connect.CodeAborted, connect.CodeFailedPrecondition},
		connect.CodeOf(results[1]),
	)
	stored := getIssueForTest(ctx, t, stores, issue.UID)
	require.Empty(t, stored.Payload.GetApproval().GetApprovers())
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: issue.ProjectID,
		IssueUID:  &issue.UID,
	})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	require.Equal(t, storepb.IssuePayloadApproval_Approver_PENDING, comments[0].Payload.GetApproval().GetStatus())
	select {
	case <-requestReceived:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the winning request webhook")
	}
	select {
	case <-requestReceived:
		t.Fatal("stale request action emitted a duplicate webhook")
	case <-time.After(200 * time.Millisecond):
	}
	require.Empty(t, b.RolloutCreationChan)
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
	require.Equal(t, "original title", issues[0].Title)
	require.Equal(t, "original description", issues[0].Description)
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

func TestCreateDraftIssueValidatesEveryPlanKind(t *testing.T) {
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

	changePlan := createPlan("valid change", changeSpec(""))
	createDatabasePlan := createPlan("valid create", createSpec())
	gitOpsPlan := createPlan("gitops", changeSpec("projects/project-a/releases/release"))
	mixedPlan := createPlan("mixed", createSpec(), changeSpec(""))
	exportPlan := createPlan("export", exportSpec())
	unknownPlan := createPlan("unknown", &storepb.PlanConfig_Spec{})

	tests := []struct {
		name      string
		issueType v1pb.Issue_Type
		plan      *store.PlanMessage
		wantError string
	}{
		{
			name:      "create database plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			plan:      createDatabasePlan,
		},
		{
			name:      "change database plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			plan:      changePlan,
		},
		{
			name:      "no plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			wantError: "plan is required",
		},
		{
			name:      "GitOps plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			plan:      gitOpsPlan,
			wantError: "draft issues are not supported for GitOps plans",
		},
		{
			name:      "mixed plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			plan:      mixedPlan,
			wantError: "draft issues are not supported for mixed plans",
		},
		{
			name:      "export plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			plan:      exportPlan,
			wantError: "draft issues are not supported for export plans",
		},
		{
			name:      "unknown plan",
			issueType: v1pb.Issue_DATABASE_CHANGE,
			plan:      unknownPlan,
			wantError: "draft issues require a database plan",
		},
		{
			name:      "non-database issue",
			issueType: v1pb.Issue_ROLE_GRANT,
			plan:      changePlan,
			wantError: "draft issues must be database change issues",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issue := &v1pb.Issue{
				Type:  test.issueType,
				Draft: true,
			}
			if test.plan != nil {
				issue.Plan = common.FormatPlan("project-a", test.plan.UID)
			}
			response, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/project-a",
				Issue:  issue,
			}))
			if test.wantError != "" {
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				var connectErr *connect.Error
				require.ErrorAs(t, err, &connectErr)
				require.Equal(t, test.wantError, connectErr.Message())
				return
			}
			require.NoError(t, err)
			require.True(t, response.Msg.GetDraft())
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

func TestUpdateIssueSubmitsDraftAndRecordsReviewActivity(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, draft := createReadyDraftIssue(ctx, t, stores, service, "submission")

	response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:  common.FormatIssue(draft.ProjectID, draft.UID),
			Draft: false,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.NoError(t, err)
	require.False(t, response.Msg.GetDraft())
	require.Equal(t, "submission", response.Msg.GetTitle())
	require.Len(t, service.bus.ApprovalCheckChan, 1)

	comments, err := service.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: common.FormatIssue(draft.ProjectID, draft.UID),
	}))
	require.NoError(t, err)
	require.Len(t, comments.Msg.GetIssueComments(), 1)
	require.NotNil(t, comments.Msg.GetIssueComments()[0].GetReviewSubmission())

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.NoError(t, err)
	require.Len(t, service.bus.ApprovalCheckChan, 1)
	comments, err = service.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: common.FormatIssue(draft.ProjectID, draft.UID),
	}))
	require.NoError(t, err)
	require.Len(t, comments.Msg.GetIssueComments(), 1)

	direct, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:   v1pb.Issue_DATABASE_CHANGE,
			Plan:   common.FormatPlan("project-a", createReadyReviewPlan(ctx, t, stores, "direct").UID),
			Labels: []string{"team:database"},
		},
	}))
	require.NoError(t, err)
	require.False(t, direct.Msg.GetDraft())
	require.Len(t, service.bus.ApprovalCheckChan, 2)
	directComments, err := service.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: direct.Msg.GetName(),
	}))
	require.NoError(t, err)
	require.Len(t, directComments.Msg.GetIssueComments(), 1)
	require.NotNil(t, directComments.Msg.GetIssueComments()[0].GetReviewSubmission())
	require.NotEqual(t, common.FormatPlan("project-a", plan.UID), direct.Msg.GetPlan())
}

func TestUpdateIssueDraftTransitionIsOneWay(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "owned by plan")

	require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())

	_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.NoError(t, err)

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:  common.FormatIssue(draft.ProjectID, draft.UID),
			Draft: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.False(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
	require.Len(t, service.bus.ApprovalCheckChan, 1)
}

func TestDraftLabelUpdateConflictsWithConcurrentSubmission(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "label submission race")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var lockedUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT id
		FROM issue
		WHERE project = $1 AND id = $2
		FOR UPDATE
	`, draft.ProjectID, draft.UID).Scan(&lockedUID))
	require.Equal(t, draft.UID, lockedUID)

	updateResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue: &v1pb.Issue{
				Name:   common.FormatIssue(draft.ProjectID, draft.UID),
				Labels: []string{"environment:staging"},
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
		}))
		updateResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)

	_, err = tx.ExecContext(ctx, `
		UPDATE issue
		SET payload = payload || jsonb_build_object('draft', false)
		WHERE project = $1 AND id = $2
	`, draft.ProjectID, draft.UID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	require.Equal(t, connect.CodeAborted, connect.CodeOf(receiveTestResult(t, updateResult, "draft label update did not return")))
	stored := getIssueForTest(ctx, t, stores, draft.UID)
	require.False(t, stored.Payload.GetDraft())
	require.Equal(t, []string{"team:database"}, stored.Payload.GetLabels())
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID,
		IssueUID:  &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestDraftSubmissionConflictsWithConcurrentPlanClose(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, draft := createReadyDraftIssue(ctx, t, stores, service, "plan close race")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		UPDATE plan
		SET deleted = true, updated_at = now()
		WHERE project = $1 AND id = $2
	`, plan.ProjectID, plan.UID)
	require.NoError(t, err)

	submissionResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		submissionResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)
	require.NoError(t, tx.Commit())

	submissionErr := receiveTestResult(t, submissionResult, "submission did not return after concurrent Plan close")
	require.Equal(t, connect.CodeAborted, connect.CodeOf(submissionErr), "error: %v", submissionErr)
	stored := getIssueForTest(ctx, t, stores, draft.UID)
	require.True(t, stored.Payload.GetDraft())
	require.Empty(t, service.bus.ApprovalCheckChan)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID,
		IssueUID:  &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestConcurrentDraftSubmissionRunsSideEffectsOnce(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "concurrent submission")

	type result struct {
		response *connect.Response[v1pb.Issue]
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
				Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
			}))
			results <- result{response: response, err: err}
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	for result := range results {
		require.NoError(t, result.err)
		require.NotNil(t, result.response)
		require.False(t, result.response.Msg.GetDraft())
	}
	require.Len(t, service.bus.ApprovalCheckChan, 1)
	comments, err := service.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: common.FormatIssue(draft.ProjectID, draft.UID),
	}))
	require.NoError(t, err)
	require.Len(t, comments.Msg.GetIssueComments(), 1)
	require.NotNil(t, comments.Msg.GetIssueComments()[0].GetReviewSubmission())
}

func TestSubmitDraftRequiresLabelsAndAcceptsLabelsInSameUpdate(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan := createReadyReviewPlan(ctx, t, stores, "labels required")
	draftResponse, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  common.FormatPlan("project-a", plan.UID),
			Draft: true,
		},
	}))
	require.NoError(t, err)
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{ForceIssueLabels: true},
	}))

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: draftResponse.Msg.GetName()},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	setPlanCheckResult(ctx, t, stores, plan, storepb.Advice_SUCCESS)

	submitted, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:   draftResponse.Msg.GetName(),
			Labels: []string{" team:database ", "team:database"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels", "draft"}},
	}))
	require.NoError(t, err)
	require.False(t, submitted.Msg.GetDraft())
	require.Equal(t, []string{"team:database"}, submitted.Msg.GetLabels())
	require.Len(t, service.bus.ApprovalCheckChan, 1)

	createDatabasePlan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "create database labels required",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
					Target:   "instances/prod",
					Database: "app",
				},
			},
		}}},
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
	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: createDatabaseDraft.Msg.GetName()},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	createDatabaseSubmitted, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue: &v1pb.Issue{
			Name:   createDatabaseDraft.Msg.GetName(),
			Labels: []string{"team:database"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels", "draft"}},
	}))
	require.NoError(t, err)
	require.False(t, createDatabaseSubmitted.Msg.GetDraft())
	require.Len(t, service.bus.ApprovalCheckChan, 2)
}

func TestReviewSubmissionRejectsNonCurrentPlanCheckSnapshot(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	for _, test := range []struct {
		name   string
		status store.PlanCheckRunStatus
		stale  bool
		noRun  bool
	}{
		{name: "no row", noRun: true},
		{name: "stale done", status: store.PlanCheckRunStatusDone, stale: true},
		{name: "canceled", status: store.PlanCheckRunStatusCanceled},
		{name: "failed", status: store.PlanCheckRunStatusFailed},
		{name: "available", status: store.PlanCheckRunStatusAvailable},
		{name: "running", status: store.PlanCheckRunStatusRunning},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, draft := createReadyDraftIssue(ctx, t, stores, service, test.name)
			if test.noRun {
				_, err := stores.GetDB().ExecContext(ctx, `
					DELETE FROM plan_check_run
					WHERE project = $1 AND plan_id = $2`,
					plan.ProjectID, plan.UID)
				require.NoError(t, err)
			} else {
				created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
					ProjectID: plan.ProjectID,
					PlanUID:   plan.UID,
					Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()},
				})
				require.NoError(t, err)
				require.True(t, created)
				run, err := stores.GetPlanCheckRun(ctx, plan.ProjectID, plan.UID)
				require.NoError(t, err)
				require.NotNil(t, run)
				if test.status != store.PlanCheckRunStatusAvailable {
					require.NoError(t, stores.UpdatePlanCheckRun(ctx, plan.ProjectID, test.status, &storepb.PlanCheckRunResult{
						ApprovalInputVersion: plan.Config.GetApprovalInputVersion(),
						Results: []*storepb.PlanCheckRunResult_Result{{
							Type:   storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
							Status: storepb.Advice_SUCCESS,
						}},
					}, run.UID))
				}
			}
			if test.stale {
				config := proto.CloneOf(plan.Config)
				config.ApprovalInputVersion++
				_, err := stores.UpdatePlan(ctx, &store.UpdatePlanMessage{
					UID:       plan.UID,
					ProjectID: plan.ProjectID,
					Config:    config,
				})
				require.NoError(t, err)
			}

			_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
				Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
			}))
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
		})
	}
}

func TestReviewSubmissionPlanCheckPolicies(t *testing.T) {
	t.Run("DONE errors allowed without enforcement", func(t *testing.T) {
		ctx := issueServiceTestContext()
		stores := setupIssueServiceTestStore(ctx, t)
		service := newIssueServiceForTest(t, stores)
		plan, draft := createReadyDraftIssue(ctx, t, stores, service, "allowed errors")
		setPlanCheckResult(ctx, t, stores, plan, storepb.Advice_ERROR)

		response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		require.NoError(t, err)
		require.False(t, response.Msg.GetDraft())
	})

	t.Run("require plan check no error rejects summary error", func(t *testing.T) {
		ctx := issueServiceTestContext()
		stores := setupIssueServiceTestStore(ctx, t)
		service := newIssueServiceForTest(t, stores)
		require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
			ResourceID: "project-a",
			Workspace:  "default",
			Setting:    &storepb.Project{RequirePlanCheckNoError: true},
		}))
		plan, draft := createReadyDraftIssue(ctx, t, stores, service, "required checks")
		created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: plan.ProjectID,
			PlanUID:   plan.UID,
			Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()},
		})
		require.NoError(t, err)
		require.True(t, created)
		run, err := stores.GetPlanCheckRun(ctx, plan.ProjectID, plan.UID)
		require.NoError(t, err)
		require.NotNil(t, run)
		require.NoError(t, stores.UpdatePlanCheckRun(ctx, plan.ProjectID, store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
			ApprovalInputVersion: plan.Config.GetApprovalInputVersion(),
			Results: []*storepb.PlanCheckRunResult_Result{{
				Type:   storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
				Status: storepb.Advice_ERROR,
			}},
		}, run.UID))

		_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
	})
}

func TestReviewSubmissionRequiresPlanTitle(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan := createReadyReviewPlan(ctx, t, stores, "   ")
	response, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Title:  "explicit issue title",
			Type:   v1pb.Issue_DATABASE_CHANGE,
			Plan:   common.FormatPlan(plan.ProjectID, plan.UID),
			Labels: []string{"team:database"},
			Draft:  true,
		},
	}))
	require.NoError(t, err)
	setPlanCheckResult(ctx, t, stores, plan, storepb.Advice_SUCCESS)

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: response.Msg.GetName()},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestReviewSubmissionRejectsClosedPlan(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, draft := createReadyDraftIssue(ctx, t, stores, service, "closed plan")
	deleted := true
	_, err := stores.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Deleted:   &deleted,
	})
	require.NoError(t, err)

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
}

func TestDraftSubmissionConflictsWithConcurrentPlanTitleClear(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, draft := createReadyDraftIssue(ctx, t, stores, service, "title race")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		UPDATE plan
		SET name = '', updated_at = updated_at + INTERVAL '1 second'
		WHERE project = $1 AND id = $2`,
		plan.ProjectID, plan.UID)
	require.NoError(t, err)

	submissionResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		submissionResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)
	require.NoError(t, tx.Commit())

	select {
	case err := <-submissionResult:
		require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not return after the Plan title change committed")
	}
	require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
}

func TestReviewSubmissionReusesPlanReadinessAndSQLReviewGates(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{EnforceSqlReview: true},
	}))

	tests := []struct {
		name      string
		spec      *storepb.PlanConfig_Spec
		wantError string
	}{
		{
			name: "missing create target uses spec ID",
			spec: &storepb.PlanConfig_Spec{
				Id: "create-target",
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
						Database: "app",
					},
				},
			},
			wantError: `plan spec "create-target" is missing create target`,
		},
		{
			name: "missing create database name uses index",
			spec: &storepb.PlanConfig_Spec{
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{
						Target: "instances/prod",
					},
				},
			},
			wantError: "plan spec at index 0 is missing create database name",
		},
		{
			name: "missing change targets uses spec ID",
			spec: &storepb.PlanConfig_Spec{
				Id: "change-targets",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						SheetSha256: "sheet",
					},
				},
			},
			wantError: `plan spec "change-targets" is missing change targets`,
		},
		{
			name: "missing sheet uses index",
			spec: &storepb.PlanConfig_Spec{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
					},
				},
			},
			wantError: "plan spec at index 0 is missing sheet",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
				ProjectID: "project-a",
				Name:      test.name,
				Config:    &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{test.spec}},
			}, "creator@example.com")
			require.NoError(t, err)
			draft, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/project-a",
				Issue: &v1pb.Issue{
					Type:   v1pb.Issue_DATABASE_CHANGE,
					Plan:   common.FormatPlan("project-a", plan.UID),
					Labels: []string{"team:database"},
					Draft:  true,
				},
			}))
			require.NoError(t, err)
			_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
				Issue:      &v1pb.Issue{Name: draft.Msg.GetName()},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
			}))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			var connectErr *connect.Error
			require.ErrorAs(t, err, &connectErr)
			require.Equal(t, test.wantError, connectErr.Message())
		})
	}

	t.Run("checks running", func(t *testing.T) {
		plan, draft := createReadyDraftIssue(ctx, t, stores, service, "checks running")
		created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: plan.ProjectID,
			PlanUID:   plan.UID,
			Status:    store.PlanCheckRunStatusAvailable,
			Result:    &storepb.PlanCheckRunResult{},
		})
		require.NoError(t, err)
		require.True(t, created)
		_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("SQL review error", func(t *testing.T) {
		plan, draft := createReadyDraftIssue(ctx, t, stores, service, "review error")
		setPlanCheckResult(ctx, t, stores, plan, storepb.Advice_ERROR)
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
	})

	t.Run("acknowledged warning reaches backend submission", func(t *testing.T) {
		plan, draft := createReadyDraftIssue(ctx, t, stores, service, "review warning")
		setPlanCheckResult(ctx, t, stores, plan, storepb.Advice_WARNING)
		response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		require.NoError(t, err)
		require.False(t, response.Msg.GetDraft())
	})
}

func TestDirectNonDraftIssueCreationPreservesLegacyPlanCheckBehavior(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{EnforceSqlReview: true},
	}))

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "automation plan",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
					Targets: []string{"instances/prod/databases/app"},
				},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: plan.ProjectID,
		PlanUID:   plan.UID,
		Status:    store.PlanCheckRunStatusAvailable,
		Result:    &storepb.PlanCheckRunResult{},
	})
	require.NoError(t, err)
	require.True(t, created)

	issue, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:   v1pb.Issue_DATABASE_CHANGE,
			Plan:   common.FormatPlan("project-a", plan.UID),
			Labels: []string{"team:database"},
		},
	}))
	require.NoError(t, err)
	require.False(t, issue.Msg.GetDraft())
}

func TestReviewSubmissionActivityFailureDoesNotFailSubmission(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "activity failure")

	_, err := stores.GetDB().ExecContext(ctx, `
		CREATE FUNCTION fail_review_submission_activity() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'activity unavailable';
		END;
		$$;
		CREATE TRIGGER fail_review_submission_activity
		BEFORE INSERT ON issue_comment
		FOR EACH ROW EXECUTE FUNCTION fail_review_submission_activity();
	`)
	require.NoError(t, err)

	response, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.NoError(t, err)
	require.False(t, response.Msg.GetDraft())
	require.False(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
	require.Len(t, service.bus.ApprovalCheckChan, 1)
}

func TestReviewSubmissionEmitsIssueCreatedWebhook(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	requestReceived := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestReceived <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	_, err := stores.CreateProjectWebhook(ctx, "project-a", &store.ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhook{
			Type:       storepb.WebhookType_SLACK,
			Url:        server.URL,
			Activities: []storepb.Activity_Type{storepb.Activity_ISSUE_CREATED},
		},
	})
	require.NoError(t, err)
	b, err := bus.New()
	require.NoError(t, err)
	service := NewIssueService(stores, webhook.NewManager(stores, &config.Profile{
		ExternalURL: "http://bytebase.example",
	}), b, nil, nil)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "webhook")

	_, err = service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
		Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
	}))
	require.NoError(t, err)
	select {
	case <-requestReceived:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ISSUE_CREATED webhook")
	}
}

func TestDraftSubmissionConflictsWithConcurrentRequiredLabelRemoval(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, err := stores.GetDB().ExecContext(ctx, `
		UPDATE project
		SET setting = jsonb_build_object('forceIssueLabels', true)
		WHERE resource_id = 'project-a'`)
	require.NoError(t, err)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "required label race")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		UPDATE issue
		SET payload = payload || jsonb_build_object('labels', '[]'::jsonb)
		WHERE project = $1 AND id = $2`,
		draft.ProjectID, draft.UID)
	require.NoError(t, err)

	submissionResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		submissionResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)
	require.NoError(t, tx.Commit())

	select {
	case err := <-submissionResult:
		require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not return after the label removal committed")
	}
	stored := getIssueForTest(ctx, t, stores, draft.UID)
	require.True(t, stored.Payload.GetDraft())
	require.Empty(t, stored.Payload.GetLabels())
	require.Empty(t, service.bus.ApprovalCheckChan)
	require.Empty(t, service.bus.RolloutCreationChan)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID,
		IssueUID:  &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestBatchUpdateIssuesStatusRejectsDraftIssue(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "draft status")

	_, err := service.BatchUpdateIssuesStatus(ctx, connect.NewRequest(&v1pb.BatchUpdateIssuesStatusRequest{
		Issues: []string{common.FormatIssue(draft.ProjectID, draft.UID)},
		Status: v1pb.IssueStatus_CANCELED,
		Reason: "cancel draft",
	}))

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	stored := getIssueForTest(ctx, t, stores, draft.UID)
	require.True(t, stored.Payload.GetDraft())
	require.Equal(t, storepb.Issue_OPEN, stored.Status)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID,
		IssueUID:  &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
	require.Empty(t, service.bus.ApprovalCheckChan)
	require.Empty(t, service.bus.RolloutCreationChan)
}

func TestDraftSubmissionConflictsWithConcurrentStatusChange(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	_, draft := createReadyDraftIssue(ctx, t, stores, service, "status race")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		UPDATE issue
		SET status = 'CANCELED'
		WHERE project = $1 AND id = $2`,
		draft.ProjectID, draft.UID)
	require.NoError(t, err)

	submissionResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		submissionResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)
	require.NoError(t, tx.Commit())

	select {
	case err := <-submissionResult:
		require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not return after the status change committed")
	}
	stored := getIssueForTest(ctx, t, stores, draft.UID)
	require.True(t, stored.Payload.GetDraft())
	require.Equal(t, storepb.Issue_CANCELED, stored.Status)
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID,
		IssueUID:  &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
	require.Empty(t, service.bus.ApprovalCheckChan)
	require.Empty(t, service.bus.RolloutCreationChan)
}

func TestDraftSubmissionConflictsWithConcurrentPlanCheckRerun(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)
	plan, draft := createReadyDraftIssue(ctx, t, stores, service, "completed check rerun")

	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var lockedPlanUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT id
		FROM plan
		WHERE project = $1 AND id = $2
		FOR UPDATE`,
		plan.ProjectID, plan.UID,
	).Scan(&lockedPlanUID))
	_, err = tx.ExecContext(ctx, `
		UPDATE plan_check_run
		SET status = $3, updated_at = updated_at + INTERVAL '1 second'
		WHERE project = $1 AND plan_id = $2`,
		plan.ProjectID, plan.UID, store.PlanCheckRunStatusAvailable)
	require.NoError(t, err)

	submissionResult := make(chan error, 1)
	go func() {
		_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue:      &v1pb.Issue{Name: common.FormatIssue(draft.ProjectID, draft.UID)},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"draft"}},
		}))
		submissionResult <- err
	}()

	waitForTransactionBlock(ctx, t, stores.GetDB(), tx)
	require.NoError(t, tx.Commit())

	select {
	case err := <-submissionResult:
		require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not return after the Plan check rerun committed")
	}
	require.True(t, getIssueForTest(ctx, t, stores, draft.UID).Payload.GetDraft())
	comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID: draft.ProjectID,
		IssueUID:  &draft.UID,
	})
	require.NoError(t, err)
	require.Empty(t, comments)
	require.Empty(t, service.bus.ApprovalCheckChan)
	require.Empty(t, service.bus.RolloutCreationChan)
}

func issueNames(issues []*v1pb.Issue) []string {
	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		names = append(names, issue.GetName())
	}
	return names
}

func TestUpdateDraftIssueMetadataMustUsePlan(t *testing.T) {
	ctx := issueServiceTestContext()
	stores := setupIssueServiceTestStore(ctx, t)
	service := newIssueServiceForTest(t, stores)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan-owned title",
		Config:    &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        plan.Name,
		Description:  "plan-owned description",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	for _, path := range []string{"title", "description"} {
		t.Run(path, func(t *testing.T) {
			_, err := service.UpdateIssue(ctx, connect.NewRequest(&v1pb.UpdateIssueRequest{
				Issue: &v1pb.Issue{
					Name:        common.FormatIssue(issue.ProjectID, issue.UID),
					Title:       "direct issue title",
					Description: "direct issue description",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{path}},
			}))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}

	stored, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{issue.ProjectID},
		UID:        &issue.UID,
	})
	require.NoError(t, err)
	require.Equal(t, plan.Name, stored.Title)
	require.Equal(t, "plan-owned description", stored.Description)
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

func createReadyReviewPlan(ctx context.Context, t *testing.T, stores *store.Store, title string) *store.PlanMessage {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      title,
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets:     []string{"instances/prod/databases/app"},
						SheetSha256: "sheet",
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	return plan
}

func createReadyDraftIssue(
	ctx context.Context,
	t *testing.T,
	stores *store.Store,
	service *IssueService,
	title string,
) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	plan := createReadyReviewPlan(ctx, t, stores, title)
	response, err := service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Type:   v1pb.Issue_DATABASE_CHANGE,
			Plan:   common.FormatPlan("project-a", plan.UID),
			Labels: []string{"team:database"},
			Draft:  true,
		},
	}))
	require.NoError(t, err)
	_, issueUID, err := common.GetProjectIDIssueUID(response.Msg.GetName())
	require.NoError(t, err)
	setPlanCheckResult(ctx, t, stores, plan, storepb.Advice_SUCCESS)
	return plan, getIssueForTest(ctx, t, stores, issueUID)
}

func setPlanCheckResult(
	ctx context.Context,
	t *testing.T,
	stores *store.Store,
	plan *store.PlanMessage,
	status storepb.Advice_Status,
) {
	t.Helper()
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: plan.ProjectID,
		PlanUID:   plan.UID,
		Status:    store.PlanCheckRunStatusAvailable,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()},
	})
	require.NoError(t, err)
	require.True(t, created)
	run, err := stores.GetPlanCheckRun(ctx, plan.ProjectID, plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.NoError(t, stores.UpdatePlanCheckRun(ctx, plan.ProjectID, store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: plan.Config.GetApprovalInputVersion(),
		Results: []*storepb.PlanCheckRunResult_Result{{
			Type:   storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
			Status: status,
		}},
	}, run.UID))
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

func createPendingNonDatabaseApprovalIssue(
	ctx context.Context,
	t *testing.T,
	stores *store.Store,
	title string,
) *store.IssueMessage {
	t.Helper()
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        title,
		Type:         storepb.Issue_DATABASE_EXPORT,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow:  &storepb.ApprovalFlow{Roles: []string{"roles/workspaceAdmin"}},
					Title: "manual approval",
				},
				ApprovalFindingDone: true,
			},
		},
	})
	require.NoError(t, err)
	return issue
}

func runIssueCallsBehindUpdateLock(
	ctx context.Context,
	t *testing.T,
	stores *store.Store,
	issue *store.IssueMessage,
	calls []func() error,
) []error {
	t.Helper()
	stores.GetDB().SetMaxOpenConns(10)
	tx, err := stores.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var lockedUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT id
		FROM issue
		WHERE project = $1 AND id = $2
		FOR UPDATE`,
		issue.ProjectID, issue.UID,
	).Scan(&lockedUID))
	require.Equal(t, issue.UID, lockedUID)

	results := make([]chan error, len(calls))
	for i, call := range calls {
		results[i] = make(chan error, 1)
		go func(result chan<- error, call func() error) {
			result <- call()
		}(results[i], call)

		waitForBlockedSessionCount(ctx, t, stores.GetDB(), i+1)
	}
	require.NoError(t, tx.Commit())

	errs := make([]error, len(results))
	for i, result := range results {
		select {
		case errs[i] = <-result:
		case <-time.After(5 * time.Second):
			t.Fatalf("approval action %d did not return after the Issue lock was released", i)
		}
	}
	return errs
}

func waitForTransactionBlock(ctx context.Context, t *testing.T, db *sql.DB, tx *sql.Tx) {
	t.Helper()
	waitForTransactionBlockCount(ctx, t, db, tx, 1)
}

func waitForTransactionBlockCount(ctx context.Context, t *testing.T, db *sql.DB, tx *sql.Tx, minimum int) {
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
			  AND $1 = ANY(pg_blocking_pids(activity.pid))
		`, blockerPID).Scan(&waiting))
		if waiting >= minimum {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d session(s) blocked by transaction PID %d", minimum, blockerPID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForBlockedSessionCount(ctx context.Context, t *testing.T, db *sql.DB, minimum int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT count(*)
			FROM pg_stat_activity
			WHERE datname = current_database()
			  AND pid <> pg_backend_pid()
			  AND wait_event_type = 'Lock'
		`).Scan(&waiting))
		if waiting >= minimum {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d blocked session(s)", minimum)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func receiveTestResult[T any](t *testing.T, result <-chan T, timeoutMessage string) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal(timeoutMessage)
		var zero T
		return zero
	}
}
