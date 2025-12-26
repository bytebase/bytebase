package rollout

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// RolloutCreator handles automatic rollout creation.
type RolloutCreator struct {
	store          *store.Store
	rolloutService *apiv1.RolloutService
}

// NewRolloutCreator creates a new rollout creator.
func NewRolloutCreator(store *store.Store, rolloutService *apiv1.RolloutService) *RolloutCreator {
	return &RolloutCreator{
		store:          store,
		rolloutService: rolloutService,
	}
}

// Run starts the rollout creator listening to the channel.
func (rc *RolloutCreator) Run(ctx context.Context, wg *sync.WaitGroup, rolloutCreationChan chan int64) {
	defer wg.Done()
	slog.Debug("Rollout creator started")

	for {
		select {
		case planID := <-rolloutCreationChan:
			rc.tryCreateRollout(ctx, planID)
		case <-ctx.Done():
			slog.Debug("Rollout creator stopped")
			return
		}
	}
}

// tryCreateRollout attempts to create a rollout for the given plan.
func (rc *RolloutCreator) tryCreateRollout(_ context.Context, planID int64) {
	// Use background context with timeout to avoid being affected by request cancellation
	rolloutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	plan, err := rc.store.GetPlan(rolloutCtx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		slog.Error("failed to get plan for rollout creation",
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}
	if plan == nil {
		slog.Debug("plan not found for rollout creation", slog.Int("plan_id", int(planID)))
		return
	}

	// Idempotency: skip if rollout already exists
	if plan.Config != nil && plan.Config.HasRollout {
		slog.Debug("rollout already exists, skipping creation",
			slog.Int("plan_id", int(planID)))
		return
	}

	issue, err := rc.store.GetIssue(rolloutCtx, &store.FindIssueMessage{PlanUID: &planID})
	if err != nil {
		slog.Error("failed to get issue for rollout creation",
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}
	if issue == nil {
		slog.Debug("issue not found for rollout creation", slog.Int("plan_id", int(planID)))
		return
	}

	project, err := rc.store.GetProject(rolloutCtx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
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
		approved, err := utils.CheckIssueApproved(issue)
		if err != nil {
			slog.Error("failed to check if the issue is approved",
				slog.Int("plan_id", int(planID)),
				log.BBError(err))
			return
		}
		if !approved {
			slog.Debug("issue not approved yet, skipping rollout creation",
				slog.Int("plan_id", int(planID)))
			return
		}
	}

	// Check plan check condition
	if project.Setting != nil && project.Setting.RequirePlanCheckNoError {
		planCheckRun, err := rc.store.GetPlanCheckRun(rolloutCtx, planID)
		if err != nil {
			slog.Error("failed to get plan check run for rollout creation",
				slog.Int("plan_id", int(planID)),
				log.BBError(err))
			return
		}

		// If no plan checks exist, treat as passing
		if planCheckRun != nil {
			// Check if plan checks are in DONE status
			if planCheckRun.Status != store.PlanCheckRunStatusDone {
				slog.Debug("plan checks not in DONE status, skipping rollout creation",
					slog.Int("plan_id", int(planID)),
					slog.String("status", string(planCheckRun.Status)))
				return
			}

			// Check for ERROR-level results
			if planCheckRun.Result != nil {
				for _, result := range planCheckRun.Result.Results {
					if result.Status == storepb.Advice_ERROR {
						slog.Debug("plan checks have errors, skipping rollout creation",
							slog.Int("plan_id", int(planID)))
						return
					}
				}
			}
		}
	}

	// All conditions met - create the rollout
	slog.Info("auto-creating rollout",
		slog.Int("plan_id", int(planID)))

	projectID := common.FormatProject(plan.ProjectID)
	planName := common.FormatPlan(plan.ProjectID, planID)

	_, err = rc.rolloutService.CreateRollout(rolloutCtx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: projectID,
		Rollout: &v1pb.Rollout{
			Plan: planName,
		},
	}))

	if err != nil {
		// If rollout already exists, this is not an error (race condition handled)
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			slog.Debug("rollout already exists (race condition), ignoring",
				slog.Int("plan_id", int(planID)))
			return
		}
		slog.Error("failed to auto-create rollout",
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}

	slog.Info("successfully auto-created rollout",
		slog.Int("plan_id", int(planID)))
}
