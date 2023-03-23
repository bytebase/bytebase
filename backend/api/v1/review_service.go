package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ReviewService implements the review service.
type ReviewService struct {
	v1pb.UnimplementedReviewServiceServer
	store *store.Store
}

// NewReviewService creates a new ReviewService.
func NewReviewService(store *store.Store) *ReviewService {
	return &ReviewService{
		store: store,
	}
}

// GetReview gets a review.
// Currently, only review.ApprovalTemplates and review.Approvers are set.
func (s *ReviewService) GetReview(ctx context.Context, request *v1pb.GetReviewRequest) (*v1pb.Review, error) {
	reviewID, err := getReviewID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &reviewID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, error: %v", err)
	}
	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

// ApproveReview approves the approval flow of the review.
func (*ReviewService) ApproveReview(context.Context, *v1pb.ApproveReviewRequest) (*v1pb.Review, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApproveReview not implemented")
}

func convertToReview(ctx context.Context, store *store.Store, issue *store.IssueMessage) (*v1pb.Review, error) {
	issuePayload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), issuePayload); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal issue payload")
	}

	review := &v1pb.Review{}
	if issuePayload.Approval != nil {
		for _, template := range issuePayload.Approval.ApprovalTemplates {
			review.ApprovalTemplates = append(review.ApprovalTemplates, convertToApprovalTemplate(template))
		}
		for _, approver := range issuePayload.Approval.Approvers {
			convertedApprover := &v1pb.Review_Approver{Status: v1pb.Review_Approver_Status(approver.Status)}
			user, err := store.GetUserByID(ctx, int(approver.PrincipalId))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find user by id %v", approver.PrincipalId)
			}
			convertedApprover.Principal = fmt.Sprintf("user:%s", user.Email)

			review.Approvers = append(review.Approvers, convertedApprover)
		}
	}

	return review, nil
}

func convertToApprovalTemplate(template *storepb.ApprovalTemplate) *v1pb.ApprovalTemplate {
	return &v1pb.ApprovalTemplate{
		Flow:        convertToApprovalFlow(template.Flow),
		Title:       template.Title,
		Description: template.Description,
	}
}

func convertToApprovalFlow(flow *storepb.ApprovalFlow) *v1pb.ApprovalFlow {
	convertedFlow := &v1pb.ApprovalFlow{}
	for _, step := range flow.Steps {
		convertedFlow.Steps = append(convertedFlow.Steps, convertToApprovalStep(step))
	}
	return convertedFlow
}

func convertToApprovalStep(step *storepb.ApprovalStep) *v1pb.ApprovalStep {
	convertedStep := &v1pb.ApprovalStep{
		Type: v1pb.ApprovalStep_Type(step.Type),
	}
	for _, node := range step.Nodes {
		convertedStep.Nodes = append(convertedStep.Nodes, convertToApprovalNode(node))
	}
	return convertedStep
}

func convertToApprovalNode(node *storepb.ApprovalNode) *v1pb.ApprovalNode {
	switch node.Payload.(type) {
	case *storepb.ApprovalNode_GroupValue_:
		return &v1pb.ApprovalNode{
			Type: v1pb.ApprovalNode_ANY_IN_GROUP,
			Payload: &v1pb.ApprovalNode_GroupValue_{
				GroupValue: v1pb.ApprovalNode_GroupValue(node.Payload.(*storepb.ApprovalNode_GroupValue_).GroupValue)},
		}
	}
	return &v1pb.ApprovalNode{}
}
