package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestCreateRolloutTasksRequiresMatchingApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutReviewStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan-a",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	updated, tasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.RolloutGuard{ApprovalInputVersion: 1}, nil)
	require.NoError(t, err)
	require.False(t, updated)
	require.Empty(t, tasks)

	updated, tasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.RolloutGuard{ApprovalInputVersion: 2}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	require.Empty(t, tasks)
}

func TestCreateRolloutTasksRequiresCurrentIssueApproval(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutReviewStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan-a",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	approval := &storepb.IssuePayloadApproval{
		ApprovalFindingDone:  true,
		ApprovalInputVersion: 2,
		ApprovalTemplate: &storepb.ApprovalTemplate{
			Flow: &storepb.ApprovalFlow{Roles: []string{"roles/sql-reviewer"}},
		},
		Approvers: []*storepb.IssuePayloadApproval_Approver{{
			Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
			Principal: "users/reviewer@example.com",
		}},
	}
	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Approval: approval},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)
	observed := proto.CloneOf(approval)

	replacement := proto.CloneOf(approval)
	replacement.Approvers[0].Principal = "users/other-reviewer@example.com"
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Approval: replacement},
	})
	require.NoError(t, err)

	updated, tasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.RolloutGuard{
		IssueUID:             issue.UID,
		ApprovalInputVersion: 2,
		Approval:             observed,
	}, nil)
	require.NoError(t, err)
	require.False(t, updated)
	require.Empty(t, tasks)

	updated, tasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.RolloutGuard{
		IssueUID:             issue.UID,
		ApprovalInputVersion: 2,
		Approval:             replacement,
	}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	require.Empty(t, tasks)
}

func TestCreateRolloutTasksAddsMissingTasksAfterRolloutExists(t *testing.T) {
	ctx := context.Background()
	s := setupRolloutReviewStore(ctx, t)
	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan-a",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	updated, tasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, nil, []*store.TaskMessage{
		newRolloutReviewTask("instance-a", "database-a", "sheet-a"),
	})
	require.NoError(t, err)
	require.True(t, updated)
	require.Len(t, tasks, 1)

	updated, tasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, nil, []*store.TaskMessage{
		newRolloutReviewTask("instance-a", "database-a", "sheet-a"),
		newRolloutReviewTask("instance-a", "database-b", "sheet-b"),
	})
	require.NoError(t, err)
	require.True(t, updated)
	require.Len(t, tasks, 1)
	require.Equal(t, "database-b", tasks[0].GetDatabaseName())
}

func newRolloutReviewTask(instanceID, databaseName, sheetSHA256 string) *store.TaskMessage {
	return &store.TaskMessage{
		InstanceID:   instanceID,
		DatabaseName: &databaseName,
		Type:         storepb.Task_DATABASE_MIGRATE,
		Payload: &storepb.Task{
			Source: &storepb.Task_SheetSha256{SheetSha256: sheetSHA256},
		},
	}
}

func setupRolloutReviewStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
