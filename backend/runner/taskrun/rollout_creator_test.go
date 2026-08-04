package taskrun

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
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
	NewRolloutCreator(s, b, nil).tryCreateRollout(ctx, bus.PlanRef{
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
	require.Empty(t, b.TaskRunTickleChan)
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
	NewRolloutCreator(s, b, nil).tryCreateRollout(ctx, bus.PlanRef{
		ProjectID: plan.ProjectID,
		PlanID:    plan.UID,
	})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: plan.ProjectID, UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, gotPlan.Config.GetHasRollout())

	tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: plan.ProjectID, PlanID: &plan.UID})
	require.NoError(t, err)
	require.Empty(t, tasks)
	require.Empty(t, b.TaskRunTickleChan)
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
	NewRolloutCreator(s, b, nil).tryCreateRollout(ctx, bus.PlanRef{ProjectID: projectID, PlanID: plan.UID})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, gotPlan.Config.GetHasRollout())
	tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectID, PlanID: &plan.UID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Len(t, b.TaskRunTickleChan, 1)
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
	NewRolloutCreator(s, b, nil).tryCreateRollout(ctx, bus.PlanRef{ProjectID: projectID, PlanID: plan.UID})

	gotPlan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, gotPlan.Config.GetHasRollout())
	tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectID, PlanID: &plan.UID})
	require.NoError(t, err)
	require.Empty(t, tasks)
	require.Empty(t, b.TaskRunTickleChan)
}

func setupRolloutCreatorStore(ctx context.Context, t *testing.T) *store.Store {
	return setupRolloutCreatorStoreInWorkspace(ctx, t, "default")
}

func setupRolloutCreatorStoreInWorkspace(ctx context.Context, t *testing.T, workspaceID string) *store.Store {
	return setupRolloutCreatorStoreInWorkspaceWithCache(ctx, t, workspaceID, false)
}

func setupRolloutCreatorStoreInWorkspaceWithCache(ctx context.Context, t *testing.T, workspaceID string, enableCache bool) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, "INSERT INTO workspace (resource_id) VALUES ($1)", workspaceID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused')")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', $1, 'Project A')", workspaceID)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, enableCache)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
