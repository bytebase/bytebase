package taskrun

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestPlanCheckRunBlocksRolloutIgnoresStaleVersion(t *testing.T) {
	plan := &store.PlanMessage{
		Config: &storepb.PlanConfig{ApprovalInputVersion: 2},
	}

	require.False(t, planCheckRunBlocksRollout(plan, nil))
	require.False(t, planCheckRunBlocksRollout(plan, &store.PlanCheckRunMessage{
		Status: store.PlanCheckRunStatusRunning,
		Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	}))
	require.True(t, planCheckRunBlocksRollout(plan, &store.PlanCheckRunMessage{
		Status: store.PlanCheckRunStatusRunning,
		Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	}))
}

func TestTryCreateRolloutSkipsDraft(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutCreatorStore(ctx, t)
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}))

	environment := "prod"
	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID:    "prod",
		Workspace:     "default",
		EnvironmentID: &environment,
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "project-a",
		InstanceID:   "prod",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft plan",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "change",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{
			Draft: true,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	NewRolloutCreator(s, b, nil, false).tryCreateRollout(ctx, bus.PlanRef{
		ProjectID: plan.ProjectID,
		PlanID:    plan.UID,
	})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: plan.ProjectID,
		UID:       &plan.UID,
	})
	require.NoError(t, err)
	require.NotNil(t, gotPlan)
	require.False(t, gotPlan.Config.GetHasRollout())

	tasks, err := s.ListTasks(ctx, &store.TaskFind{
		ProjectID: plan.ProjectID,
		PlanID:    &plan.UID,
	})
	require.NoError(t, err)
	require.Empty(t, tasks)

	gotIssue, err := s.GetIssue(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{issue.ProjectID},
		PlanUID:    &plan.UID,
	})
	require.NoError(t, err)
	require.NotNil(t, gotIssue)
	require.True(t, gotIssue.Payload.GetDraft())
	require.Equal(t, storepb.Issue_OPEN, gotIssue.Status)
	require.Empty(t, b.TaskRunPendingTickleChan)
}

func TestTryCreateRolloutSkipsArchivedProject(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutCreatorStore(ctx, t)
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}))

	environment := "prod"
	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID:    "prod",
		Workspace:     "default",
		EnvironmentID: &environment,
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "project-a",
		InstanceID:   "prod",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "archived project plan",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "change",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	_, err = s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "archived project issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	archived := true
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Delete:     &archived,
	}))

	b, err := bus.New()
	require.NoError(t, err)
	NewRolloutCreator(s, b, nil, false).tryCreateRollout(ctx, bus.PlanRef{
		ProjectID: plan.ProjectID,
		PlanID:    plan.UID,
	})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: plan.ProjectID, UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, gotPlan.Config.GetHasRollout())

	tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: plan.ProjectID, PlanID: &plan.UID})
	require.NoError(t, err)
	require.Empty(t, tasks)
	require.Empty(t, b.TaskRunPendingTickleChan)
}

func TestTryCreateRolloutResolvesProjectInstanceOutsideRequestContext(t *testing.T) {
	ctx := context.Background()
	const workspaceID = "workspace-a"
	s := setupRolloutCreatorStoreInWorkspace(ctx, t, workspaceID)
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  workspaceID,
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}))

	environment := "prod"
	projectID := "project-a"
	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID:    "prod",
		Workspace:     workspaceID,
		ProjectID:     &projectID,
		EnvironmentID: &environment,
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    projectID,
		InstanceID:   "prod",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: projectID,
		Name:      "auto rollout plan",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "change",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"projects/project-a/instances/prod/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	_, err = s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    projectID,
		CreatorEmail: "creator@example.com",
		Title:        "auto rollout issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	NewRolloutCreator(s, b, nil, false).tryCreateRollout(ctx, bus.PlanRef{ProjectID: projectID, PlanID: plan.UID})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, gotPlan.Config.GetHasRollout())
	tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectID, PlanID: &plan.UID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Len(t, b.TaskRunPendingTickleChan, 1)
}

func TestTryCreateRolloutRejectsArchivedProjectInstanceWithWarmDatabaseCache(t *testing.T) {
	ctx := context.Background()
	const workspaceID = "workspace-a"
	s := setupRolloutCreatorStoreInWorkspaceWithCache(ctx, t, workspaceID, true)
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  workspaceID,
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}))

	environment := "prod"
	projectID := "project-a"
	instance, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID:    "prod",
		Workspace:     workspaceID,
		ProjectID:     &projectID,
		EnvironmentID: &environment,
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    projectID,
		InstanceID:   instance.ResourceID,
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: projectID,
		Name:      "archived instance rollout plan",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "change",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"projects/project-a/instances/prod/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	_, err = s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    projectID,
		CreatorEmail: "creator@example.com",
		Title:        "archived instance rollout issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	instanceID := instance.ResourceID
	databaseName := "app"
	warmed, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    workspaceID,
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, warmed)
	deleted := true
	_, err = s.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instanceID,
		Workspace:  workspaceID,
		Deleted:    &deleted,
	})
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	NewRolloutCreator(s, b, nil, false).tryCreateRollout(ctx, bus.PlanRef{ProjectID: projectID, PlanID: plan.UID})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, gotPlan.Config.GetHasRollout())
	tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectID, PlanID: &plan.UID})
	require.NoError(t, err)
	require.Empty(t, tasks)
	require.Empty(t, b.TaskRunPendingTickleChan)
}

func TestTryCreateRolloutWaitsForTheReview(t *testing.T) {
	ctx := context.Background()
	s, plan, issue := newAutoRolloutIssue(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	creator := NewRolloutCreator(s, b, nil, true)
	tryCreateRollout := func() bool {
		t.Helper()
		creator.tryCreateRollout(ctx, bus.PlanRef{ProjectID: plan.ProjectID, PlanID: plan.UID})
		got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: plan.ProjectID, UID: &plan.UID})
		require.NoError(t, err)
		return got.Config.GetHasRollout()
	}

	// An open thread on the colliding issue of another project must not hold
	// this one.
	_, err = s.GetDB().ExecContext(ctx, "INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', $1, 'Project B')", autoRolloutWorkspaceID)
	require.NoError(t, err)
	planB, err := s.CreatePlan(ctx, &store.PlanMessage{ProjectID: "project-b", Name: "colliding plan", Config: &storepb.PlanConfig{}}, "creator@example.com")
	require.NoError(t, err)
	issueB, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-b",
		CreatorEmail: "creator@example.com",
		Title:        "colliding issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &planB.UID,
	})
	require.NoError(t, err)
	require.Equal(t, issue.UID, issueB.UID, "issue ids should collide")
	open := store.ThreadStateOpen
	_, err = s.CreateIssueComments(ctx, "creator@example.com", &store.IssueCommentMessage{
		ProjectID:   "project-b",
		IssueUID:    issueB.UID,
		Payload:     &storepb.IssueCommentPayload{Comment: "Is this safe?"},
		ThreadState: &open,
	})
	require.NoError(t, err)

	_, err = s.CreateReviewRun(ctx, issue.ProjectID, issue.UID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.False(t, tryCreateRollout(), "the rule review has not finished")

	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err := s.CompleteReviewRun(ctx, claimed[0], "replica", storepb.ReviewRun_DONE, &storepb.ReviewRunPayload{}, []*store.IssueCommentMessage{{
		ProjectID: issue.ProjectID,
		IssueUID:  issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: "UPDATE without WHERE",
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
				RunType:  storepb.ReviewRun_RULE,
				RuleType: storepb.ReviewRuleType_REQUIRE_WHERE,
				Priority: storepb.IssueCommentPayload_ReviewMetadata_P1,
			},
		},
	}})
	require.NoError(t, err)
	require.True(t, updated)
	require.False(t, tryCreateRollout(), "the finding's thread is open")

	threads, err := s.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: issue.ProjectID, IssueUID: &issue.UID, ThreadState: &open})
	require.NoError(t, err)
	require.Len(t, threads, 1)
	resolved := store.ThreadStateResolved
	require.NoError(t, s.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{
		ProjectID:   issue.ProjectID,
		ResourceID:  threads[0].ResourceID,
		ThreadState: &resolved,
	}))
	require.True(t, tryCreateRollout(), "every thread is resolved")
}

func TestTryCreateRolloutStartsAMissingOrStaleReview(t *testing.T) {
	ctx := context.Background()
	s, plan, issue := newAutoRolloutIssue(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	creator := NewRolloutCreator(s, b, nil, true)
	tryCreateRollout := func() bool {
		t.Helper()
		creator.tryCreateRollout(ctx, bus.PlanRef{ProjectID: plan.ProjectID, PlanID: plan.UID})
		got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: plan.ProjectID, UID: &plan.UID})
		require.NoError(t, err)
		return got.Config.GetHasRollout()
	}
	ruleRun := func(projectID string) *store.ReviewRunMessage {
		t.Helper()
		run, err := s.GetReviewRun(ctx, projectID, issue.UID, store.ReviewRunTypeRule)
		require.NoError(t, err)
		return run
	}
	completeRuleRun := func() {
		t.Helper()
		claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica")
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		updated, err := s.CompleteReviewRun(ctx, claimed[0], "replica", storepb.ReviewRun_DONE, &storepb.ReviewRunPayload{}, nil)
		require.NoError(t, err)
		require.True(t, updated)
	}

	// The colliding issue of another project must not get a review.
	_, err = s.GetDB().ExecContext(ctx, "INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', $1, 'Project B')", autoRolloutWorkspaceID)
	require.NoError(t, err)
	planB, err := s.CreatePlan(ctx, &store.PlanMessage{ProjectID: "project-b", Name: "colliding plan", Config: &storepb.PlanConfig{}}, "creator@example.com")
	require.NoError(t, err)
	issueB, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-b",
		CreatorEmail: "creator@example.com",
		Title:        "colliding issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &planB.UID,
	})
	require.NoError(t, err)
	require.Equal(t, issue.UID, issueB.UID, "issue ids should collide")

	require.False(t, tryCreateRollout(), "the review never started")
	run := ruleRun(issue.ProjectID)
	require.NotNil(t, run)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)
	require.Equal(t, int64(2), run.Payload.GetApprovalInputVersion())
	require.Len(t, b.ReviewRunTickleChan, 1)
	require.Nil(t, ruleRun("project-b"))

	// The plan's SQL changes and is approved again, but the slot was not
	// reset: the DONE run reviewed version 2.
	completeRuleRun()
	_, err = s.GetDB().ExecContext(ctx, `
		UPDATE plan SET config = jsonb_set(config, '{approvalInputVersion}', '"3"')
		WHERE project = $1 AND id = $2
	`, plan.ProjectID, plan.UID)
	require.NoError(t, err)
	_, err = s.GetDB().ExecContext(ctx, `
		UPDATE issue SET payload = jsonb_set(payload, '{approval,approvalInputVersion}', '"3"')
		WHERE project = $1 AND id = $2
	`, issue.ProjectID, issue.UID)
	require.NoError(t, err)
	require.False(t, tryCreateRollout(), "the done run reviewed older SQL")
	run = ruleRun(issue.ProjectID)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)
	require.Equal(t, int64(1), run.Attempt)
	require.Equal(t, int64(3), run.Payload.GetApprovalInputVersion())

	completeRuleRun()
	require.True(t, tryCreateRollout(), "the review of the current SQL is done")
}

func TestReviewRestartsWhenThePlanLosesItsSQL(t *testing.T) {
	ctx := context.Background()
	s, plan, issue := newAutoRolloutIssue(ctx, t)
	project, err := s.GetProjectByResourceID(ctx, plan.ProjectID)
	require.NoError(t, err)
	b, err := bus.New()
	require.NoError(t, err)
	creator := NewRolloutCreator(s, b, nil, true)

	// The review of version 2 left an open finding.
	_, err = s.CreateReviewRun(ctx, issue.ProjectID, issue.UID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	claimed, err := s.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err := s.CompleteReviewRun(ctx, claimed[0], "replica", storepb.ReviewRun_DONE, &storepb.ReviewRunPayload{}, []*store.IssueCommentMessage{{
		ProjectID: issue.ProjectID,
		IssueUID:  issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: "UPDATE without WHERE",
			ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
				RunType:  storepb.ReviewRun_RULE,
				RuleType: storepb.ReviewRuleType_REQUIRE_WHERE,
				Priority: storepb.IssueCommentPayload_ReviewMetadata_P1,
			},
		},
	}})
	require.NoError(t, err)
	require.True(t, updated)

	// Version 3 has nothing to review.
	_, err = s.GetDB().ExecContext(ctx, `
		UPDATE plan SET config = jsonb_set(config, '{approvalInputVersion}', '"3"')
		WHERE project = $1 AND id = $2
	`, plan.ProjectID, plan.UID)
	require.NoError(t, err)
	plan, err = s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: plan.ProjectID, UID: &plan.UID})
	require.NoError(t, err)
	plan.Config.Specs[0].GetChangeDatabaseConfig().Targets = nil

	blocked, err := creator.reviewBlocksRollout(ctx, project, plan, issue)
	require.NoError(t, err)
	require.True(t, blocked)
	run, err := s.GetReviewRun(ctx, issue.ProjectID, issue.UID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Equal(t, storepb.ReviewRun_AVAILABLE, run.Status)
	require.Equal(t, int64(3), run.Payload.GetApprovalInputVersion())

	// The restarted run finds nothing, which supersedes the earlier finding.
	claimed, err = s.ClaimAvailableReviewRuns(ctx, "replica")
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	updated, err = s.CompleteReviewRun(ctx, claimed[0], "replica", storepb.ReviewRun_DONE, &storepb.ReviewRunPayload{}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	blocked, err = creator.reviewBlocksRollout(ctx, project, plan, issue)
	require.NoError(t, err)
	require.False(t, blocked)
}

func TestReviewDoesNotHoldARolloutWithNothingToRead(t *testing.T) {
	ctx := context.Background()
	s, plan, issue := newAutoRolloutIssue(ctx, t)
	project, err := s.GetProjectByResourceID(ctx, plan.ProjectID)
	require.NoError(t, err)
	plan.Config.Specs[0].GetChangeDatabaseConfig().Targets = nil

	b, err := bus.New()
	require.NoError(t, err)
	blocked, err := NewRolloutCreator(s, b, nil, true).reviewBlocksRollout(ctx, project, plan, issue)
	require.NoError(t, err)
	require.False(t, blocked)
	run, err := s.GetReviewRun(ctx, issue.ProjectID, issue.UID, store.ReviewRunTypeRule)
	require.NoError(t, err)
	require.Nil(t, run, "no review starts without SQL to read")
}

func TestTryCreateRolloutIgnoresThreadsWithoutSQLReviewV2(t *testing.T) {
	ctx := context.Background()
	s, plan, issue := newAutoRolloutIssue(ctx, t)
	open := store.ThreadStateOpen
	_, err := s.CreateIssueComments(ctx, "creator@example.com", &store.IssueCommentMessage{
		ProjectID:   issue.ProjectID,
		IssueUID:    issue.UID,
		Payload:     &storepb.IssueCommentPayload{Comment: "Is this safe?"},
		ThreadState: &open,
	})
	require.NoError(t, err)

	b, err := bus.New()
	require.NoError(t, err)
	NewRolloutCreator(s, b, nil, false).tryCreateRollout(ctx, bus.PlanRef{ProjectID: plan.ProjectID, PlanID: plan.UID})
	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: plan.ProjectID, UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())
}

const autoRolloutWorkspaceID = "workspace-a"

// newAutoRolloutIssue sets up a store with an approved issue in project-a whose
// plan the rollout creator would roll out at once.
func newAutoRolloutIssue(ctx context.Context, t *testing.T) (*store.Store, *store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	s := setupRolloutCreatorStoreInWorkspace(ctx, t, autoRolloutWorkspaceID)
	require.NoError(t, s.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  autoRolloutWorkspaceID,
		Setting:    &storepb.Project{RequireIssueApproval: false},
	}))
	environment := "prod"
	projectID := "project-a"
	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID:    "prod",
		Workspace:     autoRolloutWorkspaceID,
		ProjectID:     &projectID,
		EnvironmentID: &environment,
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    projectID,
		InstanceID:   "prod",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: projectID,
		Name:      "auto rollout plan",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "change",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"projects/project-a/instances/prod/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    projectID,
		CreatorEmail: "creator@example.com",
		Title:        "auto rollout issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)
	return s, plan, issue
}

func setupRolloutCreatorStore(ctx context.Context, t *testing.T) *store.Store {
	return setupRolloutCreatorStoreInWorkspace(ctx, t, "default")
}

func setupRolloutCreatorStoreInWorkspace(ctx context.Context, t *testing.T, workspaceID string) *store.Store {
	return setupRolloutCreatorStoreInWorkspaceWithCache(ctx, t, workspaceID, false)
}

func setupRolloutCreatorStoreInWorkspaceWithCache(ctx context.Context, t *testing.T, workspaceID string, enableCache bool) *store.Store {
	t.Helper()

	db, s, _ := testcontainer.NewMetadataDBWithCache(t, enableCache)

	_, err := db.ExecContext(ctx, "INSERT INTO workspace (resource_id) VALUES ($1)", workspaceID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused')")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', $1, 'Project A')", workspaceID)
	require.NoError(t, err)

	return s
}
