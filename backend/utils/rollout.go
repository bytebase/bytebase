package utils

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// RolloutServiceInterface defines the interface for creating rollouts.
type RolloutServiceInterface interface {
	CreateRollout(ctx context.Context, req *connect.Request[v1pb.CreateRolloutRequest]) (*connect.Response[v1pb.Rollout], error)
}

// TryCreateRollout attempts to create a rollout if all conditions are met.
// This function is called asynchronously after approval or plan check completion.
// It checks approval and plan check conditions before calling CreateRollout.
func TryCreateRollout(ctx context.Context, stores *store.Store, rolloutService RolloutServiceInterface, issueID int) {
	issue, err := stores.GetIssue(ctx, &store.FindIssueMessage{UID: &issueID})
	if err != nil {
		slog.Error("failed to get issue for rollout creation",
			slog.Int("issue_id", issueID),
			log.BBError(err))
		return
	}
	if issue == nil {
		slog.Debug("issue not found for rollout creation", slog.Int("issue_id", issueID))
		return
	}

	if issue.PlanUID == nil {
		slog.Debug("issue has no plan, skipping rollout creation", slog.Int("issue_id", issueID))
		return
	}

	plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		slog.Error("failed to get plan for rollout creation",
			slog.Int("plan_id", int(*issue.PlanUID)),
			log.BBError(err))
		return
	}
	if plan == nil {
		slog.Debug("plan not found for rollout creation", slog.Int("plan_id", int(*issue.PlanUID)))
		return
	}

	// Idempotency: skip if rollout already exists
	if plan.Config != nil && plan.Config.HasRollout {
		slog.Debug("rollout already exists, skipping creation",
			slog.Int("issue_id", issueID),
			slog.Int("plan_id", int(*issue.PlanUID)))
		return
	}

	project, err := stores.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		slog.Error("failed to get project for rollout creation",
			slog.String("project_id", plan.ProjectID),
			log.BBError(err))
		return
	}
	if project == nil {
		slog.Error("project not found for rollout creation", slog.String("project_id", plan.ProjectID))
		return
	}

	// Check approval condition
	if project.Setting != nil && project.Setting.RequireIssueApproval {
		approved, err := CheckIssueApproved(issue)
		if err != nil {
			slog.Error("failed to check if the issue is approved",
				slog.Int("issue_id", issueID),
				log.BBError(err))
			return
		}
		if !approved {
			slog.Debug("issue not approved yet, skipping rollout creation",
				slog.Int("issue_id", issueID))
			return
		}
	}

	// Check plan check condition
	if project.Setting != nil && project.Setting.RequirePlanCheckNoError {
		planCheckRun, err := stores.GetPlanCheckRun(ctx, *issue.PlanUID)
		if err != nil {
			slog.Error("failed to get plan check run for rollout creation",
				slog.Int("plan_id", int(*issue.PlanUID)),
				log.BBError(err))
			return
		}

		// If no plan checks exist, treat as passing (same as old behavior)
		if planCheckRun != nil {
			// Check if plan checks are in DONE status
			if planCheckRun.Status != store.PlanCheckRunStatusDone {
				slog.Debug("plan checks not in DONE status, skipping rollout creation",
					slog.Int("issue_id", issueID),
					slog.String("status", string(planCheckRun.Status)))
				return
			}

			// Check for ERROR-level results
			if planCheckRun.Result != nil {
				for _, result := range planCheckRun.Result.Results {
					if result.Status == storepb.Advice_ERROR {
						slog.Debug("plan checks have errors, skipping rollout creation",
							slog.Int("issue_id", issueID))
						return
					}
				}
			}
		}
	}

	// All conditions met - create the rollout
	slog.Info("auto-creating rollout",
		slog.Int("issue_id", issueID),
		slog.Int("plan_id", int(*issue.PlanUID)))

	projectID := common.FormatProject(plan.ProjectID)
	planName := common.FormatPlan(plan.ProjectID, *issue.PlanUID)

	_, err = rolloutService.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: projectID,
		Rollout: &v1pb.Rollout{
			Plan: planName,
		},
	}))

	if err != nil {
		// If rollout already exists, this is not an error (race condition handled)
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			slog.Debug("rollout already exists (race condition), ignoring",
				slog.Int("issue_id", issueID))
			return
		}
		slog.Error("failed to auto-create rollout",
			slog.Int("issue_id", issueID),
			slog.Int("plan_id", int(*issue.PlanUID)),
			log.BBError(err))
		return
	}

	slog.Info("successfully auto-created rollout",
		slog.Int("issue_id", issueID),
		slog.Int("plan_id", int(*issue.PlanUID)))
}
