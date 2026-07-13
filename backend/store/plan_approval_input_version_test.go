package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestUpdatePlanBumpsApprovalInputVersionOnlyWhenRequested(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	created, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	require.EqualValues(t, 0, created.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, created.Config, "spec-a")

	config := &storepb.PlanConfig{
		ApprovalInputVersion: 99,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-b"}},
	}
	updated, err := s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      created.UID,
		ProjectID:                created.ProjectID,
		Config:                   config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, updated.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, updated.Config, "spec-b")

	description := "description-only"
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:         created.UID,
		ProjectID:   created.ProjectID,
		Description: &description,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, updated.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, updated.Config, "spec-b")

	config = &storepb.PlanConfig{
		ApprovalInputVersion: 99,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-c"}},
	}
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      created.UID,
		ProjectID:                created.ProjectID,
		Config:                   config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, updated.Config, "spec-c")

	config = &storepb.PlanConfig{
		ApprovalInputVersion: 7,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-d"}},
	}
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       created.UID,
		ProjectID: created.ProjectID,
		Config:    config,
	})
	require.NoError(t, err)
	require.EqualValues(t, 7, updated.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, updated.Config, "spec-d")

	config = &storepb.PlanConfig{
		ApprovalInputVersion: 99,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-e"}},
	}
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      created.UID,
		ProjectID:                created.ProjectID,
		Config:                   config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 8, updated.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, updated.Config, "spec-e")
}

func TestCreateRolloutTasksRequiresMatchingApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	staleVersion := int64(1)
	updated, createdTasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: staleVersion}, nil)
	require.NoError(t, err)
	require.False(t, updated)
	require.Empty(t, createdTasks)

	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, got.Config, "spec-a")

	currentVersion := int64(2)
	updated, createdTasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: currentVersion}, nil)
	require.NoError(t, err)
	require.True(t, updated)
	require.Empty(t, createdTasks)

	got, err = s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, got.Config, "spec-a")
}

func TestCreateRolloutTasksRequiresCurrentIssueApproval(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{"roles/sql-reviewer"}},
				},
				Approvers: []*storepb.IssuePayloadApproval_Approver{
					{
						Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
						Principal: "users/reviewer@example.com",
					},
				},
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	approvedIssueApproval := &storepb.IssuePayloadApproval{}
	proto.Merge(approvedIssueApproval, issue.Payload.GetApproval())
	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Labels: []string{"security"}},
		ConditionalPayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: approvalInputVersion,
			},
		},
		ConditionalPlanApprovalInputVersion: &approvalInputVersion,
		ConditionalRequireNoRollout:         true,
	})
	require.NoError(t, err)
	require.False(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
	require.Equal(t, []string{"security"}, updatedIssue.Payload.Labels)

	updatedRollout, createdTasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{
		IssueUID:             issue.UID,
		ApprovalInputVersion: approvalInputVersion,
		Approval:             approvedIssueApproval,
	}, nil)
	require.NoError(t, err)
	require.False(t, updatedRollout)
	require.Empty(t, createdTasks)

	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())

	labels := []string{"security"}
	recomputedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{"roles/sql-reviewer"}},
				},
				Approvers: []*storepb.IssuePayloadApproval_Approver{
					{
						Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
						Principal: "users/other-reviewer@example.com",
					},
				},
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
	})
	require.NoError(t, err)
	require.False(t, approvedIssueApproval.Equal(recomputedIssue.Payload.GetApproval()))

	updatedRollout, createdTasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{
		IssueUID:             issue.UID,
		ApprovalInputVersion: approvalInputVersion,
		Approval:             approvedIssueApproval,
	}, nil)
	require.NoError(t, err)
	require.False(t, updatedRollout)
	require.Empty(t, createdTasks)

	got, err = s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())

	currentIssueApproval := &storepb.IssuePayloadApproval{}
	proto.Merge(currentIssueApproval, recomputedIssue.Payload.GetApproval())
	updatedRollout, createdTasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{
		IssueUID:             issue.UID,
		ApprovalInputVersion: approvalInputVersion,
		Approval:             currentIssueApproval,
	}, nil)
	require.NoError(t, err)
	require.True(t, updatedRollout)
	require.Empty(t, createdTasks)

	got, err = s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())
}

func TestCreateRolloutTasksAddsMissingTasksAfterRolloutExists(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	updated, createdTasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, nil, []*store.TaskMessage{
		newTestRolloutTask("instance-a", "database-a", "sheet-a"),
	})
	require.NoError(t, err)
	require.True(t, updated)
	require.Len(t, createdTasks, 1)

	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())

	updated, createdTasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, nil, []*store.TaskMessage{
		newTestRolloutTask("instance-a", "database-a", "sheet-a"),
		newTestRolloutTask("instance-a", "database-b", "sheet-b"),
	})
	require.NoError(t, err)
	require.True(t, updated)
	require.Len(t, createdTasks, 1)
	require.Equal(t, "instance-a", createdTasks[0].InstanceID)
	require.Equal(t, "database-b", createdTasks[0].GetDatabaseName())
	require.Equal(t, "sheet-b", createdTasks[0].Payload.GetSheetSha256())
}

func TestUpdatePlanRequireNoRolloutDoesNotOverwriteRollout(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 1,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	staleConfig := proto.CloneOf(plan.Config)
	staleConfig.Specs = []*storepb.PlanConfig_Spec{{Id: "spec-b"}}

	approvalInputVersion := int64(1)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      plan.UID,
		ProjectID:                "project-a",
		Config:                   staleConfig,
		BumpApprovalInputVersion: true,
		RequireNoRollout:         true,
	})
	require.Error(t, err)

	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())
	require.EqualValues(t, 1, got.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, got.Config, "spec-a")
}

func TestUpdatePlanRequireNoRolloutRejectsConfigUpdateAfterRollout(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 1,
			Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	staleConfig := proto.CloneOf(plan.Config)
	staleConfig.Specs = []*storepb.PlanConfig_Spec{{Id: "spec-b"}}

	approvalInputVersion := int64(1)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:              plan.UID,
		ProjectID:        "project-a",
		Config:           staleConfig,
		RequireNoRollout: true,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, store.ErrPlanHasRollout)

	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())
	require.EqualValues(t, 1, got.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, got.Config, "spec-a")
}

func TestUpdateIssueWithPlanApprovalInputVersionGuardUpdatesMatchingVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{RiskLevel: storepb.RiskLevel_LOW},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssuePayloadUpsertCanRemoveLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels:    []string{"prod"},
			RiskLevel: storepb.RiskLevel_LOW,
		},
	})
	require.NoError(t, err)

	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{RiskLevel: storepb.RiskLevel_HIGH},
		RemoveLabels:  true,
	})
	require.NoError(t, err)
	require.Empty(t, updatedIssue.Payload.GetLabels())
	require.Equal(t, storepb.RiskLevel_HIGH, updatedIssue.Payload.GetRiskLevel())
}

func TestUpdateIssueConditionalPayloadRequiresPlanGuard(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{},
	})
	require.NoError(t, err)

	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		ConditionalPayloadUpsert: &storepb.Issue{RiskLevel: storepb.RiskLevel_HIGH},
	})
	require.ErrorContains(t, err, "ConditionalPayloadUpsert requires ConditionalPlanApprovalInputVersion")
}

func TestUpdateIssueWithPlanApprovalInputVersionGuardSkipsStaleVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{RiskLevel: storepb.RiskLevel_LOW},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(1)
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 1,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}

func TestUpdateIssueWithPlanApprovalInputVersionGuardSkipsIssueWithoutPlan(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{RiskLevel: storepb.RiskLevel_LOW},
	})
	require.NoError(t, err)

	approvalInputVersion := int64(0)
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}

func TestUpdateIssueWithPlanApprovalInputVersionAndLabelsGuardsUpdatesMatchingLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels:    []string{"prod", "security"},
			RiskLevel: storepb.RiskLevel_LOW,
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	labels := []string{"security", "prod", "prod"}
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithPlanApprovalInputVersionAndLabelsGuardsUpdatesEmptyLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload:      &storepb.Issue{RiskLevel: storepb.RiskLevel_LOW},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	var labels []string
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithPlanApprovalInputVersionAndLabelsGuardsUpdatesJSONNullLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels:    []string{"prod"},
			RiskLevel: storepb.RiskLevel_LOW,
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{RemoveLabels: true})
	require.NoError(t, err)
	require.Empty(t, updatedIssue.Payload.GetLabels())

	approvalInputVersion := int64(2)
	var labels []string
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Empty(t, got.Payload.GetLabels())
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithPlanApprovalInputVersionAndLabelsGuardsSkipsStaleLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels:    []string{"prod"},
			RiskLevel: storepb.RiskLevel_LOW,
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Labels: []string{"stage"}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())

	approvalInputVersion := int64(2)
	labels := []string{"prod"}
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}

func TestUpdateIssueWithPlanApprovalInputVersionLabelsAndNoRolloutGuardsUpdatesBeforeRollout(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod", "security"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	labels := []string{"security", "prod", "prod"}
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
		RequireNoRollout:                true,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithPlanApprovalInputVersionLabelsAndNoRolloutGuardsUpdatesJSONNullLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{RemoveLabels: true})
	require.NoError(t, err)
	require.Empty(t, updatedIssue.Payload.GetLabels())

	approvalInputVersion := int64(2)
	var labels []string
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
		RequireNoRollout:                true,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Empty(t, got.Payload.GetLabels())
	require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithPlanApprovalInputVersionLabelsAndNoRolloutGuardsSkipsAfterRollout(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod", "security"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	labels := []string{"security", "prod", "prod"}
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
		RequireNoRollout:                true,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueConditionalPayloadSkipsAfterRolloutButMainUpdateApplies(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Labels: []string{"stage"}},
		ConditionalPayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: approvalInputVersion,
			},
		},
		ConditionalPlanApprovalInputVersion: &approvalInputVersion,
		ConditionalRequireNoRollout:         true,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.True(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsAndMaybeResetApprovalResetsBeforeRollout(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, approvalResetApplied, err := s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.True(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.False(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsAndMaybeResetApprovalDoesNotResetAfterRollout(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			HasRollout:           false,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &store.IssueApprovalGuard{ApprovalInputVersion: approvalInputVersion}, nil)
	require.NoError(t, err)
	require.True(t, marked)

	updatedIssue, approvalResetApplied, err := s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.False(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.True(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsAndMaybeResetApprovalDoesNotResetDraft(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Draft:  true,
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, approvalResetApplied, err := s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.False(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.True(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsAndMaybeResetApprovalDoesNotResetCreateDatabase(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
					CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, approvalResetApplied, err := s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.False(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.True(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsAndMaybeResetApprovalUsesCurrentPlanVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedPlan, err := s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      plan.UID,
		ProjectID:                plan.ProjectID,
		Config:                   plan.Config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, updatedPlan.Config.GetApprovalInputVersion())

	updatedIssue, approvalResetApplied, err := s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.True(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.False(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 3, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueLabelsAndMaybeResetApprovalNoopsWhenLockedLabelsMatch(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			ApprovalInputVersion: 2,
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels: []string{"prod"},
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, approvalResetApplied, err := s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.True(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.False(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())

	approvalInputVersion := int64(2)
	labels := []string{"stage"}
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: approvalInputVersion,
			},
		},
		RequirePlanApprovalInputVersion: &approvalInputVersion,
		RequireLabels:                   &labels,
	})
	require.NoError(t, err)

	updatedIssue, approvalResetApplied, err = s.UpdateIssueLabelsAndMaybeResetApproval(ctx, "project-a", issue.UID, []string{"stage"})
	require.NoError(t, err)
	require.False(t, approvalResetApplied)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())
	require.True(t, updatedIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, updatedIssue.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithCurrentDoneApprovalGuardSkipsCurrentDoneApproval(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion:  &approvalInputVersion,
		SkipIfCurrentApprovalFindingDone: &approvalInputVersion,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithCurrentApprovalInputVersionGuardsUpdatesMatchingVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_LOW,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 2,
			},
		},
		RequirePlanApprovalInputVersion:  &approvalInputVersion,
		RequireIssueApprovalInputVersion: &approvalInputVersion,
	})
	require.NoError(t, err)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueWithCurrentApprovalInputVersionGuardsSkipsStaleIssueApprovalVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_LOW,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 1,
			},
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	approvalInputVersion := int64(2)
	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			RiskLevel: storepb.RiskLevel_HIGH,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  true,
				ApprovalInputVersion: 1,
			},
		},
		RequirePlanApprovalInputVersion:  &approvalInputVersion,
		RequireIssueApprovalInputVersion: &approvalInputVersion,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.EqualValues(t, 1, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssueDraftGuardAllowsSubmissionAndSkipsStaleDraftLabelWrite(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
	})
	require.NoError(t, err)

	requireDraft := true
	submitted, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		RemoveDraft:  true,
		RequireDraft: &requireDraft,
	})
	require.NoError(t, err)
	require.False(t, submitted.Payload.GetDraft())

	_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Labels: []string{"stage"}},
		RequireDraft:  &requireDraft,
	})
	require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.False(t, got.Payload.GetDraft())
	require.Empty(t, got.Payload.GetLabels())
}

func TestUpdateIssuePlanSnapshotGuard(t *testing.T) {
	tests := []struct {
		name                   string
		mutatePlan             func(context.Context, *testing.T, *store.Store, *store.PlanMessage) *store.PlanMessage
		wantSkip               bool
		useCurrentPlanSnapshot bool
	}{
		{
			name: "matching active plan",
			mutatePlan: func(_ context.Context, _ *testing.T, _ *store.Store, plan *store.PlanMessage) *store.PlanMessage {
				return plan
			},
		},
		{
			name: "stale plan update time",
			mutatePlan: func(ctx context.Context, t *testing.T, s *store.Store, plan *store.PlanMessage) *store.PlanMessage {
				description := "closed after validation"
				updated, err := s.UpdatePlan(ctx, &store.UpdatePlanMessage{
					UID:         plan.UID,
					ProjectID:   plan.ProjectID,
					Description: &description,
				})
				require.NoError(t, err)
				return updated
			},
			wantSkip: true,
		},
		{
			name: "inactive plan",
			mutatePlan: func(ctx context.Context, t *testing.T, s *store.Store, plan *store.PlanMessage) *store.PlanMessage {
				deleted := true
				updated, err := s.UpdatePlan(ctx, &store.UpdatePlanMessage{
					UID:       plan.UID,
					ProjectID: plan.ProjectID,
					Deleted:   &deleted,
				})
				require.NoError(t, err)
				return updated
			},
			wantSkip:               true,
			useCurrentPlanSnapshot: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			s := setupPlanApprovalInputVersionStore(ctx, t)

			plan, err := s.CreatePlan(ctx, &store.PlanMessage{
				ProjectID: "project-a",
				Name:      "plan-a",
				Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
			}, "creator@example.com")
			require.NoError(t, err)
			issue, err := s.CreateIssue(ctx, &store.IssueMessage{
				ProjectID:    "project-a",
				CreatorEmail: "creator@example.com",
				Title:        "issue-a",
				Type:         storepb.Issue_DATABASE_CHANGE,
				Payload:      &storepb.Issue{Draft: true},
				PlanUID:      &plan.UID,
			})
			require.NoError(t, err)

			validatedUpdatedAt := plan.UpdatedAt
			currentPlan := test.mutatePlan(ctx, t, s, plan)
			if test.useCurrentPlanSnapshot {
				validatedUpdatedAt = currentPlan.UpdatedAt
			}
			version := int64(2)
			requireDraft := true
			_, err = s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
				RemoveDraft:                     true,
				RequireDraft:                    &requireDraft,
				RequirePlanApprovalInputVersion: &version,
				RequirePlanUpdatedAt:            &validatedUpdatedAt,
				RequirePlanActive:               true,
			})
			if test.wantSkip {
				require.ErrorIs(t, err, store.ErrIssueUpdateSkipped)
				got, getErr := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
				require.NoError(t, getErr)
				require.True(t, got.Payload.GetDraft())
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateIssuePlanSnapshotGuardWaitsForConcurrentPlanUpdate(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "plan-a",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	planUpdate, err := s.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer planUpdate.Rollback()
	_, err = planUpdate.ExecContext(ctx, `
		UPDATE plan
		SET updated_at = updated_at + INTERVAL '1 second'
		WHERE project = $1 AND id = $2
	`, plan.ProjectID, plan.UID)
	require.NoError(t, err)

	requireDraft := true
	version := int64(2)
	updateResult := make(chan error, 1)
	go func() {
		_, err := s.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{
			RemoveDraft:                     true,
			RequireDraft:                    &requireDraft,
			RequirePlanApprovalInputVersion: &version,
			RequirePlanUpdatedAt:            &plan.UpdatedAt,
			RequirePlanActive:               true,
		})
		updateResult <- err
	}()

	waitForTransactionBlock(ctx, t, s.GetDB(), planUpdate)

	require.NoError(t, planUpdate.Commit())
	require.ErrorIs(t, receiveTestResult(t, updateResult, "guarded Issue update did not return"), store.ErrIssueUpdateSkipped)
	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.True(t, got.Payload.GetDraft())
}

func waitForTransactionBlock(ctx context.Context, t *testing.T, db *sql.DB, tx *sql.Tx) {
	t.Helper()
	var blockerPID int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockerPID))
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity AS activity
				WHERE activity.pid <> pg_backend_pid()
				  AND $1 = ANY(pg_blocking_pids(activity.pid))
			)
		`, blockerPID).Scan(&waiting))
		if waiting {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for a session blocked by transaction PID %d", blockerPID)
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

func requirePlanSpecID(t *testing.T, config *storepb.PlanConfig, id string) {
	t.Helper()
	require.Len(t, config.GetSpecs(), 1)
	require.Equal(t, id, config.GetSpecs()[0].GetId())
}

func newTestRolloutTask(instanceID string, databaseName string, sheetSha256 string) *store.TaskMessage {
	return &store.TaskMessage{
		InstanceID:   instanceID,
		DatabaseName: &databaseName,
		Type:         storepb.Task_DATABASE_MIGRATE,
		Payload: &storepb.Task{
			Source: &storepb.Task_SheetSha256{
				SheetSha256: sheetSha256,
			},
		},
	}
}

func setupPlanApprovalInputVersionStore(ctx context.Context, t *testing.T) *store.Store {
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

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
