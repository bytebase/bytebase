package v1

import (
	"context"
	"fmt"
	"sync"
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
			Type:    v1pb.Issue_DATABASE_CHANGE,
			Plan:    common.FormatPlan("project-a", plan.UID),
			IsDraft: true,
		},
	}))
	require.NoError(t, err)
	require.Equal(t, common.FormatPlan("project-a", plan.UID), response.Msg.Plan)
	require.Equal(t, "draft plan", response.Msg.Title)
	require.Equal(t, "draft description", response.Msg.Description)
	require.True(t, response.Msg.IsDraft)

	stored, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{"project-a"},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, err)
	require.True(t, stored.Payload.GetIsDraft())

	var rawIsDraft string
	err = stores.GetDB().QueryRowContext(
		ctx,
		"SELECT payload->>'isDraft' FROM issue WHERE project = $1 AND id = $2",
		"project-a",
		stored.UID,
	).Scan(&rawIsDraft)
	require.NoError(t, err)
	require.Equal(t, "true", rawIsDraft)

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
			Type:    v1pb.Issue_DATABASE_CHANGE,
			Plan:    common.FormatPlan("project-a", createDatabasePlan.UID),
			IsDraft: true,
		},
	}))
	require.NoError(t, err)
	require.True(t, createDatabaseDraft.Msg.GetIsDraft())
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
			IsDraft:     true,
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
			IsDraft:     true,
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
					Title:   title,
					Type:    v1pb.Issue_DATABASE_CHANGE,
					Plan:    common.FormatPlan("project-a", concurrentPlan.UID),
					IsDraft: true,
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
	require.False(t, existing.Payload.GetIsDraft())

	var hasIsDraft bool
	err = stores.GetDB().QueryRowContext(
		ctx,
		"SELECT payload ? 'isDraft' FROM issue WHERE project = $1 AND id = $2",
		"project-a",
		existing.UID,
	).Scan(&hasIsDraft)
	require.NoError(t, err)
	require.False(t, hasIsDraft)

	_, err = service.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: "projects/project-a",
		Issue: &v1pb.Issue{
			Title:   "draft replacement",
			Type:    v1pb.Issue_DATABASE_CHANGE,
			Plan:    common.FormatPlan("project-a", plan.UID),
			IsDraft: true,
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
	require.False(t, stored.Payload.GetIsDraft())

	direct, err := service.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
		Name: common.FormatIssue("project-a", existing.UID),
	}))
	require.NoError(t, err)
	require.False(t, direct.Msg.GetIsDraft())
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
				Type:    v1pb.Issue_DATABASE_CHANGE,
				IsDraft: true,
			},
		},
		{
			name: "GitOps plan",
			issue: &v1pb.Issue{
				Type:    v1pb.Issue_DATABASE_CHANGE,
				Plan:    common.FormatPlan("project-a", gitOpsPlan.UID),
				IsDraft: true,
			},
		},
		{
			name: "mixed plan",
			issue: &v1pb.Issue{
				Type:    v1pb.Issue_DATABASE_CHANGE,
				Plan:    common.FormatPlan("project-a", mixedPlan.UID),
				IsDraft: true,
			},
		},
		{
			name: "export plan",
			issue: &v1pb.Issue{
				Type:    v1pb.Issue_DATABASE_CHANGE,
				Plan:    common.FormatPlan("project-a", exportPlan.UID),
				IsDraft: true,
			},
		},
		{
			name: "non-database issue",
			issue: &v1pb.Issue{
				Title:   "role request",
				Type:    v1pb.Issue_ROLE_GRANT,
				Plan:    common.FormatPlan("project-a", validPlan.UID),
				IsDraft: true,
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
		Payload:      &storepb.Issue{IsDraft: true},
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
	require.True(t, direct.Msg.GetIsDraft())
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
	return NewIssueService(stores, nil, b, nil, nil)
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
