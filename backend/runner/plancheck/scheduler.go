// Package plancheck is the runner for plan checks.
package plancheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	planCheckSchedulerInterval = time.Duration(1) * time.Second
)

func NewScheduler(s *store.Store, licenseService enterpriseAPI.LicenseService, stateCfg *state.State) *Scheduler {
	return &Scheduler{
		store:          s,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		executors:      make(map[store.PlanCheckRunType]Executor),
	}
}

type Scheduler struct {
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
	stateCfg       *state.State
	executors      map[store.PlanCheckRunType]Executor
}

func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(planCheckSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Plan check scheduler started and will run every %v", planCheckSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

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
			log.Error("Plan check scheduler PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		Status: &[]store.PlanCheckRunStatus{
			store.PlanCheckRunStatusRunning,
		},
	})
	if err != nil {
		log.Error("failed to list running plan check runs", zap.Error(err))
		return
	}

	for _, planCheckRun := range planCheckRuns {
		s.runPlanCheckRun(ctx, planCheckRun)
	}
}

func (s *Scheduler) runPlanCheckRun(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) {
	executor, ok := s.executors[planCheckRun.Type]
	if !ok {
		log.Error("Skip running plan check for unknown type", zap.Int("uid", planCheckRun.UID), zap.Int("plan_uid", planCheckRun.PlanUID), zap.String("type", string(planCheckRun.Type)))
		return
	}
	if _, ok := s.stateCfg.RunningPlanChecks.Load(planCheckRun.UID); ok {
		return
	}

	databaseUID := int(planCheckRun.Config.DatabaseId)
	db, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseUID})
	if err != nil {
		log.Error("failed to get db for plan check run")
		s.markPlanCheckRunFailed(ctx, planCheckRun, errors.Wrapf(err, "failed to get database for plan check run").Error())
		return
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &db.InstanceID,
	})
	if err != nil {
		log.Error("failed to get instance for plan check run")
		s.markPlanCheckRunFailed(ctx, planCheckRun, errors.Wrapf(err, "failed to get instance for plan check run").Error())
		return
	}

	s.stateCfg.Lock()
	if s.stateCfg.InstanceOutstandingConnections[instance.UID] >= state.InstanceMaximumConnectionNumber {
		s.stateCfg.Unlock()
		return
	}
	s.stateCfg.InstanceOutstandingConnections[instance.UID]++
	s.stateCfg.Unlock()

	s.stateCfg.RunningPlanChecks.Store(planCheckRun.UID, true)
	go func() {
		defer func() {
			s.stateCfg.RunningPlanChecks.Delete(planCheckRun.UID)
			s.stateCfg.Lock()
			s.stateCfg.InstanceOutstandingConnections[instance.UID]--
			s.stateCfg.Unlock()
		}()
		results, err := executor.Run(ctx, planCheckRun)
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}
		s.markPlanCheckRunDone(ctx, planCheckRun, results)
	}()
}

func (s *Scheduler) markPlanCheckRunDone(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, results []*storepb.PlanCheckRunResult_Result) {
	result := &storepb.PlanCheckRunResult{
		Results: results,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		api.SystemBotID,
		store.PlanCheckRunStatusDone,
		result,
		planCheckRun.UID,
	); err != nil {
		log.Error("failed to mark plan check run failed", zap.Error(err))
	}
}

func (s *Scheduler) markPlanCheckRunFailed(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error: reason,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		api.SystemBotID,
		store.PlanCheckRunStatusFailed,
		result,
		planCheckRun.UID,
	); err != nil {
		log.Error("failed to mark plan check run failed", zap.Error(err))
	}
}
