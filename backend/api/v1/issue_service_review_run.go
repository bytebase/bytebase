package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// Review run resource IDs: the {reviewRun} segment of
// projects/{project}/issues/{issue}/reviewRuns/{reviewRun}.
const (
	reviewRunIDRule      = "rule"
	reviewRunIDGuideline = "guideline"
)

// RunReview triggers a review run. The slot is reset unconditionally: a
// RUNNING execution is superseded, the attempt number is bumped, and the
// returned run is AVAILABLE.
func (s *IssueService) RunReview(ctx context.Context, req *connect.Request[v1pb.RunReviewRequest]) (*connect.Response[v1pb.ReviewRun], error) {
	projectID, issueUID, reviewRunID, err := common.GetProjectIDIssueUIDReviewRunID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reviewType, ok := reviewRunTypeFromID(reviewRunID)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unknown reviewer type %q", reviewRunID))
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
	if issue.Status != storepb.Issue_OPEN {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("review runs only on an open issue; issue is %s", issue.Status))
	}
	if issue.PlanUID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("issue %d has no SQL to review", issueUID))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: issue.PlanUID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", *issue.PlanUID, projectID))
	}
	if plan.Config.GetHasRollout() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("the plan already has a rollout; its SQL is frozen"))
	}

	// Applicability: the review type must apply to this issue right now — a
	// slot for an inapplicable type must not exist, so it is not created.
	project, err := s.store.GetProjectByResourceID(ctx, projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", projectID))
	}
	databaseGroup, err := plancheck.GetDatabaseGroupForPlan(ctx, s.store, plan, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database group for plan"))
	}
	targets, err := plancheck.DeriveReviewTargets(ctx, s.store, project, plan, databaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to derive review targets"))
	}
	if len(targets) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("the plan has no reviewable SQL"))
	}
	if reviewType == store.ReviewRunTypeGuideline {
		aiSetting, err := s.store.GetAISetting(ctx, common.GetWorkspaceIDFromContext(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get AI setting"))
		}
		if !aiSetting.Enabled {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("guideline review requires AI to be enabled"))
		}
	}

	reviewRun, err := s.store.CreateReviewRun(ctx, projectID, issueUID, reviewType)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create review run"))
	}

	s.bus.ReviewRunTickleChan <- 0

	return connect.NewResponse(convertToReviewRun(reviewRun)), nil
}

// reviewRunTypeFromID maps a review run resource ID to the stored reviewer
// type.
func reviewRunTypeFromID(reviewRunID string) (string, bool) {
	switch reviewRunID {
	case reviewRunIDRule:
		return store.ReviewRunTypeRule, true
	case reviewRunIDGuideline:
		return store.ReviewRunTypeGuideline, true
	default:
		return "", false
	}
}

// reviewRunIDFromType maps a stored reviewer type to its resource ID. An
// unknown type maps to itself: the reviewer-id space is open, and a name
// beats an empty segment.
func reviewRunIDFromType(reviewType string) string {
	switch reviewType {
	case store.ReviewRunTypeRule:
		return reviewRunIDRule
	case store.ReviewRunTypeGuideline:
		return reviewRunIDGuideline
	default:
		return reviewType
	}
}

func convertToReviewRun(reviewRun *store.ReviewRunMessage) *v1pb.ReviewRun {
	converted := &v1pb.ReviewRun{
		Name:       common.FormatReviewRun(reviewRun.ProjectID, reviewRun.IssueUID, reviewRunIDFromType(reviewRun.Type)),
		Type:       convertToReviewRunType(reviewRun.Type),
		Status:     convertToReviewRunStatus(reviewRun.Status),
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
		converted.Error = reviewRun.Payload.GetError()
	}
	return converted
}

func convertToReviewRunType(reviewType string) v1pb.ReviewRun_Type {
	switch reviewType {
	case store.ReviewRunTypeRule:
		return v1pb.ReviewRun_RULE
	case store.ReviewRunTypeGuideline:
		return v1pb.ReviewRun_GUIDELINE
	default:
		return v1pb.ReviewRun_TYPE_UNSPECIFIED
	}
}

func convertToReviewRunStatus(status storepb.ReviewRun_Status) v1pb.ReviewRun_Status {
	switch status {
	case storepb.ReviewRun_AVAILABLE:
		return v1pb.ReviewRun_AVAILABLE
	case storepb.ReviewRun_RUNNING:
		return v1pb.ReviewRun_RUNNING
	case storepb.ReviewRun_DONE:
		return v1pb.ReviewRun_DONE
	case storepb.ReviewRun_FAILED:
		return v1pb.ReviewRun_FAILED
	default:
		return v1pb.ReviewRun_STATUS_UNSPECIFIED
	}
}
