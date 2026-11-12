package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/code"
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

	reviewRun, err := s.store.GetReviewRun(ctx, projectID, issueUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get review run"))
	}
	if reviewRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("review run %q not found", req.Msg.Name))
	}
	converted := convertToReviewRun(reviewRun)
	converted.Outdated, err = reviewRunOutdated(ctx, s.store, reviewRun)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(converted), nil
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

// RunReview starts the issue's review. The slot is reset unconditionally,
// which supersedes an execution in flight, and the returned run is RUNNING.
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
	if err := s.checkAReviewerRuns(ctx); err != nil {
		return nil, err
	}

	reviewRun, err := s.store.CreateReviewRun(ctx, projectID, issueUID)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create review run"))
	}

	s.bus.ReviewRunTickleChan <- 0

	return connect.NewResponse(convertToReviewRun(reviewRun)), nil
}

// checkAReviewerRuns refuses a review that no reviewer would run. A server
// without SQL Review V2 has no standard rule review, so there the AI review
// has to be on; the error says why it is off.
func (s *IssueService) checkAReviewerRuns(ctx context.Context) error {
	if s.profile.SQLReviewV2 {
		return nil
	}
	aiSetting, err := s.store.GetAISettingUncached(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get AI setting"))
	}
	if err := aireview.CheckSetting(aiSetting); err != nil {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return nil
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

// startReview starts the issue's review where SQL Review V2 is on, at the
// points the plan's SQL is submitted or changed. It never fails its caller:
// an issue the review does not apply to is skipped, and a failure is logged.
func startReview(ctx context.Context, profile *config.Profile, stores *store.Store, b *bus.Bus, issue *store.IssueMessage) {
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
	if _, err := stores.CreateReviewRun(ctx, issue.ProjectID, issue.UID); err != nil {
		slog.Error("failed to start the review",
			slog.String("project", issue.ProjectID),
			slog.Int64("issue_id", issue.UID),
			log.BBError(err))
		return
	}
	b.ReviewRunTickleChan <- 0
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

func convertToReviewRun(reviewRun *store.ReviewRunMessage) *v1pb.ReviewRun {
	converted := &v1pb.ReviewRun{
		Name:       common.FormatReviewRun(reviewRun.ProjectID, reviewRun.IssueUID),
		State:      convertToReviewRunState(reviewRun.Status),
		CreateTime: timestamppb.New(reviewRun.CreatedAt),
	}
	switch reviewRun.Status {
	case storepb.ReviewRun_DONE, storepb.ReviewRun_FAILED:
		// A terminal row is never written again, so updated_at is the end
		// time.
		converted.EndTime = timestamppb.New(reviewRun.UpdatedAt)
	default:
	}
	if reviewRun.Status == storepb.ReviewRun_FAILED {
		// A failure with no more specific code stored is UNKNOWN, never OK.
		failure := reviewRun.Payload.GetCode()
		if failure == code.Code_OK {
			failure = code.Code_UNKNOWN
		}
		converted.Error = &spb.Status{
			Code:    int32(failure),
			Message: reviewRun.Payload.GetError(),
		}
	}
	return converted
}

func convertToReviewRunState(status storepb.ReviewRun_Status) v1pb.ReviewRun_State {
	switch status {
	case storepb.ReviewRun_AVAILABLE, storepb.ReviewRun_RUNNING:
		return v1pb.ReviewRun_RUNNING
	case storepb.ReviewRun_DONE:
		return v1pb.ReviewRun_SUCCEEDED
	case storepb.ReviewRun_FAILED:
		return v1pb.ReviewRun_FAILED
	default:
		return v1pb.ReviewRun_STATE_UNSPECIFIED
	}
}
