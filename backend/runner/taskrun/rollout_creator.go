package taskrun

import (
	"context"
	"log/slog"
	"sync"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// RolloutCreator handles automatic rollout creation.
// nolint:revive
type RolloutCreator struct {
	store          *store.Store
	bus            *bus.Bus
	webhookManager *webhook.Manager
}

// NewRolloutCreator creates a new rollout creator.
func NewRolloutCreator(store *store.Store, bus *bus.Bus, webhookManager *webhook.Manager) *RolloutCreator {
	return &RolloutCreator{
		store:          store,
		bus:            bus,
		webhookManager: webhookManager,
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
func (rc *RolloutCreator) tryCreateRollout(ctx context.Context, planID int64) {
	plan, err := rc.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
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

	issue, err := rc.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &planID})
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

	project, err := rc.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
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

	// Check approval status (must be approved)
	approved, err := utils.CheckIssueApproved(issue)
	if err != nil {
		slog.Error("failed to check if the issue is approved",
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}
	if !approved {
		slog.Debug("issue not approved, skipping rollout creation",
			slog.Int("plan_id", int(planID)))
		return
	}

	// Check plan check status (must have no errors)
	planCheckRun, err := rc.store.GetPlanCheckRun(ctx, planID)
	if err != nil {
		slog.Error("failed to get plan check run for rollout creation",
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}

	// If plan checks exist, they must be DONE with no errors
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

	// All conditions met - create the rollout
	slog.Info("auto-creating rollout", slog.Int("plan_id", int(planID)))

	// Create rollout and pending tasks
	if err := apiv1.CreateRolloutAndPendingTasks(ctx, rc.store, plan, issue, project, nil); err != nil {
		slog.Error("failed to create rollout and pending tasks",
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}

	// Tickle task run scheduler
	rc.bus.TaskRunTickleChan <- 0

	slog.Info("successfully auto-created rollout", slog.Int("plan_id", int(planID)))
}
