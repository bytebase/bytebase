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
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	planCheckSchedulerInterval = 5 * time.Second
)

// NewScheduler creates a new plan check scheduler.
func NewScheduler(s *store.Store, licenseService *enterprise.LicenseService, stateCfg *state.State) *Scheduler {
	return &Scheduler{
		store:          s,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		executors:      make(map[store.PlanCheckRunType]Executor),
	}
}

// Scheduler is the plan check run scheduler.
type Scheduler struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
	stateCfg       *state.State
	executors      map[store.PlanCheckRunType]Executor
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
		case <-s.stateCfg.PlanCheckTickleChan:
			s.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// Register registers a plan check executor.
func (s *Scheduler) Register(planCheckRunType store.PlanCheckRunType, executor Executor) {
	if executor == nil {
		panic("plan check scheduler: Register executor is nil for plan check run type: " + planCheckRunType)
	}
	if _, dup := s.executors[planCheckRunType]; dup {
		panic("plan check scheduler: Register called twice for plan check run type: " + planCheckRunType)
	}
	s.executors[planCheckRunType] = executor
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
	executor, ok := s.executors[planCheckRun.Type]
	if !ok {
		slog.Error("Skip running plan check for unknown type", slog.Int("uid", planCheckRun.UID), slog.Int64("plan_uid", planCheckRun.PlanUID), slog.String("type", string(planCheckRun.Type)))
		return
	}
	// Skip the plan check run if it is already running.
	if _, ok := s.stateCfg.RunningPlanChecks.Load(planCheckRun.UID); ok {
		return
	}

	s.stateCfg.RunningPlanChecks.Store(planCheckRun.UID, true)
	go func() {
		defer func() {
			s.stateCfg.RunningPlanChecks.Delete(planCheckRun.UID)
			s.stateCfg.RunningPlanCheckRunsCancelFunc.Delete(planCheckRun.UID)
		}()

		ctxWithCancel, cancel := context.WithCancel(ctx)
		defer cancel()
		s.stateCfg.RunningPlanCheckRunsCancelFunc.Store(planCheckRun.UID, cancel)

		results, err := runExecutorOnce(ctxWithCancel, executor, planCheckRun.Config)
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
