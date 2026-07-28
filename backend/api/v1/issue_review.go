package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/review"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// ApproveIssue approves the approval flow of the issue.
func (s *IssueService) ApproveIssue(ctx context.Context, req *connect.Request[v1pb.ApproveIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	return s.reviewIssue(ctx, req.Msg.Name, req.Msg.Comment, review.ActionApprove)
}

// RejectIssue rejects an issue.
func (s *IssueService) RejectIssue(ctx context.Context, req *connect.Request[v1pb.RejectIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	return s.reviewIssue(ctx, req.Msg.Name, req.Msg.Comment, review.ActionReject)
}

// RequestIssue requests an issue after rejection.
func (s *IssueService) RequestIssue(ctx context.Context, req *connect.Request[v1pb.RequestIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	return s.reviewIssue(ctx, req.Msg.Name, req.Msg.Comment, review.ActionRequest)
}

func (s *IssueService) reviewIssue(ctx context.Context, name, comment string, action review.Action) (*connect.Response[v1pb.Issue], error) {
	projectID, issueUID, err := common.GetProjectIDIssueUID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}
	result, err := s.reviewWorkflow.ReviewIssue(ctx, review.IssueInput{
		Workspace: common.GetWorkspaceIDFromContext(ctx),
		ProjectID: projectID,
		IssueUID:  issueUID,
		Actor:     user,
		Action:    action,
		Comment:   comment,
	})
	if err != nil {
		return nil, mapReviewError(err, action)
	}

	createRollout := false
	for _, event := range result.Events {
		switch event := event.(type) {
		case review.IssueCommentEvent:
			if _, err := s.store.CreateIssueComments(ctx, event.ActorEmail, &store.IssueCommentMessage{
				ProjectID: result.Issue.ProjectID,
				IssueUID:  result.Issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Comment: event.Comment,
					Event: &storepb.IssueCommentPayload_Approval_{
						Approval: &storepb.IssueCommentPayload_Approval{Status: event.ApprovalStatus},
					},
				},
			}); err != nil {
				slog.Warn("failed to create issue comment", log.BBError(err))
			}
		case review.ApprovalRequestedEvent:
			review.NotifyApprovalRequested(ctx, s.store, s.webhookManager, result.Issue, result.Project)
		case review.IssueApprovedEvent:
			review.NotifyIssueApproved(ctx, s.store, s.webhookManager, result.Issue, result.Project, user)
		case review.IssueSentBackEvent:
			creator, err := s.store.GetAccountByEmail(ctx, result.Issue.CreatorEmail)
			if err != nil {
				slog.Warn("failed to get issue creator", log.BBError(err))
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get issue creator"))
			}
			s.webhookManager.CreateEvent(ctx, &webhook.Event{
				Type:    storepb.Activity_ISSUE_SENT_BACK,
				Project: webhook.NewProject(result.Project),
				SentBack: &webhook.EventIssueSentBack{
					Approver: &webhook.User{Name: user.Name, Email: user.Email, Phone: user.Phone},
					Creator:  &webhook.User{Name: creator.Name, Email: creator.Email, Phone: creator.Phone},
					Issue:    webhook.NewIssue(result.Issue),
					Reason:   comment,
				},
			})
		case review.CompleteAccessRequestEvent:
			completed, err := completeAccessRequestIssue(ctx, s.store, user.Email, result.Issue)
			if err != nil {
				slog.Debug("failed to complete role grant issue", log.BBError(err))
			} else {
				result.Issue = completed
			}
		case review.CreateRolloutEvent:
			createRollout = true
		default:
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("unexpected review event %T", event))
		}
	}

	converted, err := s.convertToIssue(result.Issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to convert to issue"))
	}
	if createRollout && result.Issue.PlanUID != nil {
		s.bus.RolloutCreationChan <- bus.PlanRef{ProjectID: result.Issue.ProjectID, PlanID: *result.Issue.PlanUID}
	}
	return connect.NewResponse(converted), nil
}

func mapReviewError(err error, action review.Action) error {
	var workflowErr *review.Error
	if !errors.As(err, &workflowErr) {
		return connect.NewError(connect.CodeInternal, err)
	}
	switch workflowErr.Code {
	case review.ErrorNotFound:
		return connect.NewError(connect.CodeNotFound, workflowErr)
	case review.ErrorInvalidAction:
		return connect.NewError(connect.CodeInvalidArgument, workflowErr)
	case review.ErrorFailedPrecondition:
		return connect.NewError(connect.CodeFailedPrecondition, workflowErr)
	case review.ErrorPermissionDenied:
		return connect.NewError(connect.CodePermissionDenied, workflowErr)
	case review.ErrorConflict:
		message := "cannot request issue because approval finding is stale"
		switch action {
		case review.ActionApprove:
			message = "cannot approve because approval finding is stale"
		case review.ActionReject:
			message = "cannot reject because approval finding is stale"
		case review.ActionRequest:
		default:
			return connect.NewError(connect.CodeInternal, errors.Errorf("unexpected review action %d", action))
		}
		return connect.NewError(connect.CodeFailedPrecondition, errors.New(message))
	default:
		return connect.NewError(connect.CodeInternal, workflowErr)
	}
}
