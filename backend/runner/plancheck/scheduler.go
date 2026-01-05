// Package plancheck is the runner for plan checks.
package plancheck

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	planCheckSchedulerInterval = 5 * time.Second
)

// NewScheduler creates a new plan check scheduler.
func NewScheduler(s *store.Store, bus *bus.Bus, executor *CombinedExecutor) *Scheduler {
	return &Scheduler{
		store:    s,
		bus:      bus,
		executor: executor,
	}
}

// Scheduler is the plan check run scheduler.
type Scheduler struct {
	store    *store.Store
	bus      *bus.Bus
	executor *CombinedExecutor
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
			s.runOnce(ctx)
		case <-s.bus.PlanCheckTickleChan:
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

	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		Status: &[]store.PlanCheckRunStatus{
			store.PlanCheckRunStatusRunning,
		},
	})
	if err != nil {
		slog.Error("failed to list running plan check runs", log.BBError(err))
		return
	}

	for _, planCheckRun := range planCheckRuns {
		s.runPlanCheckRun(ctx, planCheckRun)
	}
}

func (s *Scheduler) runPlanCheckRun(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) {
	// Skip the plan check run if it is already running.
	if _, ok := s.bus.RunningPlanChecks.Load(planCheckRun.UID); ok {
		return
	}

	s.bus.RunningPlanChecks.Store(planCheckRun.UID, true)
	go func() {
		defer func() {
			s.bus.RunningPlanChecks.Delete(planCheckRun.UID)
			s.bus.RunningPlanCheckRunsCancelFunc.Delete(planCheckRun.UID)
		}()

		ctxWithCancel, cancel := context.WithCancel(ctx)
		defer cancel()
		s.bus.RunningPlanCheckRunsCancelFunc.Store(planCheckRun.UID, cancel)

		// Fetch plan to derive check targets at runtime
		plan, err := s.store.GetPlan(ctxWithCancel, &store.FindPlanMessage{UID: &planCheckRun.PlanUID})
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}
		if plan == nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, "plan not found")
			return
		}

		project, err := s.store.GetProject(ctxWithCancel, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}
		if project == nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, "project not found")
			return
		}

		// Get database group if needed (for spec expansion)
		databaseGroup, err := s.getDatabaseGroupForPlan(ctxWithCancel, plan)
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}

		// Derive check targets from plan
		targets, err := DeriveCheckTargets(project, plan, databaseGroup)
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
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
				s.markPlanCheckRunCanceled(ctx, planCheckRun, err.Error())
			} else {
				s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			}
		} else {
			s.markPlanCheckRunDone(ctx, planCheckRun, results)
		}
	}()
}

func (s *Scheduler) markPlanCheckRunDone(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, results []*storepb.PlanCheckRunResult_Result) {
	result := &storepb.PlanCheckRunResult{
		Results: results,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusDone,
		result,
		planCheckRun.UID,
	); err != nil {
		slog.Error("failed to mark plan check run done", log.BBError(err))
		return
	}

	// Auto-create rollout if plan checks pass
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &planCheckRun.PlanUID})
	if err != nil {
		slog.Error("failed to get issue for approval check after plan check",
			slog.Int("plan_id", int(planCheckRun.PlanUID)),
			log.BBError(err))
		return
	}
	if issue != nil && issue.PlanUID != nil {
		// Trigger approval finding
		s.bus.ApprovalCheckChan <- int64(issue.UID)
		// Trigger rollout creation (existing behavior)
		s.bus.RolloutCreationChan <- planCheckRun.PlanUID
	}
}

func (s *Scheduler) markPlanCheckRunFailed(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error: reason,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusFailed,
		result,
		planCheckRun.UID,
	); err != nil {
		slog.Error("failed to mark plan check run failed", log.BBError(err))
	}
}

func (s *Scheduler) markPlanCheckRunCanceled(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error: reason,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusCanceled,
		result,
		planCheckRun.UID,
	); err != nil {
		slog.Error("failed to mark plan check run canceled", log.BBError(err))
	}
}

// getDatabaseGroupForPlan checks if the plan targets a database group and returns it with matched databases.
// Returns nil if the plan does not target a database group.
func (s *Scheduler) getDatabaseGroupForPlan(ctx context.Context, plan *store.PlanMessage) (*v1pb.DatabaseGroup, error) {
	for _, spec := range plan.Config.Specs {
		cfg, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig)
		if !ok {
			continue
		}
		if len(cfg.ChangeDatabaseConfig.Targets) != 1 {
			continue
		}

		target := cfg.ChangeDatabaseConfig.Targets[0]
		_, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(target)
		if err != nil {
			// Not a database group reference, skip
			continue
		}

		// Found a database group reference - fetch and expand it
		dbGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
			ResourceID: &databaseGroupID,
			ProjectID:  &plan.ProjectID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database group %q", target)
		}
		if dbGroup == nil {
			return nil, errors.Errorf("database group %q not found", target)
		}

		// Get all databases in the project to compute matches
		allDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &plan.ProjectID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list databases for project %q", plan.ProjectID)
		}

		// Compute matched databases using CEL expression
		matchedDatabases, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, dbGroup, allDatabases)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get matched databases for group %q", databaseGroupID)
		}

		// Convert to v1pb.DatabaseGroup format
		result := &v1pb.DatabaseGroup{
			Name: target,
		}
		for _, db := range matchedDatabases {
			result.MatchedDatabases = append(result.MatchedDatabases, &v1pb.DatabaseGroup_Database{
				Name: common.FormatDatabase(db.InstanceID, db.DatabaseName),
			})
		}
		return result, nil
	}
	return nil, nil
}
