package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// GetReviewRun returns the issue's review run, and whether the plan's SQL
// changed after it started.
func (s *IssueService) GetReviewRun(ctx context.Context, req *connect.Request[v1pb.GetReviewRunRequest]) (*connect.Response[v1pb.ReviewRun], error) {
	projectID, issueUID, err := common.GetProjectIDIssueUIDFromReviewRun(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	jobs, err := ReviewJobs(ctx, s.store, projectID, issueUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(jobs) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("review run %q not found", req.Msg.Name))
	}
	converted := convertToReviewRun(projectID, issueUID, jobs)
	// Every job of a run reviews the same version of the SQL.
	converted.Outdated, err = reviewRunOutdated(ctx, s.store, jobs[0])
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(converted), nil
}

// ReviewJobs returns the reviewers' jobs the issue's review run is read from:
// the rule job, and the AI job of the same SQL version. Each reviewer runs as
// a job of its own (BYT-10505). The rollout creator waits for the same jobs,
// so the run a caller reads and the rollout agree.
//
// No setting is read here. A job changes only when it runs, and every run
// asks the rollout creator to look again, so nothing a caller reads changes
// without the rollout hearing of it.
func ReviewJobs(ctx context.Context, stores *store.Store, projectID string, issueUID int64) ([]*store.ReviewRunMessage, error) {
	var jobs []*store.ReviewRunMessage
	ruleJob, err := stores.GetReviewRun(ctx, projectID, issueUID, store.ReviewRunTypeRule)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the rule review run")
	}
	if ruleJob != nil {
		jobs = append(jobs, ruleJob)
	}
	aiJob, err := stores.GetReviewRun(ctx, projectID, issueUID, store.ReviewRunTypeAI)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the AI review run")
	}
	// An AI job of another version is not part of this run: the rule job
	// restarted alone on newer SQL, as the rollout creator restarts it.
	if aiJob != nil && (ruleJob == nil || aiJob.Payload.GetApprovalInputVersion() == ruleJob.Payload.GetApprovalInputVersion()) {
		jobs = append(jobs, aiJob)
	}
	return jobs, nil
}

// reviewRunOutdated reports whether the run's approval input version is no
// longer its issue's plan's: the plan's SQL changed after the run started.
func reviewRunOutdated(ctx context.Context, stores *store.Store, reviewRun *store.ReviewRunMessage) (bool, error) {
	issue, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		ProjectIDs: []string{reviewRun.ProjectID},
		UID:        &reviewRun.IssueUID,
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get issue")
	}
	if issue == nil || issue.PlanUID == nil {
		return false, nil
	}
	plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: reviewRun.ProjectID, UID: issue.PlanUID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return false, nil
	}
	return reviewRun.Payload.GetApprovalInputVersion() != plan.Config.GetApprovalInputVersion(), nil
}

// RunReview starts the issue's review. Every reviewer's job is reset
// unconditionally, which supersedes an execution in flight, and the returned
// run is RUNNING.
func (s *IssueService) RunReview(ctx context.Context, req *connect.Request[v1pb.RunReviewRequest]) (*connect.Response[v1pb.ReviewRun], error) {
	projectID, issueUID, err := common.GetProjectIDIssueUIDFromReviewRun(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		ProjectIDs: []string{projectID},
		UID:        &issueUID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get issue"))
	}
	if issue == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("issue %d not found in project %s", issueUID, projectID))
	}
	if err := checkReviewable(ctx, s.store, issue); err != nil {
		return nil, err
	}
	reviewTypes, err := s.startableReviewTypes(ctx, projectID, issueUID)
	if err != nil {
		return nil, err
	}

	jobs := make([]*store.ReviewRunMessage, 0, len(reviewTypes))
	for _, reviewType := range reviewTypes {
		job, err := s.store.CreateReviewRun(ctx, projectID, issueUID, reviewType)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create review run"))
		}
		jobs = append(jobs, job)
	}

	s.bus.ReviewRunTickleChan <- 0

	return connect.NewResponse(convertToReviewRun(projectID, issueUID, jobs)), nil
}

// startableReviewTypes returns the reviewers a review of the issue starts
// now, in the order their jobs restart: the AI review, and the standard rules
// where SQL Review V2 is on. With neither, the error is why the AI review is
// off.
//
// The jobs restart one write at a time, and the AI job goes first. The rule
// job is quick, so restarted first it could finish beside the AI job's
// earlier result, and the two would read as a finished review.
func (s *IssueService) startableReviewTypes(ctx context.Context, projectID string, issueUID int64) ([]string, error) {
	var reviewTypes []string
	startsAI, aiErr, err := aiReviewStarts(ctx, s.profile, s.store, projectID, issueUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if startsAI {
		reviewTypes = append(reviewTypes, store.ReviewRunTypeAI)
	}
	if s.profile.SQLReviewV2 {
		reviewTypes = append(reviewTypes, store.ReviewRunTypeRule)
	}
	if len(reviewTypes) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, aiErr)
	}
	return reviewTypes, nil
}

// aiReviewStarts reports whether a review of the issue restarts the AI job:
// while the workspace allows the AI review, and under SQL Review V2 also once
// the issue has an AI job. Switched off, that run reaches no provider and
// ends with no finding, which resolves the findings the job left and clears
// its failure. Left alone, a failed AI job would hold the rollout with no way
// to retry it. aiErr is why the workspace does not allow the AI review.
func aiReviewStarts(ctx context.Context, profile *config.Profile, stores *store.Store, projectID string, issueUID int64) (starts bool, aiErr error, err error) {
	aiSetting, err := stores.GetAISettingUncached(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to get AI setting")
	}
	aiErr = aireview.CheckSetting(aiSetting)
	if aiErr == nil {
		return true, nil, nil
	}
	if !profile.SQLReviewV2 {
		return false, aiErr, nil
	}
	aiJob, err := stores.GetReviewRun(ctx, projectID, issueUID, store.ReviewRunTypeAI)
	if err != nil {
		return false, aiErr, errors.Wrapf(err, "failed to get the AI review run")
	}
	return aiJob != nil, aiErr, nil
}

// checkReviewable refuses a review the issue cannot have now: the issue must
// be open, and its plan's SQL must be unfrozen by a rollout and reach at least
// one reviewable target.
func checkReviewable(ctx context.Context, stores *store.Store, issue *store.IssueMessage) error {
	if issue.Status != storepb.Issue_OPEN {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("review runs only on an open issue; issue is %s", issue.Status))
	}
	if issue.PlanUID == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("issue %d has no SQL to review", issue.UID))
	}
	plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{ProjectID: issue.ProjectID, UID: issue.PlanUID})
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
	}
	if plan == nil {
		return connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", *issue.PlanUID, issue.ProjectID))
	}
	if plan.Config.GetHasRollout() {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("the plan already has a rollout; its SQL is frozen"))
	}
	project, err := stores.GetProjectByResourceID(ctx, issue.ProjectID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", issue.ProjectID))
	}
	hasTargets, err := plancheck.HasReviewTargets(ctx, stores, project, plan)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if !hasTargets {
		return connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("the plan has no reviewable SQL"))
	}
	return nil
}

// startReviews starts the issue's reviews of the given types, in the given
// order, where SQL Review V2 is on, at the points the plan checks run. The AI
// review starts as aiReviewStarts says. A caller that starts both names the
// AI review first, as startableReviewTypes does. It never fails its caller:
// an issue the reviews do not apply to is skipped, and a failure is logged.
func startReviews(ctx context.Context, profile *config.Profile, stores *store.Store, b *bus.Bus, issue *store.IssueMessage, reviewTypes ...string) {
	if !profile.SQLReviewV2 || issue == nil || issue.Type != storepb.Issue_DATABASE_CHANGE {
		return
	}
	if err := checkReviewable(ctx, stores, issue); err != nil {
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			slog.Warn("failed to start the review",
				slog.String("project", issue.ProjectID),
				slog.Int64("issue_id", issue.UID),
				log.BBError(err))
		}
		return
	}
	started := false
	for _, reviewType := range reviewTypes {
		if reviewType == store.ReviewRunTypeAI {
			startsAI, _, err := aiReviewStarts(ctx, profile, stores, issue.ProjectID, issue.UID)
			if err != nil {
				slog.Warn("failed to tell whether the AI review starts",
					slog.String("project", issue.ProjectID),
					slog.Int64("issue_id", issue.UID),
					log.BBError(err))
				continue
			}
			if !startsAI {
				continue
			}
		}
		if _, err := stores.CreateReviewRun(ctx, issue.ProjectID, issue.UID, reviewType); err != nil {
			slog.Error("failed to start the review",
				slog.String("project", issue.ProjectID),
				slog.Int64("issue_id", issue.UID),
				slog.String("type", reviewType),
				log.BBError(err))
			continue
		}
		started = true
	}
	if started {
		b.ReviewRunTickleChan <- 0
	}
}

// requestRolloutCheck asks the rollout creator to look at the issue's plan
// again. Under SQL Review V2 open threads hold the automatic rollout, so
// resolving one can release it.
func requestRolloutCheck(ctx context.Context, stores *store.Store, b *bus.Bus, projectID string, issueUID int64) {
	issue, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectID}, UID: &issueUID})
	if err != nil {
		slog.Warn("failed to get the issue to check its rollout",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			log.BBError(err))
		return
	}
	if issue == nil || issue.PlanUID == nil {
		return
	}
	select {
	case b.RolloutCreationChan <- bus.PlanRef{ProjectID: projectID, PlanID: *issue.PlanUID}:
	case <-ctx.Done():
	}
}

// convertToReviewRun reads the reviewers' jobs as one run: RUNNING while any
// job is queued or running, else FAILED with the error of the first job that
// failed, else SUCCEEDED. The run was triggered when its latest job was, and
// ended when its last job did.
func convertToReviewRun(projectID string, issueUID int64, jobs []*store.ReviewRunMessage) *v1pb.ReviewRun {
	converted := &v1pb.ReviewRun{
		Name:  common.FormatReviewRun(projectID, issueUID),
		State: v1pb.ReviewRun_SUCCEEDED,
	}
	var failed, created, ended *store.ReviewRunMessage
	running := false
	for _, job := range jobs {
		if created == nil || job.CreatedAt.After(created.CreatedAt) {
			created = job
		}
		// A terminal row is never written again, so updated_at is the end time.
		if ended == nil || job.UpdatedAt.After(ended.UpdatedAt) {
			ended = job
		}
		switch job.Status {
		case storepb.ReviewRun_DONE:
		case storepb.ReviewRun_FAILED:
			if failed == nil {
				failed = job
			}
		default:
			running = true
		}
	}
	if created != nil {
		converted.CreateTime = timestamppb.New(created.CreatedAt)
	}
	switch {
	case running:
		converted.State = v1pb.ReviewRun_RUNNING
	case failed != nil:
		converted.State = v1pb.ReviewRun_FAILED
		converted.Error = &spb.Status{
			Code:    int32(connect.CodeUnknown),
			Message: failed.Payload.GetError(),
		}
	default:
	}
	if !running && ended != nil {
		converted.EndTime = timestamppb.New(ended.UpdatedAt)
	}
	return converted
}
