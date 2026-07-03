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
	updated, createdTasks, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &staleVersion, nil)
	require.NoError(t, err)
	require.False(t, updated)
	require.Empty(t, createdTasks)

	got, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.False(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, got.Config, "spec-a")

	currentVersion := int64(2)
	updated, createdTasks, err = s.CreateRolloutTasks(ctx, "project-a", plan.UID, &currentVersion, nil)
	require.NoError(t, err)
	require.True(t, updated)
	require.Empty(t, createdTasks)

	got, err = s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: "project-a", UID: &plan.UID})
	require.NoError(t, err)
	require.True(t, got.Config.GetHasRollout())
	require.EqualValues(t, 2, got.Config.GetApprovalInputVersion())
	requirePlanSpecID(t, got.Config, "spec-a")
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
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &approvalInputVersion, nil)
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
	marked, _, err := s.CreateRolloutTasks(ctx, "project-a", plan.UID, &approvalInputVersion, nil)
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

func TestUpdateIssuePayloadIfPlanApprovalInputVersionUpdatesMatchingVersion(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersion(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2)
	require.NoError(t, err)
	require.True(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssuePayloadIfPlanApprovalInputVersionSkipsStaleVersion(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersion(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 1,
		},
	}, 1)
	require.NoError(t, err)
	require.False(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}

func TestUpdateIssuePayloadIfPlanApprovalInputVersionSkipsIssueWithoutPlan(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersion(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
		},
	}, 0)
	require.NoError(t, err)
	require.False(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}

func TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsUpdatesMatchingLabels(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2, []string{"security", "prod", "prod"})
	require.NoError(t, err)
	require.True(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsUpdatesEmptyLabels(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2, nil)
	require.NoError(t, err)
	require.True(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsSkipsStaleLabels(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2, []string{"prod"})
	require.NoError(t, err)
	require.False(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}

func TestResetIssueApprovalFindingIfPlanApprovalInputVersionSkipsCurrentDoneApproval(t *testing.T) {
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

	updated, err := s.ResetIssueApprovalFindingIfPlanApprovalInputVersion(ctx, "project-a", issue.UID, 2)
	require.NoError(t, err)
	require.False(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssuePayloadIfCurrentApprovalInputVersionUpdatesMatchingVersion(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfCurrentApprovalInputVersion(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2)
	require.NoError(t, err)
	require.True(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
}

func TestUpdateIssuePayloadIfCurrentApprovalInputVersionSkipsStaleIssueApprovalVersion(t *testing.T) {
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

	updated, err := s.UpdateIssuePayloadIfCurrentApprovalInputVersion(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 1,
		},
	}, 2)
	require.NoError(t, err)
	require.False(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.EqualValues(t, 1, got.Payload.GetApproval().GetApprovalInputVersion())
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
