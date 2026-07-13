// Package plancheck is the runner for plan checks.
package plancheck

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	planCheckSchedulerInterval = 5 * time.Second
)

// NewScheduler creates a new plan check scheduler.
func NewScheduler(s *store.Store, bus *bus.Bus, executor *CombinedExecutor, licenseService *enterprise.LicenseService) *Scheduler {
	return &Scheduler{
		store:          s,
		bus:            bus,
		executor:       executor,
		licenseService: licenseService,
	}
}

// Scheduler is the plan check run scheduler.
type Scheduler struct {
	store          *store.Store
	bus            *bus.Bus
	executor       *CombinedExecutor
	licenseService *enterprise.LicenseService
}

// Run runs the scheduler.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(planCheckSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Plan check scheduler started and will run every %v", planCheckSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			if err := s.licenseService.CheckReplicaLimit(ctx); err != nil {
				slog.Warn("Plan check scheduler skipped due to HA license restriction", log.BBError(err))
				continue
			}
			s.runOnce(ctx)
		case <-s.bus.PlanCheckTickleChan:
			if err := s.licenseService.CheckReplicaLimit(ctx); err != nil {
				slog.Warn("Plan check scheduler skipped due to HA license restriction", log.BBError(err))
				continue
			}
			s.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Plan check scheduler PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	claimed, err := s.store.ClaimAvailablePlanCheckRuns(ctx)
	if err != nil {
		slog.Error("failed to claim available plan check runs", log.BBError(err))
		return
	}

	for _, c := range claimed {
		go s.runPlanCheckRun(ctx, c.ProjectID, c.UID, c.PlanUID, c.ApprovalInputVersion)
	}
}

func (s *Scheduler) runPlanCheckRun(ctx context.Context, projectID string, uid int64, planUID int64, approvalInputVersion int64) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	planCheckRef := bus.PlanCheckRunRef{ProjectID: projectID, UID: uid, ApprovalInputVersion: approvalInputVersion}
	s.bus.RunningPlanCheckRunsCancelFunc.Store(planCheckRef, cancel)
	defer s.bus.RunningPlanCheckRunsCancelFunc.Delete(planCheckRef)

	// Fetch plan to derive check targets at runtime
	plan, err := s.store.GetPlan(ctxWithCancel, &store.FindPlanMessage{ProjectID: projectID, UID: &planUID})
	if err != nil {
		s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, err.Error())
		return
	}
	if plan == nil {
		s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, "plan not found")
		return
	}

	if plan.Config.GetApprovalInputVersion() != approvalInputVersion {
		slog.Info("skip stale plan check run",
			slog.String("project", projectID),
			slog.Int64("plan_id", planUID),
			slog.Int64("plan_check_run_id", uid),
			slog.Int64("claimed_approval_input_version", approvalInputVersion),
			slog.Int64("current_approval_input_version", plan.Config.GetApprovalInputVersion()))
		s.markPlanCheckRunCanceled(ctxWithCancel, projectID, uid, approvalInputVersion, "stale plan check run")
		return
	}

	project, err := s.store.GetProjectByResourceID(ctxWithCancel, plan.ProjectID)
	if err != nil {
		s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, err.Error())
		return
	}
	if project == nil {
		s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, "project not found")
		return
	}

	// Get database group if needed (for spec expansion)
	databaseGroup, err := GetDatabaseGroupForPlan(ctxWithCancel, s.store, plan, nil)
	if err != nil {
		s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, err.Error())
		return
	}

	// Derive check targets from plan
	targets, err := DeriveCheckTargets(ctxWithCancel, s.store, project, plan, databaseGroup)
	if err != nil {
		s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, err.Error())
		return
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, target := range targets {
		targetResults, targetErr := s.executor.RunForTarget(ctxWithCancel, target)
		if targetErr != nil {
			err = targetErr
			break
		}
		results = append(results, targetResults...)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.markPlanCheckRunCanceled(ctxWithCancel, projectID, uid, approvalInputVersion, err.Error())
		} else {
			s.markPlanCheckRunFailed(ctxWithCancel, projectID, uid, approvalInputVersion, err.Error())
		}
	} else {
		s.markPlanCheckRunDone(ctxWithCancel, projectID, uid, planUID, approvalInputVersion, results)
	}
}

func (s *Scheduler) markPlanCheckRunDone(ctx context.Context, projectID string, uid int64, planUID int64, approvalInputVersion int64, results []*storepb.PlanCheckRunResult_Result) {
	result := &storepb.PlanCheckRunResult{
		Results:              results,
		ApprovalInputVersion: approvalInputVersion,
	}
	updated, err := s.store.UpdatePlanCheckRunIfApprovalInputVersion(ctx,
		projectID,
		store.PlanCheckRunStatusDone,
		result,
		uid,
		approvalInputVersion,
	)
	if err != nil {
		slog.Error("failed to mark plan check run done", log.BBError(err))
		return
	}
	if !updated {
		slog.Info("skip stale plan check run done update",
			slog.String("project", projectID),
			slog.Int64("plan_id", planUID),
			slog.Int64("plan_check_run_id", uid),
			slog.Int64("claimed_approval_input_version", approvalInputVersion))
		return
	}

	// Trigger approval finding after plan checks complete.
	// The approval runner will trigger rollout creation after it finishes.
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectID}, PlanUID: &planUID})
	if err != nil {
		slog.Error("failed to get issue for approval check after plan check",
			slog.String("project", projectID), slog.Int("plan_id", int(planUID)),
			log.BBError(err))
		return
	}
	if issue != nil && issue.PlanUID != nil && !issue.Payload.GetDraft() {
		// Trigger approval finding.
		s.bus.ApprovalCheckChan <- bus.IssueRef{ProjectID: projectID, UID: issue.UID}
	}
}

func (s *Scheduler) markPlanCheckRunFailed(ctx context.Context, projectID string, uid int64, approvalInputVersion int64, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error:                reason,
		ApprovalInputVersion: approvalInputVersion,
	}
	updated, err := s.store.UpdatePlanCheckRunIfApprovalInputVersion(ctx,
		projectID,
		store.PlanCheckRunStatusFailed,
		result,
		uid,
		approvalInputVersion,
	)
	if err != nil {
		slog.Error("failed to mark plan check run failed", log.BBError(err))
		return
	}
	if !updated {
		slog.Info("skip stale plan check run failed update",
			slog.String("project", projectID),
			slog.Int64("plan_check_run_id", uid),
			slog.Int64("claimed_approval_input_version", approvalInputVersion))
	}
}

func (s *Scheduler) markPlanCheckRunCanceled(ctx context.Context, projectID string, uid int64, approvalInputVersion int64, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error:                reason,
		ApprovalInputVersion: approvalInputVersion,
	}
	updated, err := s.store.UpdatePlanCheckRunIfApprovalInputVersion(ctx,
		projectID,
		store.PlanCheckRunStatusCanceled,
		result,
		uid,
		approvalInputVersion,
	)
	if err != nil {
		slog.Error("failed to mark plan check run canceled", log.BBError(err))
		return
	}
	if !updated {
		slog.Info("skip stale plan check run canceled update",
			slog.String("project", projectID),
			slog.Int64("plan_check_run_id", uid),
			slog.Int64("claimed_approval_input_version", approvalInputVersion))
	}
}
