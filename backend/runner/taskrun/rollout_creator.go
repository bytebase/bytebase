package taskrun

import (
	"context"
	"log/slog"
	"sync"

	"github.com/pkg/errors"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// RolloutCreator handles automatic rollout creation.
// nolint:revive
type RolloutCreator struct {
	store          *store.Store
	bus            *bus.Bus
	webhookManager *webhook.Manager
	sqlReviewV2    bool
}

// NewRolloutCreator creates a new rollout creator.
func NewRolloutCreator(store *store.Store, bus *bus.Bus, webhookManager *webhook.Manager, sqlReviewV2 bool) *RolloutCreator {
	return &RolloutCreator{
		store:          store,
		bus:            bus,
		webhookManager: webhookManager,
		sqlReviewV2:    sqlReviewV2,
	}
}

// Run starts the rollout creator listening to the channel.
func (rc *RolloutCreator) Run(ctx context.Context, wg *sync.WaitGroup, rolloutCreationChan chan bus.PlanRef) {
	defer wg.Done()
	slog.Debug("Rollout creator started")

	for {
		select {
		case ref := <-rolloutCreationChan:
			rc.tryCreateRollout(ctx, ref)
		case <-ctx.Done():
			slog.Debug("Rollout creator stopped")
			return
		}
	}
}

// tryCreateRollout attempts to create a rollout for the given plan.
func (rc *RolloutCreator) tryCreateRollout(ctx context.Context, ref bus.PlanRef) {
	planID := ref.PlanID
	plan, err := rc.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: ref.ProjectID, UID: &planID})
	if err != nil {
		slog.Error("failed to get plan for rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}
	if plan == nil {
		slog.Debug("plan not found for rollout creation", slog.String("project", ref.ProjectID), slog.Int("plan_id", int(planID)))
		return
	}

	// Skip deleted (closed) plans
	if plan.Deleted {
		slog.Debug("plan is deleted, skipping rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)))
		return
	}

	// Idempotency: skip if rollout already exists
	if plan.Config != nil && plan.Config.HasRollout {
		slog.Debug("rollout already exists, skipping creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)))
		return
	}

	// Only auto-create rollout for plans with change database specs
	if !hasOnlyChangeDatabaseSpecs(plan) {
		slog.Debug("plan has non-change-database specs, skipping auto-creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)))
		return
	}

	issue, err := rc.store.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{ref.ProjectID}, PlanUID: &planID})
	if err != nil {
		slog.Error("failed to get issue for rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}
	if issue == nil {
		slog.Debug("issue not found for rollout creation", slog.String("project", ref.ProjectID), slog.Int("plan_id", int(planID)))
		return
	}
	if issue.Payload.GetDraft() {
		slog.Debug("issue is a draft, skipping rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)))
		return
	}

	// Skip canceled issues
	if issue.Status == storepb.Issue_CANCELED {
		slog.Debug("issue is canceled, skipping rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)))
		return
	}

	project, err := rc.store.GetProjectByResourceID(ctx, plan.ProjectID)
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
	if project.Deleted {
		slog.Debug("project is archived, skipping rollout creation", slog.String("project_id", plan.ProjectID))
		return
	}

	// Check approval status (must be approved)
	approved, err := utils.CheckIssueApprovedForPlan(issue, plan)
	if err != nil {
		slog.Error("failed to check if the issue is approved",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}
	if !approved {
		slog.Debug("issue not approved, skipping rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)))
		return
	}

	// Check plan check status (must have no errors)
	planCheckRun, err := rc.store.GetPlanCheckRun(ctx, plan.ProjectID, planID)
	if err != nil {
		slog.Error("failed to get plan check run for rollout creation",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}

	// If current plan checks exist, they must be DONE with no errors.
	if planCheckRunBlocksRollout(plan, planCheckRun) {
		// Check if plan checks are in DONE status
		if planCheckRun.Status != store.PlanCheckRunStatusDone {
			slog.Debug("plan checks not in DONE status, skipping rollout creation",
				slog.String("project", ref.ProjectID),
				slog.Int("plan_id", int(planID)),
				slog.String("status", string(planCheckRun.Status)))
			return
		}

		// Check for ERROR-level results
		if planCheckRun.Result != nil {
			for _, result := range planCheckRun.Result.Results {
				if result.Status == storepb.Advice_ERROR {
					slog.Debug("plan checks have errors, skipping rollout creation",
						slog.String("project", ref.ProjectID),
						slog.Int("plan_id", int(planID)))
					return
				}
			}
		}
	}

	if rc.sqlReviewV2 {
		blocked, err := rc.reviewBlocksRollout(ctx, project, plan, issue)
		if err != nil {
			slog.Error("failed to check the review for rollout creation",
				slog.String("project", ref.ProjectID),
				slog.Int("plan_id", int(planID)),
				log.BBError(err))
			return
		}
		if blocked {
			slog.Debug("review not finished or threads open, skipping rollout creation",
				slog.String("project", ref.ProjectID),
				slog.Int("plan_id", int(planID)))
			return
		}
	}

	// All conditions met - create the rollout
	slog.Info("auto-creating rollout", slog.String("project", ref.ProjectID), slog.Int("plan_id", int(planID)))

	// Create rollout and pending tasks
	// Use issue creator's email since this is auto-rollout for their issue
	if err := apiv1.CreateRolloutAndPendingTasks(ctx, rc.store, issue.CreatorEmail, plan, issue, project, nil); err != nil {
		if apiv1.IsDraftIssueNotSubmittedError(err) {
			slog.Info("skip auto-rollout because issue is still a draft",
				slog.String("project", ref.ProjectID),
				slog.Int("plan_id", int(planID)))
			return
		}
		if apiv1.IsStaleRolloutApprovalError(err) {
			slog.Info("skip auto-rollout because issue approval is stale",
				slog.String("project", ref.ProjectID),
				slog.Int("plan_id", int(planID)))
			return
		}
		slog.Error("failed to create rollout and pending tasks",
			slog.String("project", ref.ProjectID),
			slog.Int("plan_id", int(planID)),
			log.BBError(err))
		return
	}

	// Tickle the pending task run scheduler.
	bus.Tickle(rc.bus.TaskRunPendingTickleChan)

	slog.Info("successfully auto-created rollout", slog.String("project", ref.ProjectID), slog.Int("plan_id", int(planID)))
}

// reviewBlocksRollout reports whether the issue's review holds its rollout
// under SQL Review V2: the rule review of the plan's current SQL has not
// finished, or a thread on the issue is open. An open thread blocks whatever
// its priority, and whoever started it.
//
// A run for an earlier approval input version holds too, and so does a
// missing run on a plan with SQL to review: the slot write that should have
// started the review failed, or never happened. The review starts here then,
// as the approval evaluator does for a missing or stale plan check run, and its
// completion asks for this check again. A stale run restarts even when the
// plan has nothing left to review: it then completes with no findings, which
// supersedes those of the earlier SQL.
func (rc *RolloutCreator) reviewBlocksRollout(ctx context.Context, project *store.ProjectMessage, plan *store.PlanMessage, issue *store.IssueMessage) (bool, error) {
	run, err := rc.store.GetReviewRun(ctx, issue.ProjectID, issue.UID, store.ReviewRunTypeRule)
	if err != nil {
		return false, errors.Wrap(err, "failed to get the rule review run")
	}
	start := run != nil && run.Payload.GetApprovalInputVersion() != plan.Config.GetApprovalInputVersion()
	if run == nil {
		start, err = plancheck.HasReviewTargets(ctx, rc.store, project, plan)
		if err != nil {
			return false, err
		}
	}
	if start {
		if _, err := rc.store.CreateReviewRun(ctx, issue.ProjectID, issue.UID, store.ReviewRunTypeRule); err != nil {
			return false, errors.Wrap(err, "failed to start the rule review")
		}
		bus.Tickle(rc.bus.ReviewRunTickleChan)
		return true, nil
	}
	if run != nil && run.Status != storepb.ReviewRun_DONE {
		return true, nil
	}
	open := store.ThreadStateOpen
	limit := 1
	threads, err := rc.store.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		ProjectID:   issue.ProjectID,
		IssueUID:    &issue.UID,
		ThreadState: &open,
		Limit:       &limit,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list open threads")
	}
	return len(threads) > 0, nil
}

func planCheckRunBlocksRollout(plan *store.PlanMessage, planCheckRun *store.PlanCheckRunMessage) bool {
	if planCheckRun == nil {
		return false
	}
	return planCheckRun.Result.GetApprovalInputVersion() == plan.Config.GetApprovalInputVersion()
}

// hasOnlyChangeDatabaseSpecs checks if every spec in the plan is a change database spec.
// Only plans with change database specs support auto-creation of rollouts.
func hasOnlyChangeDatabaseSpecs(plan *store.PlanMessage) bool {
	if plan.Config == nil {
		return false
	}
	if len(plan.Config.Specs) == 0 {
		return false
	}
	for _, spec := range plan.Config.Specs {
		if _, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig); !ok {
			return false
		}
	}
	return true
}
