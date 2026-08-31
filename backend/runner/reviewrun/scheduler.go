// Package reviewrun is the runner executing SQL review runs.
//
// The execution model follows the task run runner: claim every AVAILABLE
// slot for this replica, dispatch serially, execute each run in its own
// goroutine, and guarantee a terminal write on every path. There is no
// cancel — re-running supersedes a RUNNING execution, whose fenced
// completion then matches zero rows — and no execution timeout: a run on a
// dead replica is failed by the heartbeat reaper, and a hung run on a live
// replica is superseded by the next re-run.
package reviewrun

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/productmetrics"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const reviewRunSchedulerInterval = 5 * time.Second

// Executor executes one review run for one reviewer type.
type Executor interface {
	// RunOnce evaluates every (spec, target) unit of the issue's plan
	// (collect-all, no fail-fast). A nil error marks the run DONE; a non-nil
	// error marks it FAILED with the aggregated message. Findings will
	// surface as issue comments once the comment integration lands; until
	// then executors only report status.
	RunOnce(ctx context.Context, projectID string, issueUID int64) error
}

// NewScheduler creates a new review run scheduler.
func NewScheduler(s *store.Store, b *bus.Bus, profile *config.Profile, licenseService *enterprise.LicenseService, productMetrics *productmetrics.ProductMetrics) *Scheduler {
	return &Scheduler{
		store:          s,
		bus:            b,
		profile:        profile,
		licenseService: licenseService,
		productMetrics: productMetrics,
		executorMap:    make(map[string]Executor),
	}
}

// Scheduler is the review run scheduler.
type Scheduler struct {
	store          *store.Store
	bus            *bus.Bus
	profile        *config.Profile
	licenseService *enterprise.LicenseService
	productMetrics *productmetrics.ProductMetrics
	executorMap    map[string]Executor
}

// Register registers an executor for a reviewer type. Call before Run.
func (s *Scheduler) Register(reviewType string, executor Executor) {
	if executor == nil {
		panic(fmt.Sprintf("registered nil executor for reviewer type %q", reviewType))
	}
	s.executorMap[reviewType] = executor
}

// Run runs the scheduler.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(reviewRunSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Review run scheduler started and will run every %v", reviewRunSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			if err := s.licenseService.CheckReplicaLimit(ctx); err != nil {
				slog.Warn("Review run scheduler skipped due to HA license restriction", log.BBError(err))
				continue
			}
			if err := s.scheduleReviewRuns(ctx); err != nil {
				slog.Error("failed to schedule review runs", log.BBError(err))
			}
		case <-s.bus.ReviewRunTickleChan:
			if err := s.licenseService.CheckReplicaLimit(ctx); err != nil {
				slog.Warn("Review run scheduler skipped due to HA license restriction", log.BBError(err))
				continue
			}
			if err := s.scheduleReviewRuns(ctx); err != nil {
				slog.Error("failed to schedule review runs", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) scheduleReviewRuns(ctx context.Context) (retErr error) {
	startedAt := time.Now()
	result := productmetrics.ResultFailure
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			retErr = errors.Wrap(panicErr, "review run scheduler panic")
			slog.Error("Review run scheduler PANIC RECOVER", log.BBError(retErr), log.BBStack("panic-stack"))
		}
		if !errors.Is(ctx.Err(), context.Canceled) && s.productMetrics != nil {
			s.productMetrics.RecordRunnerRun(productmetrics.RunnerReviewRun, result, time.Since(startedAt))
		}
	}()

	claimed, err := s.store.ClaimAvailableReviewRuns(ctx, s.profile.ReplicaID)
	if err != nil {
		return errors.Wrapf(err, "failed to claim available review runs")
	}
	for _, c := range claimed {
		s.dispatchReviewRun(ctx, c)
	}
	result = productmetrics.ResultSuccess
	return nil
}

// dispatchReviewRun starts one claimed run's execution. A reviewer type with
// no registered executor fails immediately: the type space is open, and a
// claimed run must always reach a terminal status.
func (s *Scheduler) dispatchReviewRun(ctx context.Context, claimed *store.ClaimedReviewRun) {
	executor, ok := s.executorMap[claimed.Type]
	if !ok {
		s.completeReviewRun(ctx, claimed, errors.Errorf("no executor registered for reviewer type %q", claimed.Type))
		return
	}
	go s.runReviewRunOnce(ctx, claimed, executor)
}

func (s *Scheduler) runReviewRunOnce(ctx context.Context, claimed *store.ClaimedReviewRun, executor Executor) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("runReviewRunOnce PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	err := runExecutorOnce(ctx, executor, claimed)
	if err != nil && errors.Is(err, context.Canceled) {
		// Shutdown: the completion write would fail on the same canceled
		// context. The replica's heartbeat stops with the process, so the
		// reaper fails this run.
		slog.Info("review run interrupted by shutdown; leaving it to the reaper",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID),
			slog.String("type", claimed.Type))
		return
	}
	s.completeReviewRun(ctx, claimed, err)
}

// completeReviewRun writes the terminal status: DONE on a nil error, FAILED
// with the aggregated message otherwise. Zero rows means the run was
// superseded or reaped since the claim; the result is discarded.
func (s *Scheduler) completeReviewRun(ctx context.Context, claimed *store.ClaimedReviewRun, execErr error) {
	status := storepb.ReviewRun_DONE
	payload := &storepb.ReviewRunPayload{}
	if execErr != nil {
		status = storepb.ReviewRun_FAILED
		payload.Error = execErr.Error()
	}
	updated, err := s.store.CompleteReviewRun(ctx, claimed, s.profile.ReplicaID, status, payload)
	if err != nil {
		slog.Error("failed to complete review run",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID),
			slog.String("type", claimed.Type),
			log.BBError(err))
		return
	}
	if !updated {
		slog.Info("skip completing superseded review run",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID),
			slog.String("type", claimed.Type),
			slog.Int64("attempt", claimed.Attempt))
	}
}

// runExecutorOnce wraps Executor.RunOnce with panic recovery so a panicking
// executor fails its run instead of crashing the process.
func runExecutorOnce(ctx context.Context, executor Executor, claimed *store.ClaimedReviewRun) (err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			slog.Error("Review run executor PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
			err = errors.Errorf("review run executor panic: %v", panicErr)
		}
	}()
	return executor.RunOnce(ctx, claimed.ProjectID, claimed.IssueUID)
}
