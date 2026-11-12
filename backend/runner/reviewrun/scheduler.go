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

// Executor executes a review run, or one reviewer's part of it.
type Executor interface {
	// RunOnce evaluates every (spec, target) unit of the issue's plan in
	// scope (collect-all, no fail-fast) and returns the results to post as
	// issue comments, sorted by sortReviewResults. A nil error marks the run
	// DONE and posts the results; a non-nil error marks it FAILED with the
	// aggregated message and posts nothing.
	RunOnce(ctx context.Context, projectID string, issueUID int64) ([]*store.IssueCommentMessage, error)
}

// NewScheduler creates a new review run scheduler that runs every review run
// with executor.
func NewScheduler(s *store.Store, b *bus.Bus, profile *config.Profile, licenseService *enterprise.LicenseService, productMetrics *productmetrics.ProductMetrics, executor Executor) *Scheduler {
	return &Scheduler{
		store:          s,
		bus:            b,
		profile:        profile,
		licenseService: licenseService,
		productMetrics: productMetrics,
		executor:       executor,
	}
}

// Scheduler is the review run scheduler.
type Scheduler struct {
	store          *store.Store
	bus            *bus.Bus
	profile        *config.Profile
	licenseService *enterprise.LicenseService
	productMetrics *productmetrics.ProductMetrics
	executor       Executor

	// runs tracks the goroutines dispatchReviewRun spawns; Run waits for them,
	// or they outlive the store.
	runs sync.WaitGroup
}

// Run runs the scheduler.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(reviewRunSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	defer s.runs.Wait()
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
		s.runs.Go(func() { s.runReviewRunOnce(ctx, c) })
	}
	result = productmetrics.ResultSuccess
	return nil
}

func (s *Scheduler) runReviewRunOnce(ctx context.Context, claimed *store.ClaimedReviewRun) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("runReviewRunOnce PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	results, err := runExecutorOnce(ctx, s.executor, claimed)
	if err != nil && errors.Is(err, context.Canceled) {
		// Shutdown: the completion write would fail on the same canceled
		// context. The replica's heartbeat stops with the process, so the
		// reaper fails this run.
		slog.Info("review run interrupted by shutdown; leaving it to the reaper",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID))
		return
	}
	s.completeReviewRun(ctx, claimed, results, err)
}

// completeReviewRun writes the terminal status: DONE with the results posted
// on a nil error, FAILED with the aggregated message otherwise. Zero rows
// means the run was superseded or reaped since the claim; the results are
// discarded.
func (s *Scheduler) completeReviewRun(ctx context.Context, claimed *store.ClaimedReviewRun, results []*store.IssueCommentMessage, execErr error) {
	status := storepb.ReviewRun_DONE
	payload := &storepb.ReviewRunPayload{}
	if execErr != nil {
		status = storepb.ReviewRun_FAILED
		payload.Error = execErr.Error()
		payload.Code = errorCode(execErr)
		results = nil
	}
	updated, err := s.store.CompleteReviewRun(ctx, claimed, s.profile.ReplicaID, status, payload, results)
	if err != nil {
		slog.Error("failed to complete review run",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID),
			log.BBError(err))
		return
	}
	if !updated {
		slog.Info("skip completing superseded review run",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID),
			slog.Int64("attempt", claimed.Attempt))
		return
	}
	if s.profile.SQLReviewV2 {
		s.requestRolloutCheck(ctx, claimed)
	}
}

// requestRolloutCheck asks the rollout creator to look at the issue's plan
// again: under SQL Review V2 the review holds the automatic rollout until it
// completes.
func (s *Scheduler) requestRolloutCheck(ctx context.Context, claimed *store.ClaimedReviewRun) {
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{claimed.ProjectID}, UID: &claimed.IssueUID})
	if err != nil {
		slog.Warn("failed to get the issue to check its rollout",
			slog.String("project", claimed.ProjectID),
			slog.Int64("issue_id", claimed.IssueUID),
			log.BBError(err))
		return
	}
	if issue == nil || issue.PlanUID == nil {
		return
	}
	select {
	case s.bus.RolloutCreationChan <- bus.PlanRef{ProjectID: claimed.ProjectID, PlanID: *issue.PlanUID}:
	case <-ctx.Done():
	}
}

// runExecutorOnce wraps Executor.RunOnce with panic recovery so a panicking
// executor fails its run instead of crashing the process.
func runExecutorOnce(ctx context.Context, executor Executor, claimed *store.ClaimedReviewRun) (results []*store.IssueCommentMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			slog.Error("Review run executor PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
			results = nil
			err = errors.Errorf("review run executor panic: %v", panicErr)
		}
	}()
	return executor.RunOnce(ctx, claimed.ProjectID, claimed.IssueUID)
}
